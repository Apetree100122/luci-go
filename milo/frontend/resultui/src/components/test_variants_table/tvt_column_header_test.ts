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

import { fixture, fixtureCleanup, html } from '@open-wc/testing/index-no-side-effects';
import { assert } from 'chai';

import './tvt_column_header';
import { AppState } from '../../context/app_state';
import { InvocationState } from '../../context/invocation_state';
import { TestVariantsTableColumnHeader } from './tvt_column_header';

describe('tvt_column_header', () => {
  let invState: InvocationState;
  let columnHeaderEle: TestVariantsTableColumnHeader;

  beforeEach(async () => {
    invState = new InvocationState(new AppState());
    columnHeaderEle = await fixture<TestVariantsTableColumnHeader>(html`
      <milo-tvt-column-header .invocationState=${invState} .propKey=${'v.propKey'} .label=${'label'}>
      </milo-tvt-column-header>
    `);
  });
  afterEach(fixtureCleanup);

  it('can hide column', async () => {
    invState.columnsParam = ['v.propKey'];

    assert.deepEqual(invState.displayedColumns, ['v.propKey']);

    columnHeaderEle.hideColumn();

    assert.deepEqual(invState.displayedColumns, []);
  });

  it('can group rows', async () => {
    invState.columnsParam = ['v.propKey'];

    assert.deepEqual(invState.displayedColumns, ['v.propKey']);
    assert.deepEqual(invState.groupingKeys, ['status']);

    columnHeaderEle.groupRows();

    // The grouping key should be removed from displayedColumns.
    assert.deepEqual(invState.displayedColumns, []);
    // The new grouping key should prepended to the existing grouping keys.
    assert.deepEqual(invState.groupingKeys, ['v.propKey', 'status']);
  });

  it('should not group by the same column multiple times', async () => {
    invState.groupingKeysParam = ['status', 'v.propKey'];

    assert.deepEqual(invState.groupingKeys, ['status', 'v.propKey']);

    columnHeaderEle.groupRows();

    // The new grouping key should prepended to the existing grouping keys.
    // Duplicated grouping keys should be removed.
    assert.deepEqual(invState.groupingKeys, ['v.propKey', 'status']);
  });

  it('can sort rows', async () => {
    invState.columnsParam = ['v.propKey'];

    assert.deepEqual(invState.displayedColumns, ['v.propKey']);
    assert.deepEqual(invState.sortingKeys, ['status', 'v.test_suite', 'name']);

    columnHeaderEle.sortColumn(true);

    // The sorting key should not be removed from displayedColumns.
    assert.deepEqual(invState.displayedColumns, ['v.propKey']);
    // The new sorting key should prepended to the existing sorting keys.
    assert.deepEqual(invState.sortingKeys, ['v.propKey', 'status', 'v.test_suite', 'name']);
  });

  it('can sort rows in descending order', async () => {
    invState.columnsParam = ['v.propKey'];

    assert.deepEqual(invState.displayedColumns, ['v.propKey']);
    assert.deepEqual(invState.sortingKeys, ['status', 'v.test_suite', 'name']);

    columnHeaderEle.sortColumn(false);

    // The sorting key should not be removed from displayedColumns.
    assert.deepEqual(invState.displayedColumns, ['v.propKey']);
    // The new sorting key should prepended to the existing sorting keys.
    assert.deepEqual(invState.sortingKeys, ['-v.propKey', 'status', 'v.test_suite', 'name']);
  });

  it('should not sort by the same column multiple times', async () => {
    invState.sortingKeysParam = ['status', 'v.propKey'];
    assert.deepEqual(invState.sortingKeys, ['status', 'v.propKey']);
    columnHeaderEle.sortColumn(true);
    // The new sorting key should prepended to the existing sorting keys.
    // Duplicated sorting keys should be removed.
    assert.deepEqual(invState.sortingKeys, ['v.propKey', 'status']);

    invState.sortingKeysParam = ['status', '-v.propKey'];
    assert.deepEqual(invState.sortingKeys, ['status', '-v.propKey']);
    columnHeaderEle.sortColumn(true);
    // Sorting keys with '-' prefix should be removed too if they have the same
    // prop key.
    assert.deepEqual(invState.sortingKeys, ['v.propKey', 'status']);

    invState.sortingKeysParam = ['status', 'v.propKey'];
    assert.deepEqual(invState.sortingKeys, ['status', 'v.propKey']);
    // Sorting in descending order should remove duplicated keys too.
    columnHeaderEle.sortColumn(false);
    assert.deepEqual(invState.sortingKeys, ['-v.propKey', 'status']);
  });
});
