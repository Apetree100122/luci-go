// Copyright 2015 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serialize

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"go.chromium.org/luci/common/data/cmpbin"
	"go.chromium.org/luci/common/data/stringset"
	ds "go.chromium.org/luci/gae/service/datastore"
)

// WritePropertyMapDeterministic allows tests to make Serializer.PropertyMap
// deterministic.
//
// This should be set once at the top of your tests in an init() function.
var WritePropertyMapDeterministic = false

// Serializer allows writing binary-encoded datastore types (like Properties,
// Keys, etc.)
//
// See the `Serialize` and `SerializeKC` variables for common shortcuts.
type Serializer struct {
	// If true, WithKeyContext controls whether bytes written with this Serializer
	// include the Key's appid and namespace.
	//
	// Frequently the appid and namespace of keys are known in advance and so
	// there's no reason to redundantly encode them.
	WithKeyContext bool
}

var (
	// Serialize is a Serializer{WithKeyContext:false}, useful for inline
	// invocations like:
	//
	//   datastore.Serialize.Time(...)
	Serialize Serializer

	// SerializeKC is a Serializer{WithKeyContext:true}, useful for inline
	// invocations like:
	//
	//   datastore.SerializeKC.Key(...)
	SerializeKC = Serializer{true}
)

// Key encodes a key to the buffer. If context is WithContext, then this
// encoded value will include the appid and namespace of the key.
func (s Serializer) Key(buf cmpbin.WriteableBytesBuffer, k *ds.Key) (err error) {
	// [appid ++ namespace]? ++ [1 ++ token]* ++ NULL
	defer recoverTo(&err)
	appid, namespace, toks := k.Split()
	if s.WithKeyContext {
		panicIf(buf.WriteByte(1))
		_, e := cmpbin.WriteString(buf, appid)
		panicIf(e)
		_, e = cmpbin.WriteString(buf, namespace)
		panicIf(e)
	} else {
		panicIf(buf.WriteByte(0))
	}
	for _, tok := range toks {
		panicIf(buf.WriteByte(1))
		panicIf(s.KeyTok(buf, tok))
	}
	return buf.WriteByte(0)
}

// KeyTok writes a KeyTok to the buffer. You usually want Key instead of this.
func (s Serializer) KeyTok(buf cmpbin.WriteableBytesBuffer, tok ds.KeyTok) (err error) {
	// tok.kind ++ typ ++ [tok.stringID || tok.intID]
	defer recoverTo(&err)
	_, e := cmpbin.WriteString(buf, tok.Kind)
	panicIf(e)
	if tok.StringID != "" {
		panicIf(buf.WriteByte(byte(ds.PTString)))
		_, e := cmpbin.WriteString(buf, tok.StringID)
		panicIf(e)
	} else {
		panicIf(buf.WriteByte(byte(ds.PTInt)))
		_, e := cmpbin.WriteInt(buf, tok.IntID)
		panicIf(e)
	}
	return nil
}

// GeoPoint writes a GeoPoint to the buffer.
func (s Serializer) GeoPoint(buf cmpbin.WriteableBytesBuffer, gp ds.GeoPoint) (err error) {
	defer recoverTo(&err)
	_, e := cmpbin.WriteFloat64(buf, gp.Lat)
	panicIf(e)
	_, e = cmpbin.WriteFloat64(buf, gp.Lng)
	return e
}

// Time writes a time.Time to the buffer.
//
// The supplied time is rounded via datastore.RoundTime and written as a
// microseconds-since-epoch integer to comform to datastore storage standards.
func (s Serializer) Time(buf cmpbin.WriteableBytesBuffer, t time.Time) error {
	name, off := t.Zone()
	if name != "UTC" || off != 0 {
		panic(fmt.Errorf("helper: UTC OR DEATH: %s", t))
	}

	_, err := cmpbin.WriteInt(buf, ds.TimeToInt(t))
	return err
}

// Property writes a Property to the buffer. `context` behaves the same
// way that it does for WriteKey, but only has an effect if `p` contains a
// Key as its IndexValue.
func (s Serializer) Property(buf cmpbin.WriteableBytesBuffer, p ds.Property) error {
	return s.propertyImpl(buf, &p, false)
}

