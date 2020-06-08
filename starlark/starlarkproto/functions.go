// Copyright 2019 The LUCI Authors.
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

package starlarkproto

import (
	"bytes"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/protocolbuffers/txtpbfmt/parser"
)

// ToTextPB serializes a protobuf message to text proto.
func ToTextPB(msg *Message) ([]byte, error) {
	opts := prototext.MarshalOptions{
		AllowPartial: true,
		Indent:       " ",
		Resolver:     msg.typ.loader.types, // used for google.protobuf.Any fields
	}
	blob, err := opts.Marshal(msg.ToProto())
	if err != nil {
		return nil, err
	}
	// prototext randomly injects spaces into the generate output. Pass it through
	// a formatter to get rid of them.
	return parser.FormatWithConfig(blob, parser.Config{
		SkipAllColons: true,
	})
}

// ToJSONPB serializes a protobuf message to JSONPB string.
func ToJSONPB(msg *Message) ([]byte, error) {
	opts := protojson.MarshalOptions{
		AllowPartial: true,
		Resolver:     msg.typ.loader.types, // used for google.protobuf.Any fields
	}
	blob, err := opts.Marshal(msg.ToProto())
	if err != nil {
		return nil, err
	}
	// protojson randomly injects spaces into the generate output. Pass it through
	// a formatter to get rid of them.
	return formatJSON(blob)
}

// ToWirePB serializes a protobuf message to binary wire format.
func ToWirePB(msg *Message) ([]byte, error) {
	opts := proto.MarshalOptions{
		AllowPartial:  true,
		Deterministic: true,
	}
	return opts.Marshal(msg.ToProto())
}

// FromTextPB deserializes a protobuf message given in text proto form.
func FromTextPB(typ *MessageType, blob []byte) (*Message, error) {
	pb := dynamicpb.NewMessage(typ.desc)
	opts := prototext.UnmarshalOptions{
		AllowPartial: true,
		Resolver:     typ.loader.types, // used for google.protobuf.Any fields
	}
	if err := opts.Unmarshal(blob, pb); err != nil {
		return nil, err
	}
	return typ.MessageFromProto(pb), nil
}

// FromJSONPB deserializes a protobuf message given as JBONPB string.
func FromJSONPB(typ *MessageType, blob []byte) (*Message, error) {
	pb := dynamicpb.NewMessage(typ.desc)
	opts := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
		Resolver:       typ.loader.types, // used for google.protobuf.Any fields
	}
	if err := opts.Unmarshal(blob, pb); err != nil {
		return nil, err
	}
	return typ.MessageFromProto(pb), nil
}

// FromWirePB deserializes a protobuf message given as a wire-encoded blob.
func FromWirePB(typ *MessageType, blob []byte) (*Message, error) {
	pb := dynamicpb.NewMessage(typ.desc)
	opts := proto.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
		Resolver:       typ.loader.types, // used for google.protobuf.Any fields
	}
	if err := opts.Unmarshal(blob, pb); err != nil {
		return nil, err
	}
	return typ.MessageFromProto(pb), nil
}

