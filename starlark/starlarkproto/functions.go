// Copyright 2018 The LUCI Authors.
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
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ProtoLib() returns a dict with single struct named "proto" that holds helper
// functions to manipulate protobuf messages (in particular serialize them).
//
// Exported functions:
//
//    def to_pbtext(msg):
//      """Serializes a protobuf message to text proto.
//
//      Args:
//        msg: a *Message to serialize.
//
//      Returns:
//        An str representing msg in text format.
//      """
//
//    def to_jsonpb(msg):
//      """Serializes a protobuf message to JSONPB string.
//
//      Args:
//        msg: a *Message to serialize.
//
//      Returns:
//        An str representing msg in JSONPB format.
//      """
func ProtoLib() starlark.StringDict {
	return starlark.StringDict{
		"proto": starlarkstruct.FromStringDict(starlark.String("proto"), starlark.StringDict{
			"to_pbtext": starlark.NewBuiltin("to_pbtext", toPbText),
			"to_jsonpb": starlark.NewBuiltin("to_jsonpb", toJSONPb),
		}),
	}
}

// toPbText takes a single protobuf message and serializes it using text
// protobuf serialization.
func toPbText(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg *Message
	if err := starlark.UnpackArgs("to_pbtext", args, kwargs, "msg", &msg); err != nil {
		return nil, err
	}
	pb, err := msg.ToProto()
	if err != nil {
		return nil, err
	}
	return starlark.String(proto.MarshalTextString(pb)), nil
}

// toJSONPb takes a single protobuf message and serializes it using JSONPB
// serialization.
func toJSONPb(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg *Message
	var emitDefaults starlark.Bool
	if err := starlark.UnpackArgs("to_jsonpb", args, kwargs, "msg", &msg, "emit_defaults?", &emitDefaults); err != nil {
		return nil, err
	}
	pb, err := msg.ToProto()
	if err != nil {
		return nil, err
	}
	// More jsonpb Marshaler options may be added here as needed.
	var jsonMarshaler = &jsonpb.Marshaler{Indent: "\t", EmitDefaults: bool(emitDefaults)}
	str, err := jsonMarshaler.MarshalToString(pb)
	if err != nil {
		return nil, err
	}
	return starlark.String(str), nil
}
