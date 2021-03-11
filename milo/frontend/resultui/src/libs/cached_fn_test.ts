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

import { aTimeout } from '@open-wc/testing-helpers/index-no-side-effects';
import { assert } from 'chai';
import Sinon, * as sinon from 'sinon';

import { cached, CacheOption } from './cached_fn';

describe('cached_fn', () => {
  let cachedFn: (opt: CacheOption, param1: number, param2: string) => string;
  let fnSpy: Sinon.SinonSpy<[number, string], string>;

  beforeEach(() => {
    let callCount = 0;
    const fn = (param1: number, param2: string) => `${param1}-${param2}-${callCount++}`;
    fnSpy = sinon.spy(fn);
    cachedFn = cached(fnSpy, {key: (...params) => JSON.stringify(params)});
  });

  it('should return cached response when params are identical', async () => {
    const res1 = cachedFn(CacheOption.Cached, 1, 'a');
    const res2 = cachedFn(CacheOption.Cached, 1, 'a');
    assert.strictEqual(res1, res2);
    assert.strictEqual(fnSpy.callCount, 1);
  });

  it('should return cached response when params are different', async () => {
    const res1 = cachedFn(CacheOption.Cached, 1, 'a');
    const res2 = cachedFn(CacheOption.Cached, 2, 'a');
    const res3 = cachedFn(CacheOption.Cached, 1, 'b');
    assert.strictEqual(res1, '1-a-0');
    assert.strictEqual(res2, '2-a-1');
    assert.strictEqual(res3, '1-b-2');
    assert.strictEqual(fnSpy.callCount, 3);
  });

  it('should be able to cache multiple different function calls', async () => {
    const res1a = cachedFn(CacheOption.Cached, 1, 'a');
    const res2a = cachedFn(CacheOption.Cached, 2, 'a');
    const res3a = cachedFn(CacheOption.Cached, 1, 'b');
    const res1b = cachedFn(CacheOption.Cached, 1, 'a');
    const res2b = cachedFn(CacheOption.Cached, 2, 'a');
    const res3b = cachedFn(CacheOption.Cached, 1, 'b');
    assert.strictEqual(res1a, res1b);
    assert.strictEqual(res2a, res2b);
    assert.strictEqual(res3a, res3b);
    assert.strictEqual(fnSpy.callCount, 3);
  });

  it('should refresh the cache when calling with ForceRefresh', async () => {
    const res1 = cachedFn(CacheOption.Cached, 1, 'a');
    const res2 = cachedFn(CacheOption.ForceRefresh, 1, 'a');
    const res3 = cachedFn(CacheOption.Cached, 1, 'a');
    assert.strictEqual(res1, '1-a-0');
    assert.strictEqual(res2, '1-a-1');
    assert.strictEqual(res3, '1-a-1');
    assert.strictEqual(fnSpy.callCount, 2);
  });

  it('should bypass the cache when calling with NoCache', async () => {
    const res1 = cachedFn(CacheOption.Cached, 1, 'a');
    const res2 = cachedFn(CacheOption.NoCache, 1, 'a');
    const res3 = cachedFn(CacheOption.Cached, 1, 'a');
    assert.strictEqual(res1, '1-a-0');
    assert.strictEqual(res2, '1-a-1');
    assert.strictEqual(res3, '1-a-0');
    assert.strictEqual(fnSpy.callCount, 2);
  });

  describe('when config.expire(...) returns a promise that resolves', () => {
    beforeEach(() => {
      cachedFn = cached(
        fnSpy,
        {
          key: (...params) => JSON.stringify(params),
          expire: () => aTimeout(20),
        },
      );
    });

    it('should return cached response when cache has not expired', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(10);
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, res2);
      assert.strictEqual(fnSpy.callCount, 1);
    });

    it('should return a new response when cache has expired', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(30);
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });

    it('should not invalidate refreshed cache too early', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(15);
      const res2 = cachedFn(CacheOption.ForceRefresh, 1, 'a');
      await aTimeout(15);
      const res3 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(res3, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });
  });

  describe('when config.expire() returns a promise that rejects', () => {
    beforeEach(() => {
      cachedFn = cached(
        fnSpy,
        {
          key: (...params) => JSON.stringify(params),
          expire: async () => {
            await aTimeout(20);
            throw new Error();
          },
        },
      );
    });

    it('should return cached response when cache has not expired', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(10);
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, res2);
      assert.strictEqual(fnSpy.callCount, 1);
    });

    it('should return a new response when cache has expired', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(30);
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });

    it('should not invalidate refreshed cache too early', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(15);
      const res2 = cachedFn(CacheOption.ForceRefresh, 1, 'a');
      await aTimeout(15);
      const res3 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(res3, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });
  });

  describe('when config.expire() resolves immediately', () => {
    beforeEach(() => {
      cachedFn = cached(
        fnSpy,
        {
          key: (...params) => JSON.stringify(params),
          expire: () => Promise.resolve(),
        },
      );
    });

    it('should not delete the cache before the function returns', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(fnSpy.callCount, 1);
    });

    it('should delete the cache in the next event cycle', async () => {
      const res1 = cachedFn(CacheOption.Cached, 1, 'a');
      await aTimeout(0);
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res1, '1-a-0');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });
  });

  describe('when config.expire() throws immediately', () => {
    beforeEach(() => {
      let firstCall = true;
      cachedFn = cached(
        fnSpy,
        {
          key: (...params) => JSON.stringify(params),
          expire: () => {
            if (firstCall) {
              firstCall = false;
              throw new Error();
            }
            return Promise.resolve();
          },
        },
      );
    });

    it('should not cache the response', async () => {
      try {
        cachedFn(CacheOption.Cached, 1, 'a');
      } catch {}
      const res2 = cachedFn(CacheOption.Cached, 1, 'a');
      assert.strictEqual(res2, '1-a-1');
      assert.strictEqual(fnSpy.callCount, 2);
    });
  });
});
