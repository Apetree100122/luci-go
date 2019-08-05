// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/internal/encoding/json"
	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/flags"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/internal/set"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Unmarshal reads the given []byte into the given proto.Message.
func Unmarshal(b []byte, m proto.Message) error {
	return UnmarshalOptions{}.Unmarshal(b, m)
}

// UnmarshalOptions is a configurable JSON format parser.
type UnmarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// If AllowPartial is set, input for messages that will result in missing
	// required fields will not return an error.
	AllowPartial bool

	// If DiscardUnknown is set, unknown fields are ignored.
	DiscardUnknown bool

	// Resolver is used for looking up types when unmarshaling
	// google.protobuf.Any messages or extension fields.
	// If nil, this defaults to using protoregistry.GlobalTypes.
	Resolver interface {
		protoregistry.MessageTypeResolver
		protoregistry.ExtensionTypeResolver
	}

	decoder *json.Decoder
}

// Unmarshal reads the given []byte and populates the given proto.Message using
// options in UnmarshalOptions object. It will clear the message first before
// setting the fields. If it returns an error, the given message may be
// partially set.
func (o UnmarshalOptions) Unmarshal(b []byte, m proto.Message) error {
	// TODO: Determine if we would like to have an option for merging or only
	// have merging behavior. We should at least be consistent with textproto
	// marshaling.
	proto.Reset(m)

	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}
	o.decoder = json.NewDecoder(b)

	if err := o.unmarshalMessage(m.ProtoReflect(), false); err != nil {
		return err
	}

	// Check for EOF.
	val, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if val.Type() != json.EOF {
		return unexpectedJSONError{val}
	}

	if o.AllowPartial {
		return nil
	}
	return proto.IsInitialized(m)
}

// unexpectedJSONError is an error that contains the unexpected json.Value. This
// is returned by methods to provide callers the read json.Value that it did not
// expect.
// TODO: Consider moving this to internal/encoding/json for consistency with
// errors that package returns.
type unexpectedJSONError struct {
	value json.Value
}

func (e unexpectedJSONError) Error() string {
	return newError("unexpected value %s", e.value).Error()
}

// newError returns an error object. If one of the values passed in is of
// json.Value type, it produces an error with position info.
func newError(f string, x ...interface{}) error {
	var hasValue bool
	var line, column int
	for i := 0; i < len(x); i++ {
		if val, ok := x[i].(json.Value); ok {
			line, column = val.Position()
			hasValue = true
			break
		}
	}
	e := errors.New(f, x...)
	if hasValue {
		return errors.New("(line %d:%d): %v", line, column, e)
	}
	return e
}

// unmarshalMessage unmarshals a message into the given protoreflect.Message.
func (o UnmarshalOptions) unmarshalMessage(m pref.Message, skipTypeURL bool) error {
	if isCustomType(m.Descriptor().FullName()) {
		return o.unmarshalCustomType(m)
	}

	jval, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.StartObject {
		return unexpectedJSONError{jval}
	}

	if err := o.unmarshalFields(m, skipTypeURL); err != nil {
		return err
	}

	return nil
}

