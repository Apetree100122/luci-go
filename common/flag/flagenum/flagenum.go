// Copyright 2016 The LUCI Authors.
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

// Package flagenum is a utility package which facilitates implementation of
// flag.Value, json.Marshaler, and json.Unmarshaler interfaces via a string-to-
// value mapping.
package flagenum

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Enum is a mapping of enumeration key strings to values that can be used as
// flags.
//
// Strings can be mapped to any value type that is comparable via
// reflect.DeepEqual.
type Enum map[string]any

// GetKey performs reverse lookup of the enumeration value, returning the
// key that corresponds to the value.
//
// If multiple keys correspond to the same value, the result is undefined.
func (e Enum) GetKey(value any) string {
	for k, v := range e {
		if reflect.DeepEqual(v, value) {
			return k
		}
	}
	return ""
}

// GetValue returns the mapped enumeration value associated with a key.
func (e Enum) GetValue(key string) (any, error) {
	for k, v := range e {
		if k == key {
			return v, nil
		}
	}
	return nil, fmt.Errorf("flagenum: Invalid value; must be one of [%s]", e.Choices())
}

// Choices returns a comma-separated string listing sorted enumeration choices.
func (e Enum) Choices() string {
	keys := make([]string, 0, len(e))
	for k := range e {
		if k == "" {
			k = `""`
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// Sets the value v to the enumeration value mapped to the supplied key.
func (e Enum) setValue(v any, key string) error {
	i, err := e.GetValue(key)
	if err != nil {
		return err
	}

	vValue := reflect.ValueOf(v).Elem()
	if !vValue.CanSet() {
		panic(fmt.Errorf("flagenum: Cannot set supplied value, %v", vValue))
	}

	iValue := reflect.ValueOf(i)
	if !vValue.Type().AssignableTo(iValue.Type()) {
		panic(fmt.Errorf("flagenum: Enumeration type (%v) is incompatible with supplied value (%v)",
			vValue.Type(), iValue.Type()))
	}

	vValue.Set(iValue)
	return nil
}

// FlagSet implements flag.Value's Set semantics. It identifies the mapped value
// associated with the supplied key and stores it in the supplied interface.
//
// The interface, v, must be a valid pointer to the mapped enumeration type.
func (e Enum) FlagSet(v any, key string) error {
	return e.setValue(v, key)
}

// FlagString implements flag.Value's String semantics.
func (e Enum) FlagString(v any) string {
	return e.GetKey(v)
}

// JSONUnmarshal implements json.Unmarshaler's UnmarshalJSON semantics. It
// parses data containing a quoted string, identifies the enumeration value
// associated with that string, and stores it in the supplied interface.
//
// The interface, v, must be a valid pointer to the mapped enumeration type.
// a string corresponding to one of the enum's keys.
func (e Enum) JSONUnmarshal(v any, data []byte) error {
	s := ""
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return e.FlagSet(v, s)
}

// JSONMarshal implements json.Marshaler's MarshalJSON semantics. It marshals
// the value in the supplied interface to its associated key and emits a quoted
// string containing that key.
//
// The interface, v, must be a valid pointer to the mapped enumeration type.
func (e Enum) JSONMarshal(v any) ([]byte, error) {
	key := e.GetKey(v)
	data, err := json.Marshal(&key)
	if err != nil {
		return nil, err
	}
	return data, nil
}