// ProtoLib returns a dict with single struct named "proto" that holds public
// Starlark API for working with proto messages.
//
// Exported functions:
//
//    def new_descriptor_set(name=None, blob=None, deps=None):
//      """Returns a new DescriptorSet.
//
//      Args:
//        name: name of this set for debug and error messages, default is '???'.
//        blob: raw serialized FileDescriptorSet, if any.
//        deps: an iterable of DescriptorSet's with dependencies, if any.
//
//      Returns:
//        New DescriptorSet.
//      """
//
//    def new_loader(*descriptor_sets):
//      """Returns a new proto loader."""
//
//    def default_loader():
//      """Returns a loader used by default when registering descriptor sets."""
//
//    def message_type(msg):
//      """Returns proto.MessageType of the given message."""
//
//    def to_textpb(msg):
//      """Serializes a protobuf message to text proto.
//
//      Args:
//        msg: a *Message to serialize.
//
//      Returns:
//        A str representing msg in text format.
//      """
//
//    def to_jsonpb(msg):
//      """Serializes a protobuf message to JSONPB string.
//
//      Args:
//        msg: a *Message to serialize.
//
//      Returns:
//        A str representing msg in JSONPB format.
//      """
//
//    def to_wirepb(msg):
//      """Serializes a protobuf message to a string using binary wire encoding.
//
//      Args:
//        msg: a *Message to serialize.
//
//      Returns:
//        A str representing msg in binary wire format.
//      """
//
//    def from_textpb(ctor, body):
//      """Deserializes a protobuf message given in text proto form.
//
//      Unknown fields are not allowed.
//
//      Args:
//        ctor: a message constructor function.
//        body: a string with serialized message.
//
//      Returns:
//        Deserialized message constructed via `ctor`.
//      """
//
//    def from_jsonpb(ctor, body):
//      """Deserializes a protobuf message given as JBONPB string.
//
//      Unknown fields are silently skipped.
//
//      Args:
//        ctor: a message constructor function.
//        body: a string with serialized message.
//
//      Returns:
//        Deserialized message constructed via `ctor`.
//      """
//
//    def from_wirepb(ctor, body):
//      """Deserializes a protobuf message given its wire serialization.
//
//      Unknown fields are silently skipped.
//
//      Args:
//        ctor: a message constructor function.
//        body: a string with serialized message.
//
//      Returns:
//        Deserialized message constructed via `ctor`.
//      """
//
//    def struct_to_textpb(s):
//      """Converts a struct to a text proto string.
//
//      Args:
//        s: a struct object. May not contain dicts.
//
//      Returns:
//        A str containing a text format protocol buffer message.
//      """
func ProtoLib() starlark.StringDict {
	return starlark.StringDict{
		"proto": starlarkstruct.FromStringDict(starlark.String("proto"), starlark.StringDict{
			"new_descriptor_set": starlark.NewBuiltin("new_descriptor_set", newDescriptorSet),
			"new_loader":         starlark.NewBuiltin("new_loader", newLoader),
			"default_loader":     starlark.NewBuiltin("default_loader", defaultLoader),
			"message_type":       starlark.NewBuiltin("message_type", messageType),
			"to_textpb":          marshallerBuiltin("to_textpb", ToTextPB),
			"to_jsonpb":          marshallerBuiltin("to_jsonpb", ToJSONPB),
			"to_wirepb":          marshallerBuiltin("to_wirepb", ToWirePB),
			"from_textpb":        unmarshallerBuiltin("from_textpb", FromTextPB),
			"from_jsonpb":        unmarshallerBuiltin("from_jsonpb", FromJSONPB),
			"from_wirepb":        unmarshallerBuiltin("from_wirepb", FromWirePB),
			"struct_to_textpb":   starlark.NewBuiltin("struct_to_textpb", structToTextPb),
		}),
	}
}

// newDescriptorSet constructs *DescriptorSet.
func newDescriptorSet(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	var blob string
	var deps starlark.Value
	err := starlark.UnpackArgs("new_descriptor_set", args, kwargs,
		"name?", &name,
		"blob?", &blob,
		"deps?", &deps,
	)
	if err != nil {
		return nil, err
	}

	// Name is optional.
	if name == "" {
		name = "???"
	}

	// Blob is also optional. If given, it is a serialized FileDescriptorSet.
	var fdps []*descriptorpb.FileDescriptorProto
	if blob != "" {
		fds := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal([]byte(blob), fds); err != nil {
			return nil, fmt.Errorf("new_descriptor_set: for parameter \"blob\": %s", err)
		}
		fdps = fds.GetFile()
	}

	// Collect []*DescriptorSet from 'deps'.
	var sets []*DescriptorSet
	if deps != nil && deps != starlark.None {
		iter := starlark.Iterate(deps)
		if iter == nil {
			return nil, fmt.Errorf("new_descriptor_set: for parameter \"deps\": got %s, want an iterable", deps.Type())
		}
		defer iter.Done()
		var x starlark.Value
		for iter.Next(&x) {
			ds, ok := x.(*DescriptorSet)
			if !ok {
				return nil, fmt.Errorf("new_descriptor_set: for parameter \"deps\" #%d: got %s, want proto.DescriptorSet", len(sets), x.Type())
			}
			sets = append(sets, ds)
		}
	}

	// Checks all imports can be resolved.
	ds, err := NewDescriptorSet(name, fdps, sets)
	if err != nil {
		return nil, fmt.Errorf("new_descriptor_set: %s", err)
	}
	return ds, nil
}

// newLoader constructs *Loader and populates it with given descriptor sets.
func newLoader(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(kwargs) > 0 {
		return nil, errors.New("new_loader: unexpected keyword arguments")
	}
	sets := make([]*DescriptorSet, len(args))
	for i, v := range args {
		ds, ok := v.(*DescriptorSet)
		if !ok {
			return nil, fmt.Errorf("new_loader: for parameter %d: got %s, want proto.DescriptorSet", i+1, v.Type())
		}
		sets[i] = ds
	}
	l := NewLoader()
	for _, ds := range sets {
		if err := l.AddDescriptorSet(ds); err != nil {
			return nil, fmt.Errorf("new_loader: %s", err)
		}
	}
	return l, nil
}

