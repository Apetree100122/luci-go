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

package secrets

import (
	"context"
	"errors"
)

var (
	// ErrNoStoreConfigured is returned by GetSecret if secret store is not in
	// the context.
	ErrNoStoreConfigured = errors.New("secrets.Store is not in the context")
)

// Factory knows how to make a new Store.
type Factory func(context.Context) Store

var contextKey = "secrets.Store"

// Get grabs a Store by calling Factory stored in the context. If one hasn't
// been set, it returns nil.
func Get(c context.Context) Store {
	if f, ok := c.Value(&contextKey).(Factory); ok && f != nil {
		return f(c)
	}
	return nil
}

// SetFactory sets the function to produce Store instances when Get(c) is used.
func SetFactory(c context.Context, f Factory) context.Context {
	return context.WithValue(c, &contextKey, f)
}

// Set injects the Store object in the context to be returned by Get as is.
func Set(c context.Context, s Store) context.Context {
	if s == nil {
		return SetFactory(c, nil)
	}
	return SetFactory(c, func(context.Context) Store { return s })
}

// GetSecret is a shortcut for grabbing a Store from the context and using its
// GetSecret method.
//
// If the context doesn't have Store set, returns ErrNoStoreConfigured.
func GetSecret(c context.Context, key string) (Secret, error) {
	if s := Get(c); s != nil {
		return s.GetSecret(key)
	}
	return Secret{}, ErrNoStoreConfigured
}