// unmarshalFields unmarshals the fields into the given protoreflect.Message.
func (o UnmarshalOptions) unmarshalFields(m pref.Message, skipTypeURL bool) error {
	messageDesc := m.Descriptor()
	if !flags.Proto1Legacy && messageset.IsMessageSet(messageDesc) {
		return errors.New("no support for proto1 MessageSets")
	}

	var seenNums set.Ints
	var seenOneofs set.Ints
	fieldDescs := messageDesc.Fields()
	for {
		// Read field name.
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		switch jval.Type() {
		default:
			return unexpectedJSONError{jval}
		case json.EndObject:
			return nil
		case json.Name:
			// Continue below.
		}

		name, err := jval.Name()
		if err != nil {
			return err
		}
		// Unmarshaling a non-custom embedded message in Any will contain the
		// JSON field "@type" which should be skipped because it is not a field
		// of the embedded message, but simply an artifact of the Any format.
		if skipTypeURL && name == "@type" {
			o.decoder.Read()
			continue
		}

		// Get the FieldDescriptor.
		var fd pref.FieldDescriptor
		if strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") {
			// Only extension names are in [name] format.
			extName := pref.FullName(name[1 : len(name)-1])
			extType, err := o.findExtension(extName)
			if err != nil && err != protoregistry.NotFound {
				return errors.New("unable to resolve [%v]: %v", extName, err)
			}
			fd = extType
		} else {
			// The name can either be the JSON name or the proto field name.
			fd = fieldDescs.ByJSONName(name)
			if fd == nil {
				fd = fieldDescs.ByName(pref.Name(name))
			}
			if fd != nil && fd.IsWeak() && fd.Message().IsPlaceholder() {
				fd = nil // reset since the weak reference is not linked in
			}
		}

		if fd == nil {
			// Field is unknown.
			if o.DiscardUnknown {
				if err := skipJSONValue(o.decoder); err != nil {
					return err
				}
				continue
			}
			return newError("%v contains unknown field %s", messageDesc.FullName(), jval)
		}

		// Do not allow duplicate fields.
		num := uint64(fd.Number())
		if seenNums.Has(num) {
			return newError("%v contains repeated field %s", messageDesc.FullName(), jval)
		}
		seenNums.Set(num)

		// No need to set values for JSON null unless the field type is
		// google.protobuf.Value or google.protobuf.NullValue.
		if o.decoder.Peek() == json.Null && !isKnownValue(fd) && !isNullValue(fd) {
			o.decoder.Read()
			continue
		}

		switch {
		case fd.IsList():
			list := m.Mutable(fd).List()
			if err := o.unmarshalList(list, fd); err != nil {
				return errors.New("%v|%q: %v", fd.FullName(), name, err)
			}
		case fd.IsMap():
			mmap := m.Mutable(fd).Map()
			if err := o.unmarshalMap(mmap, fd); err != nil {
				return errors.New("%v|%q: %v", fd.FullName(), name, err)
			}
		default:
			// If field is a oneof, check if it has already been set.
			if od := fd.ContainingOneof(); od != nil {
				idx := uint64(od.Index())
				if seenOneofs.Has(idx) {
					return errors.New("%v: oneof is already set", od.FullName())
				}
				seenOneofs.Set(idx)
			}

			// Required or optional fields.
			if err := o.unmarshalSingular(m, fd); err != nil {
				return errors.New("%v|%q: %v", fd.FullName(), name, err)
			}
		}
	}
}

// findExtension returns protoreflect.ExtensionType from the resolver if found.
func (o UnmarshalOptions) findExtension(xtName pref.FullName) (pref.ExtensionType, error) {
	xt, err := o.Resolver.FindExtensionByName(xtName)
	if err == nil {
		return xt, nil
	}
	return messageset.FindMessageSetExtension(o.Resolver, xtName)
}

func isKnownValue(fd pref.FieldDescriptor) bool {
	md := fd.Message()
	return md != nil && md.FullName() == "google.protobuf.Value"
}

func isNullValue(fd pref.FieldDescriptor) bool {
	ed := fd.Enum()
	return ed != nil && ed.FullName() == "google.protobuf.NullValue"
}

// unmarshalSingular unmarshals to the non-repeated field specified by the given
// FieldDescriptor.
func (o UnmarshalOptions) unmarshalSingular(m pref.Message, fd pref.FieldDescriptor) error {
	var val pref.Value
	var err error
	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		m2 := m.NewMessage(fd)
		err = o.unmarshalMessage(m2, false)
		val = pref.ValueOf(m2)
	default:
		val, err = o.unmarshalScalar(fd)
	}

	if err != nil {
		return err
	}
	m.Set(fd, val)
	return nil
}

// unmarshalScalar unmarshals to a scalar/enum protoreflect.Value specified by
// the given FieldDescriptor.
func (o UnmarshalOptions) unmarshalScalar(fd pref.FieldDescriptor) (pref.Value, error) {
	const b32 int = 32
	const b64 int = 64

	jval, err := o.decoder.Read()
	if err != nil {
		return pref.Value{}, err
	}

	kind := fd.Kind()
	switch kind {
	case pref.BoolKind:
		return unmarshalBool(jval)

	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		return unmarshalInt(jval, b32)

	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		return unmarshalInt(jval, b64)

	case pref.Uint32Kind, pref.Fixed32Kind:
		return unmarshalUint(jval, b32)

	case pref.Uint64Kind, pref.Fixed64Kind:
		return unmarshalUint(jval, b64)

	case pref.FloatKind:
		return unmarshalFloat(jval, b32)

	case pref.DoubleKind:
		return unmarshalFloat(jval, b64)

	case pref.StringKind:
		pval, err := unmarshalString(jval)
		if err != nil {
			return pval, err
		}
		return pval, nil

	case pref.BytesKind:
		return unmarshalBytes(jval)

	case pref.EnumKind:
		return unmarshalEnum(jval, fd)
	}

	panic(fmt.Sprintf("invalid scalar kind %v", kind))
}

func unmarshalBool(jval json.Value) (pref.Value, error) {
	if jval.Type() != json.Bool {
		return pref.Value{}, unexpectedJSONError{jval}
	}
	b, err := jval.Bool()
	return pref.ValueOf(b), err
}

