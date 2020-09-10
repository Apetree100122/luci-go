// Copyright 2020 The LUCI Authors.
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

import { ChainableURL } from './utils';

describe('utils', () => {
  describe('ChainableURL', () => {
    describe('withSearchParam', () => {
      const url = new ChainableURL('https://www.google.com/path?key1=val1');

      it('should set search params correctly', async () => {
        const newUrlStr = url
          .withSearchParam('key2', 'val2')
          .withSearchParam('key3', 'val3')
          .toString();
        assert.equal(newUrlStr, 'https://www.google.com/path?key1=val1&key2=val2&key3=val3');
      });

      it('should append search params correctly', async () => {
        const newUrlStr = url
          .withSearchParam('key2', 'val2')
          .withSearchParam('key3', 'val3')
          .toString();
        assert.equal(newUrlStr, 'https://www.google.com/path?key1=val1&key2=val2&key3=val3&key2=val2&key3=val3');
      });

      it('should override search params correctly', async () => {
        const newUrlStr = url
          .withSearchParam('key1', 'newVal1', true)
          .toString();
        assert.equal(newUrlStr, 'https://www.google.com/path?key1=newVal1&key2=val2&key3=val3&key2=val2&key3=val3');
      });

      it('should override search params with multiple values correctly', async () => {
        const newUrlStr = url
          .withSearchParam('key2', 'newVal2', true)
          .toString();
        assert.equal(newUrlStr, 'https://www.google.com/path?key1=newVal1&key2=newVal2&key3=val3&key3=val3');
      });
    });
  });
});
