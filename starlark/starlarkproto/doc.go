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

// Package starlarkproto exposes protobuf messages as starlark types.
//
// It is geared towards emitting messages, not reading or parsing them. Thus it
// provides only one-way bridge from Starlark to Go (but not vice-versa), i.e.
// Go programs can use Starlark scripts that return protobuf messages, but not
// accept them.
//
// Internally a message is stored as a tree of Starlark native values, with
// some type checking done when manipulating fields. For example, reading or
// assigning to a field not defined in a message will cause a runtime error.
// Similarly, trying to assign a value of a wrong type to a non-repeated field
// will fail.
//
// Repeated fields currently have more lax type checks: they just have to be
// lists or tuples. It is possible to put wrong values inside them, which will
// cause runtime error at a later stage, when trying to serialize the proto
// message (or materialize it as proto.Message on the Go side).
//
// Instantiating messages and default field values
//
// Each proto message in a loaded package is exposed via constructor function
// that takes optional keyword arguments and produces a new object of *Message
// type.
//
// All unassigned fields are implicitly set to their default zero values on
// first access, including message-typed fields. It means, for example, if a
// message 'a' has a singular field 'b', that has a field 'c', it is always fine
// to write 'a.b.c' to read or set 'c' value, without explicitly checking that
// 'b' is set.
//
// To clear a field, assign None to it (regardless of its type).
package starlarkproto

// TODO: support more known types (any, struct).
// TODO: support '==' ?
