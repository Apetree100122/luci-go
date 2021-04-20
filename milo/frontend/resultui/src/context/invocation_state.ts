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

import { autorun, comparer, computed, observable } from 'mobx';
import { fromPromise, IPromiseBasedObservable } from 'mobx-utils';

import { consumeContext, provideContext } from '../libs/context';
import { parseSearchQuery } from '../libs/search_query';
import { unwrapObservable } from '../libs/utils';
import { TestLoader } from '../models/test_loader';
import { TestPresentationConfig } from '../services/buildbucket';
import { createTVCmpFn, createTVPropGetter, Invocation, TestVariant } from '../services/resultdb';
import { AppState } from './app_state';

export class QueryInvocationError extends Error {
  constructor(readonly invId: string, readonly source: Error) {
    super(source.message);
  }
}

/**
 * Records state of an invocation.
 */
export class InvocationState {
  // '' means no associated invocation ID.
  // null means uninitialized.
  @observable.ref invocationId: string | null = null;

  @observable.ref searchText = '';
  @observable.ref searchFilter = (_v: TestVariant) => true;

  @observable.ref presentationConfig: TestPresentationConfig = {};
  @observable.ref columnsParam?: string[];
  @computed({ equals: comparer.shallow }) get defaultColumns() {
    return this.presentationConfig.column_keys || [];
  }
  @computed({ equals: comparer.shallow }) get displayedColumns() {
    return this.columnsParam || this.defaultColumns;
  }
  @computed get displayedColumnGetters() {
    return this.displayedColumns.map((col) => createTVPropGetter(col));
  }

  @observable customColumnWidths: { readonly [key: string]: number } = {};
  @computed get columnWidths() {
    return this.displayedColumns.map((col) => this.customColumnWidths[col] ?? 100);
  }

  @observable.ref sortingKeysParam?: string[];
  @computed({ equals: comparer.shallow }) get defaultSortingKeys() {
    return ['status', ...this.defaultColumns, 'name'];
  }
  @computed({ equals: comparer.shallow }) get sortingKeys() {
    return this.sortingKeysParam || this.defaultSortingKeys;
  }
  @computed get testVariantCmpFn(): (v1: TestVariant, v2: TestVariant) => number {
    return createTVCmpFn(this.sortingKeys);
  }

  @observable.ref groupingKeysParam?: string[];
  @computed({ equals: comparer.shallow }) get defaultGroupingKeys() {
    return this.presentationConfig.grouping_keys || ['status'];
  }
  @computed({ equals: comparer.shallow }) get groupingKeys() {
    return this.groupingKeysParam || this.defaultGroupingKeys;
  }
  @computed get groupers(): Array<[string, (v: TestVariant) => unknown]> {
    return this.groupingKeys.map((key) => [key, createTVPropGetter(key)]);
  }

  private disposers: Array<() => void> = [];
  constructor(private appState: AppState) {
    this.disposers.push(
      autorun(() => {
        try {
          this.searchFilter = parseSearchQuery(this.searchText);
        } catch (e) {
          //TODO(weiweilin): display the error to the user.
          console.error(e);
        }
      })
    );
    this.disposers.push(
      autorun(() => {
        if (!this.testLoader) {
          return;
        }
        this.testLoader.filter = this.searchFilter;
        this.testLoader.groupers = this.groupers;
        this.testLoader.cmpFn = this.testVariantCmpFn;
      })
    );
  }

  @observable.ref private isDisposed = false;

  /**
   * Perform cleanup.
   * Must be called before the object is GCed.
   */
  dispose() {
    this.isDisposed = true;
    for (const disposer of this.disposers) {
      disposer();
    }

    // Evaluates @computed({keepAlive: true}) properties after this.isDisposed
    // is set to true so they no longer subscribes to any external observable.
    this.testLoader;
  }

  @computed
  get invocationName(): string | null {
    if (!this.invocationId) {
      return null;
    }
    return 'invocations/' + this.invocationId;
  }

  @computed
  private get invocation$(): IPromiseBasedObservable<Invocation> {
    if (!this.appState.resultDb || !this.invocationName) {
      // Returns a promise that never resolves when resultDb isn't ready.
      return fromPromise(Promise.race([]));
    }
    const invId = this.invocationId;
    return fromPromise(
      this.appState.resultDb.getInvocation({ name: this.invocationName }).catch((e) => {
        throw new QueryInvocationError(invId!, e);
      })
    );
  }

  @computed
  get invocation(): Invocation | null {
    return unwrapObservable(this.invocation$, null);
  }

  @computed({ keepAlive: true })
  get testLoader(): TestLoader | null {
    if (this.isDisposed || !this.invocationName || !this.appState.uiSpecificService) {
      return null;
    }
    return new TestLoader({ invocations: [this.invocationName] }, this.appState.uiSpecificService);
  }
}

export const consumeInvocationState = consumeContext<'invocationState', InvocationState>('invocationState');
export const provideInvocationState = provideContext<'invocationState', InvocationState>('invocationState');