func unmarshalInt(jval json.Value, bitSize int) (pref.Value, error) {
	switch jval.Type() {
	case json.Number:
		return getInt(jval, bitSize)

	case json.String:
		// Decode number from string.
		s := strings.TrimSpace(jval.String())
		if len(s) != len(jval.String()) {
			return pref.Value{}, errors.New("invalid number %v", jval.Raw())
		}
		dec := json.NewDecoder([]byte(s))
		jval, err := dec.Read()
		if err != nil {
			return pref.Value{}, err
		}
		return getInt(jval, bitSize)
	}
	return pref.Value{}, unexpectedJSONError{jval}
}

func getInt(jval json.Value, bitSize int) (pref.Value, error) {
	n, err := jval.Int(bitSize)
	if err != nil {
		return pref.Value{}, err
	}
	if bitSize == 32 {
		return pref.ValueOf(int32(n)), nil
	}
	return pref.ValueOf(n), nil
}

func unmarshalUint(jval json.Value, bitSize int) (pref.Value, error) {
	switch jval.Type() {
	case json.Number:
		return getUint(jval, bitSize)

	case json.String:
		// Decode number from string.
		s := strings.TrimSpace(jval.String())
		if len(s) != len(jval.String()) {
			return pref.Value{}, errors.New("invalid number %v", jval.Raw())
		}
		dec := json.NewDecoder([]byte(s))
		jval, err := dec.Read()
		if err != nil {
			return pref.Value{}, err
		}
		return getUint(jval, bitSize)
	}
	return pref.Value{}, unexpectedJSONError{jval}
}

func getUint(jval json.Value, bitSize int) (pref.Value, error) {
	n, err := jval.Uint(bitSize)
	if err != nil {
		return pref.Value{}, err
	}
	if bitSize == 32 {
		return pref.ValueOf(uint32(n)), nil
	}
	return pref.ValueOf(n), nil
}

func unmarshalFloat(jval json.Value, bitSize int) (pref.Value, error) {
	switch jval.Type() {
	case json.Number:
		return getFloat(jval, bitSize)

	case json.String:
		s := jval.String()
		switch s {
		case "NaN":
			if bitSize == 32 {
				return pref.ValueOf(float32(math.NaN())), nil
			}
			return pref.ValueOf(math.NaN()), nil
		case "Infinity":
			if bitSize == 32 {
				return pref.ValueOf(float32(math.Inf(+1))), nil
			}
			return pref.ValueOf(math.Inf(+1)), nil
		case "-Infinity":
			if bitSize == 32 {
				return pref.ValueOf(float32(math.Inf(-1))), nil
			}
			return pref.ValueOf(math.Inf(-1)), nil
		}
		// Decode number from string.
		if len(s) != len(strings.TrimSpace(s)) {
			return pref.Value{}, errors.New("invalid number %v", jval.Raw())
		}
		dec := json.NewDecoder([]byte(s))
		jval, err := dec.Read()
		if err != nil {
			return pref.Value{}, err
		}
		return getFloat(jval, bitSize)
	}
	return pref.Value{}, unexpectedJSONError{jval}
}

func getFloat(jval json.Value, bitSize int) (pref.Value, error) {
	n, err := jval.Float(bitSize)
	if err != nil {
		return pref.Value{}, err
	}
	if bitSize == 32 {
		return pref.ValueOf(float32(n)), nil
	}
	return pref.ValueOf(n), nil
}

func unmarshalString(jval json.Value) (pref.Value, error) {
	if jval.Type() != json.String {
		return pref.Value{}, unexpectedJSONError{jval}
	}
	return pref.ValueOf(jval.String()), nil
}

func unmarshalBytes(jval json.Value) (pref.Value, error) {
	if jval.Type() != json.String {
		return pref.Value{}, unexpectedJSONError{jval}
	}

	s := jval.String()
	enc := base64.StdEncoding
	if strings.ContainsAny(s, "-_") {
		enc = base64.URLEncoding
	}
	if len(s)%4 != 0 {
		enc = enc.WithPadding(base64.NoPadding)
	}
	b, err := enc.DecodeString(s)
	if err != nil {
		return pref.Value{}, err
	}
	return pref.ValueOf(b), nil
}