// IndexProperty writes a Property to the buffer as its native index type.
// `context` behaves the same way that it does for WriteKey, but only has an
// effect if `p` contains a Key as its IndexValue.
func (s Serializer) IndexProperty(buf cmpbin.WriteableBytesBuffer, p ds.Property) error {
	return s.propertyImpl(buf, &p, true)
}

// propertyImpl is an implementation of WriteProperty and
// WriteIndexProperty.
func (s Serializer) propertyImpl(buf cmpbin.WriteableBytesBuffer, p *ds.Property, index bool) (err error) {
	defer recoverTo(&err)

	it, v := p.IndexTypeAndValue()
	if !index {
		it = p.Type()
	}
	typb := byte(it)
	if p.IndexSetting() != ds.NoIndex {
		typb |= 0x80
	}
	panicIf(buf.WriteByte(typb))

	err = s.indexValue(buf, v)
	return
}

// indexValue writes the index value of v to buf.
//
// v may be one of the return types from ds.Property's GetIndexTypeAndValue
// method.
func (s Serializer) indexValue(buf cmpbin.WriteableBytesBuffer, v interface{}) (err error) {
	switch t := v.(type) {
	case nil:
	case bool:
		b := byte(0)
		if t {
			b = 1
		}
		err = buf.WriteByte(b)
	case int64:
		_, err = cmpbin.WriteInt(buf, t)
	case float64:
		_, err = cmpbin.WriteFloat64(buf, t)
	case string:
		_, err = cmpbin.WriteString(buf, t)
	case []byte:
		_, err = cmpbin.WriteBytes(buf, t)
	case ds.GeoPoint:
		err = s.GeoPoint(buf, t)
	case ds.PropertyMap:
		err = s.PropertyMap(buf, t)
	case *ds.Key:
		err = s.Key(buf, t)

	default:
		err = fmt.Errorf("unsupported type: %T", t)
	}
	return
}

// PropertyMap writes an entire PropertyMap to the buffer. `context`
// behaves the same way that it does for WriteKey.
//
// If WritePropertyMapDeterministic is true, then the rows will be sorted by
// property name before they're serialized to buf (mostly useful for testing,
// but also potentially useful if you need to make a hash of the property data).
//
// Write skips metadata keys.
func (s Serializer) PropertyMap(buf cmpbin.WriteableBytesBuffer, pm ds.PropertyMap) (err error) {
	defer recoverTo(&err)
	rows := make(sort.StringSlice, 0, len(pm))
	tmpBuf := &bytes.Buffer{}
	pm, _ = pm.Save(false)
	for name, pdata := range pm {
		tmpBuf.Reset()
		_, e := cmpbin.WriteString(tmpBuf, name)
		panicIf(e)

		switch t := pdata.(type) {
		case ds.Property:
			_, e = cmpbin.WriteInt(tmpBuf, -1)
			panicIf(e)
			panicIf(s.Property(tmpBuf, t))

		case ds.PropertySlice:
			_, e = cmpbin.WriteInt(tmpBuf, int64(len(t)))
			panicIf(e)
			for _, p := range t {
				panicIf(s.Property(tmpBuf, p))
			}

		default:
			return fmt.Errorf("unknown PropertyData type %T", t)
		}
		rows = append(rows, tmpBuf.String())
	}

	if WritePropertyMapDeterministic {
		rows.Sort()
	}

	_, e := cmpbin.WriteUint(buf, uint64(len(pm)))
	panicIf(e)
	for _, r := range rows {
		_, e := buf.WriteString(r)
		panicIf(e)
	}
	return
}

// IndexColumn writes an IndexColumn to the buffer.
func (s Serializer) IndexColumn(buf cmpbin.WriteableBytesBuffer, c ds.IndexColumn) (err error) {
	defer recoverTo(&err)

	if !c.Descending {
		panicIf(buf.WriteByte(0))
	} else {
		panicIf(buf.WriteByte(1))
	}
	_, err = cmpbin.WriteString(buf, c.Property)
	return
}

