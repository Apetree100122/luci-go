// Copyright 2021 The LUCI Authors.
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

import { assert } from 'chai';

import { getAuthState, getAuthStateSync, setAuthState } from './auth_state';

describe('auth_state', () => {
  it('support accessing auth state synchronously', () => {
    const state = {
      accessToken: Math.random().toString(),
      identity: `user: ${Math.random()}`,
      accessTokenExpiry: Date.now() / 1000 + 1000,
    };
    setAuthState(state);
    assert.deepEqual(getAuthStateSync(), state);
  });

  it('support accessing auth state asynchronously', async () => {
    const state = {
      accessToken: Math.random().toString(),
      identity: `user: ${Math.random()}`,
      accessTokenExpiry: Date.now() / 1000 + 1000,
    };
    setAuthState(state);
    assert.deepEqual(await getAuthState(), state);
  });

  it('clear expired auth state when accessing synchronously', () => {
    const state = {
      accessToken: Math.random().toString(),
      identity: `user: ${Math.random()}`,
      accessTokenExpiry: Date.now() / 1000 - 1000,
    };
    setAuthState(state);
    assert.deepEqual(getAuthStateSync(), null);
  });

  it('clear expired auth state when accessing asynchronously', async () => {
    const state = {
      accessToken: Math.random().toString(),
      identity: `user: ${Math.random()}`,
      accessTokenExpiry: Date.now() / 1000 - 1000,
    };
    setAuthState(state);
    assert.deepEqual(await getAuthState(), null);
  });
});
