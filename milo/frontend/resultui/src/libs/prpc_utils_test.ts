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

import { PrpcClient } from '@chopsui/prpc-client';
import { assert } from '@open-wc/testing/index-no-side-effects';
import * as sinon from 'sinon';

import { genCacheKeyForPrpcRequest } from './prpc_utils';

describe('genCacheKeyForPrpcRequest', () => {
  it('should generate identical keys for identical requests', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 });
    client.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.strictEqual(key1, key2);
  });

  it('should generate different keys for requests with different hosts', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client1 = new PrpcClient({ host: 'host1', fetchImpl: fetchStub });
    const client2 = new PrpcClient({ host: 'host2', fetchImpl: fetchStub });
    client1.call('service', 'method', { prop: 1 });
    client2.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different access tokens', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client1 = new PrpcClient({ accessToken: 'token1', fetchImpl: fetchStub });
    const client2 = new PrpcClient({ accessToken: 'token2', fetchImpl: fetchStub });
    client1.call('service', 'method', { prop: 1 });
    client2.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different secure modes', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client1 = new PrpcClient({ insecure: true, fetchImpl: fetchStub });
    const client2 = new PrpcClient({ insecure: false, fetchImpl: fetchStub });
    client1.call('service', 'method', { prop: 1 });
    client2.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different services', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service1', 'method', { prop: 1 });
    client.call('service2', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different methods', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method1', { prop: 1 });
    client.call('service', 'method2', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different bodies', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 });
    client.call('service', 'method', { prop: 2 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different critical headers', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 }, { 'header-key': 'header-value1' });
    client.call('service', 'method', { prop: 1 }, { 'header-key': 'header-value2' });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args), ['header-key']);
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args), ['header-key']);
    assert.notStrictEqual(key1, key2);
  });

  it('should generate different keys for requests with different default critical headers', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call(
      'service',
      'method',
      { prop: 1 },
      { accept: 'accept', 'content-type': 'content-type', authorization: 'token' }
    );
    client.call(
      'service',
      'method',
      { prop: 1 },
      { accept: 'accept1', 'content-type': 'content-type', authorization: 'token' }
    );
    client.call(
      'service',
      'method',
      { prop: 1 },
      { accept: 'accept', 'content-type': 'content-type1', authorization: 'token' }
    );
    client.call(
      'service',
      'method',
      { prop: 1 },
      { accept: 'accept', 'content-type': 'content-type', authorization: 'token1' }
    );

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    const key3 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(2).args));
    const key4 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(3).args));
    assert.notStrictEqual(key1, key2);
    assert.notStrictEqual(key1, key3);
    assert.notStrictEqual(key1, key4);
  });

  it('should generate identical keys for requests with different non-critical headers', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 }, { 'header-key': 'header-value1' });
    client.call('service', 'method', { prop: 1 }, { 'header-key': 'header-value2' });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.strictEqual(key1, key2);
  });

  it('should generate identical keys for identical requests with different header key cases', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 }, { 'HEADER-KEY': 'header-value' });
    client.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.strictEqual(key1, key2);
  });

  it('should generate different keys when prefix are different', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1 });
    client.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix1', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix2', new Request(...fetchStub.getCall(1).args));
    assert.notStrictEqual(key1, key2);
  });

  it('should generate identical keys when false-ish properties are omitted', async () => {
    const fetchStub = sinon.stub<[RequestInfo, RequestInit | undefined], Promise<Response>>();
    const client = new PrpcClient({ fetchImpl: fetchStub });
    client.call('service', 'method', { prop: 1, pageToken: '', emptyArray: [] });
    client.call('service', 'method', { prop: 1 });

    const key1 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(0).args));
    const key2 = await genCacheKeyForPrpcRequest('prefix', new Request(...fetchStub.getCall(1).args));
    assert.strictEqual(key1, key2);
  });
});