// IndexDefinition writes an IndexDefinition to the buffer
func (s Serializer) IndexDefinition(buf cmpbin.WriteableBytesBuffer, i ds.IndexDefinition) (err error) {
	defer recoverTo(&err)

	_, err = cmpbin.WriteString(buf, i.Kind)
	panicIf(err)
	if !i.Ancestor {
		panicIf(buf.WriteByte(0))
	} else {
		panicIf(buf.WriteByte(1))
	}
	for _, sb := range i.SortBy {
		panicIf(buf.WriteByte(1))
		panicIf(s.IndexColumn(buf, sb))
	}
	return buf.WriteByte(0)
}

// SerializedPslice is all of the serialized DSProperty values in ASC order.
type SerializedPslice [][]byte

func (s SerializedPslice) Len() int           { return len(s) }
func (s SerializedPslice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SerializedPslice) Less(i, j int) bool { return bytes.Compare(s[i], s[j]) < 0 }

// PropertySlicePartially serializes a single row of a DSProperty map.
//
// It does not differentiate between single- and multi- properties.
//
// This is "partial" because it returns something other than `[]byte`
func (s Serializer) PropertySlicePartially(vals ds.PropertySlice) SerializedPslice {
	dups := stringset.New(0)
	ret := make(SerializedPslice, 0, len(vals))
	for _, v := range vals {
		if v.IndexSetting() == ds.NoIndex {
			continue
		}

		data := Serialize.ToBytes(v)
		dataS := string(data)
		if !dups.Add(dataS) {
			continue
		}
		ret = append(ret, data)
	}
	return ret
}

// SerializedPmap maps from
//   prop name -> [<serialized DSProperty>, ...]
// includes special values '__key__' and '__ancestor__' which contains all of
// the ancestor entries for this key.
type SerializedPmap map[string]SerializedPslice

// PropertyMapPartially turns a regular PropertyMap into a SerializedPmap.
// Essentially all the []Property's become SerializedPslice, using cmpbin and
// Serializer's encodings.
//
// This is "partial" because it returns something other than `[]byte`
func (s Serializer) PropertyMapPartially(k *ds.Key, pm ds.PropertyMap) (ret SerializedPmap) {
	ret = make(SerializedPmap, len(pm)+2)
	if k != nil {
		ret["__key__"] = [][]byte{Serialize.ToBytes(ds.MkProperty(k))}
		for k != nil {
			ret["__ancestor__"] = append(ret["__ancestor__"], Serialize.ToBytes(ds.MkProperty(k)))
			k = k.Parent()
		}
	}
	for k := range pm {
		newVals := s.PropertySlicePartially(pm.Slice(k))
		if len(newVals) > 0 {
			ret[k] = newVals
		}
	}
	return
}

// ToBytesErr serializes i to a byte slice, if it's one of the type supported
// by this library, otherwise it returns an error.
//
// Key types will be serialized using the 'WithoutContext' option (e.g. their
// encoded forms will not contain AppID or Namespace).
func (s Serializer) ToBytesErr(i interface{}) (ret []byte, err error) {
	buf := bytes.Buffer{}

	switch t := i.(type) {
	case ds.IndexColumn:
		err = s.IndexColumn(&buf, t)

	case ds.IndexDefinition:
		err = s.IndexDefinition(&buf, t)

	case ds.KeyTok:
		err = s.KeyTok(&buf, t)

	case ds.Property:
		err = s.IndexProperty(&buf, t)

	case ds.PropertyMap:
		err = s.PropertyMap(&buf, t)

	default:
		_, v := ds.MkProperty(i).IndexTypeAndValue()
		err = s.indexValue(&buf, v)
	}

	if err == nil {
		ret = buf.Bytes()
	}
	return
}

// ToBytes serializes i to a byte slice, if it's one of the type supported
// by this library. If an error is encountered (e.g. `i` is not a supported
// type), this method panics.
//
// Key types will be serialized using the 'WithoutContext' option (e.g. their
// encoded forms will not contain AppID or Namespace).
func (s Serializer) ToBytes(i interface{}) []byte {
	ret, err := s.ToBytesErr(i)
	if err != nil {
		panic(err)
	}
	return ret
}

type parseError error

func panicIf(err error) {
	if err != nil {
		panic(parseError(err))
	}
}

func recoverTo(err *error) {
	if r := recover(); r != nil {
		if rerr := r.(parseError); rerr != nil {
			*err = error(rerr)
		}
	}
}