// defaultLoader returns *Loader installed in the thread via SetDefaultLoader.
func defaultLoader(th *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("default_loader", args, kwargs); err != nil {
		return nil, err
	}
	if l := DefaultLoader(th); l != nil {
		return l, nil
	}
	return starlark.None, nil
}

// messageType returns MessageType of the given message.
func messageType(th *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg *Message
	if err := starlark.UnpackArgs("message_type", args, kwargs, "msg", &msg); err != nil {
		return nil, err
	}
	return msg.MessageType(), nil
}

// marshallerBuiltin implements Starlark shim for To*PB() functions.
func marshallerBuiltin(name string, impl func(*Message) ([]byte, error)) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var msg *Message
		if err := starlark.UnpackArgs(name, args, kwargs, "msg", &msg); err != nil {
			return nil, err
		}
		blob, err := impl(msg)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", name, err)
		}
		return starlark.String(blob), nil
	})
}

// unmarshallerBuiltin implements Starlark shim for From*PB() functions.
func unmarshallerBuiltin(name string, impl func(*MessageType, []byte) (*Message, error)) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var ctor starlark.Value
		var body string
		if err := starlark.UnpackArgs(name, args, kwargs, "ctor", &ctor, "body", &body); err != nil {
			return nil, err
		}
		typ, ok := ctor.(*MessageType)
		if !ok {
			return nil, fmt.Errorf("%s: got %s, expecting a proto message constructor", name, ctor.Type())
		}
		msg, err := impl(typ, []byte(body))
		if err != nil {
			return nil, fmt.Errorf("%s: %s", name, err)
		}
		return msg, nil
	})
}

// TODO(vadimsh): Remove once users switch to protos.

// structToTextPb takes a struct and returns a string containing a text format
// protocol buffer.
func structToTextPb(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var val starlark.Value
	if err := starlark.UnpackArgs("struct_to_textpb", args, kwargs, "struct", &val); err != nil {
		return nil, err
	}
	s, ok := val.(*starlarkstruct.Struct)
	if !ok {
		return nil, fmt.Errorf("struct_to_textpb: got %s, expecting a struct", val.Type())
	}
	var buf bytes.Buffer
	err := writeProtoStruct(&buf, 0, s)
	if err != nil {
		return nil, err
	}
	return starlark.String(buf.String()), nil
}

// Based on
// https://github.com/google/starlark-go/blob/32ce6ec36500ded2e2340a430fae42bc43da8467/starlarkstruct/struct.go
func writeProtoStruct(out *bytes.Buffer, depth int, s *starlarkstruct.Struct) error {
	for _, name := range s.AttrNames() {
		val, err := s.Attr(name)
		if err != nil {
			return err
		}
		if err = writeProtoField(out, depth, name, val); err != nil {
			return err
		}
	}
	return nil
}

func writeProtoField(out *bytes.Buffer, depth int, field string, v starlark.Value) error {
	if depth > 16 {
		return fmt.Errorf("struct_to_textpb: depth limit exceeded")
	}

	switch v := v.(type) {
	case *starlarkstruct.Struct:
		fmt.Fprintf(out, "%*s%s: <\n", 2*depth, "", field)
		if err := writeProtoStruct(out, depth+1, v); err != nil {
			return err
		}
		fmt.Fprintf(out, "%*s>\n", 2*depth, "")
		return nil

	case *starlark.List, starlark.Tuple:
		iter := starlark.Iterate(v)
		defer iter.Done()
		var elem starlark.Value
		for iter.Next(&elem) {
			if err := writeProtoField(out, depth, field, elem); err != nil {
				return err
			}
		}
		return nil
	}

	// scalars
	fmt.Fprintf(out, "%*s%s: ", 2*depth, "", field)
	switch v := v.(type) {
	case starlark.Bool:
		fmt.Fprintf(out, "%t", v)

	case starlark.Int:
		out.WriteString(v.String())

	case starlark.Float:
		fmt.Fprintf(out, "%g", v)

	case starlark.String:
		fmt.Fprintf(out, "%q", string(v))

	default:
		return fmt.Errorf("struct_to_textpb: cannot convert %s to proto", v.Type())
	}
	out.WriteByte('\n')
	return nil
}