func unmarshalEnum(jval json.Value, fd pref.FieldDescriptor) (pref.Value, error) {
	switch jval.Type() {
	case json.String:
		// Lookup EnumNumber based on name.
		s := jval.String()
		if enumVal := fd.Enum().Values().ByName(pref.Name(s)); enumVal != nil {
			return pref.ValueOf(enumVal.Number()), nil
		}
		return pref.Value{}, newError("invalid enum value %q", jval)

	case json.Number:
		n, err := jval.Int(32)
		if err != nil {
			return pref.Value{}, err
		}
		return pref.ValueOf(pref.EnumNumber(n)), nil

	case json.Null:
		// This is only valid for google.protobuf.NullValue.
		if isNullValue(fd) {
			return pref.ValueOf(pref.EnumNumber(0)), nil
		}
	}

	return pref.Value{}, unexpectedJSONError{jval}
}

func (o UnmarshalOptions) unmarshalList(list pref.List, fd pref.FieldDescriptor) error {
	jval, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.StartArray {
		return unexpectedJSONError{jval}
	}

	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		for {
			m := list.NewMessage()
			err := o.unmarshalMessage(m, false)
			if err != nil {
				if e, ok := err.(unexpectedJSONError); ok {
					if e.value.Type() == json.EndArray {
						// Done with list.
						return nil
					}
				}
				return err
			}
			list.Append(pref.ValueOf(m))
		}
	default:
		for {
			val, err := o.unmarshalScalar(fd)
			if err != nil {
				if e, ok := err.(unexpectedJSONError); ok {
					if e.value.Type() == json.EndArray {
						// Done with list.
						return nil
					}
				}
				return err
			}
			list.Append(val)
		}
	}
	return nil
}

func (o UnmarshalOptions) unmarshalMap(mmap pref.Map, fd pref.FieldDescriptor) error {
	jval, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.StartObject {
		return unexpectedJSONError{jval}
	}

	// Determine ahead whether map entry is a scalar type or a message type in
	// order to call the appropriate unmarshalMapValue func inside the for loop
	// below.
	var unmarshalMapValue func() (pref.Value, error)
	switch fd.MapValue().Kind() {
	case pref.MessageKind, pref.GroupKind:
		unmarshalMapValue = func() (pref.Value, error) {
			m := mmap.NewMessage()
			if err := o.unmarshalMessage(m, false); err != nil {
				return pref.Value{}, err
			}
			return pref.ValueOf(m), nil
		}
	default:
		unmarshalMapValue = func() (pref.Value, error) {
			return o.unmarshalScalar(fd.MapValue())
		}
	}

Loop:
	for {
		// Read field name.
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		switch jval.Type() {
		default:
			return unexpectedJSONError{jval}
		case json.EndObject:
			break Loop
		case json.Name:
			// Continue.
		}

		name, err := jval.Name()
		if err != nil {
			return err
		}

		// Unmarshal field name.
		pkey, err := unmarshalMapKey(name, fd.MapKey())
		if err != nil {
			return err
		}

		// Check for duplicate field name.
		if mmap.Has(pkey) {
			return newError("duplicate map key %q", jval)
		}

		// Read and unmarshal field value.
		pval, err := unmarshalMapValue()
		if err != nil {
			return err
		}

		mmap.Set(pkey, pval)
	}

	return nil
}

// unmarshalMapKey converts given string into a protoreflect.MapKey. A map key type is any
// integral or string type.
func unmarshalMapKey(name string, fd pref.FieldDescriptor) (pref.MapKey, error) {
	const b32 = 32
	const b64 = 64
	const base10 = 10

	kind := fd.Kind()
	switch kind {
	case pref.StringKind:
		return pref.ValueOf(name).MapKey(), nil

	case pref.BoolKind:
		switch name {
		case "true":
			return pref.ValueOf(true).MapKey(), nil
		case "false":
			return pref.ValueOf(false).MapKey(), nil
		}
		return pref.MapKey{}, errors.New("invalid value for boolean key %q", name)

	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		n, err := strconv.ParseInt(name, base10, b32)
		if err != nil {
			return pref.MapKey{}, err
		}
		return pref.ValueOf(int32(n)).MapKey(), nil

	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		n, err := strconv.ParseInt(name, base10, b64)
		if err != nil {
			return pref.MapKey{}, err
		}
		return pref.ValueOf(int64(n)).MapKey(), nil

	case pref.Uint32Kind, pref.Fixed32Kind:
		n, err := strconv.ParseUint(name, base10, b32)
		if err != nil {
			return pref.MapKey{}, err
		}
		return pref.ValueOf(uint32(n)).MapKey(), nil

	case pref.Uint64Kind, pref.Fixed64Kind:
		n, err := strconv.ParseUint(name, base10, b64)
		if err != nil {
			return pref.MapKey{}, err
		}
		return pref.ValueOf(uint64(n)).MapKey(), nil
	}

	panic(fmt.Sprintf("%s: invalid kind %s for map key", fd.FullName(), kind))
}
