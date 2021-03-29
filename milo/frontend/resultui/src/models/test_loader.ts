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

/**
 * @fileoverview This file contains functions/classes that helps loading test
 * results and exonerations from resultDb to a TestNode.
 */

import { groupBy } from 'lodash-es';
import { action, computed, observable } from 'mobx';

import {
  QueryTestVariantsRequest,
  QueryTestVariantsResponse,
  TestVariant,
  TestVariantStatus,
  UISpecificService,
} from '../services/resultdb';

/**
 * The stage of the next test variant. The stage can be
 * 1. LoadingXXX: the status of the next test variant will be no worse than XXX.
 * 2. Done: all test variants have been loaded.
 */
export const enum LoadingStage {
  LoadingUnexpected = 0,
  LoadingUnexpectedlySkipped = 1,
  LoadingFlaky = 2,
  LoadingExonerated = 3,
  LoadingExpected = 4,
  Done = 5,
}

const VARIANT_STATUS_LOADING_STAGE_MAP = Object.freeze({
  [TestVariantStatus.TEST_VARIANT_STATUS_UNSPECIFIED]: LoadingStage.LoadingUnexpected,
  [TestVariantStatus.UNEXPECTED]: LoadingStage.LoadingUnexpected,
  [TestVariantStatus.UNEXPECTEDLY_SKIPPED]: LoadingStage.LoadingUnexpectedlySkipped,
  [TestVariantStatus.FLAKY]: LoadingStage.LoadingFlaky,
  [TestVariantStatus.EXONERATED]: LoadingStage.LoadingExonerated,
  [TestVariantStatus.EXPECTED]: LoadingStage.LoadingExpected,
});

export class LoadTestVariantsError {
  constructor(readonly req: QueryTestVariantsRequest, readonly source: unknown) {}

  toString() {
    return `error when loading test variants\n ${this.source}`;
  }
}

/**
 * Keeps the progress of the iterator and loads tests into the test node on
 * request.
 */
export class TestLoader {
  @observable.ref filter = (_v: TestVariant) => true;
  @observable.ref groupers: Array<[string, (v: TestVariant) => unknown]> = [];
  @observable.ref cmpFn = (_v1: TestVariant, _v2: TestVariant) => 0;

  @computed get isLoading() {
    return !this.loadedAllVariants && this.loadingReqCount !== 0;
  }
  @observable.ref private loadingReqCount = 0;

  @computed get firstRequestSent() {
    return this._firstRequestSent;
  }
  @observable.ref private _firstRequestSent = false;

  /**
   * The queryTestVariants RPC sorted the variant by status. We can use this to
   * tell the possible status of the next test variants and therefore avoid
   * unnecessary loading.
   */
  @computed get stage() {
    return this._stage;
  }
  @observable.ref private _stage = LoadingStage.LoadingUnexpected;

  @observable.ref unfilteredTestVariantCount = 0;

  @computed
  get testVariantCount() {
    return (
      this.unexpectedTestVariants.length +
      this.unexpectedlySkippedTestVariants.length +
      this.flakyTestVariants.length +
      this.exoneratedTestVariants.length +
      this.expectedTestVariants.length
    );
  }

  /**
   * non-expected test variants grouped by keys from groupByPropGetters.
   * expected test variants are not included.
   */
  @computed get groupedNonExpectedVariants() {
    if (this.nonExpectedTestVariants.length === 0) {
      return [];
    }

    let groups = [this.nonExpectedTestVariants];
    for (const [, propGetter] of this.groupers) {
      groups = groups.flatMap((group) => Object.values(groupBy(group, (v) => propGetter(v))));
    }
    return groups.map((group) => group.sort(this.cmpFn));
  }

  /**
   * non-expected test variants include test variants of any status except
   * TestVariantStatus.Expected.
   */
  @computed get nonExpectedTestVariants() {
    return this.unfilteredNonExpectedVariants.filter(this.filter);
  }
  @computed get unexpectedTestVariants() {
    return this.unfilteredUnexpectedVariants.filter(this.filter);
  }
  @computed get unexpectedlySkippedTestVariants() {
    return this.unfilteredUnexpectedlySkippedVariants.filter(this.filter);
  }
  @computed get flakyTestVariants() {
    return this.unfilteredFlakyVariants.filter(this.filter);
  }
  @computed get exoneratedTestVariants() {
    return this.unfilteredExoneratedVariants.filter(this.filter);
  }
  @computed get expectedTestVariants() {
    return this.unfilteredExpectedVariants.filter(this.filter);
  }

  @observable.shallow private unfilteredNonExpectedVariants: TestVariant[] = [];
  @observable.shallow private unfilteredUnexpectedVariants: TestVariant[] = [];
  @observable.shallow private unfilteredUnexpectedlySkippedVariants: TestVariant[] = [];
  @observable.shallow private unfilteredFlakyVariants: TestVariant[] = [];
  @observable.shallow private unfilteredExoneratedVariants: TestVariant[] = [];
  @observable.shallow private unfilteredExpectedVariants: TestVariant[] = [];

  @computed get loadedAllVariants() {
    return this.stage === LoadingStage.Done;
  }
  @computed get loadedAllUnexpectedVariants() {
    return this.stage > LoadingStage.LoadingUnexpected;
  }
  @computed get loadedAllUnexpectedlySkippedVariants() {
    return this.stage > LoadingStage.LoadingUnexpectedlySkipped;
  }
  @computed get loadedAllFlakyVariants() {
    return this.stage > LoadingStage.LoadingFlaky;
  }
  @computed get loadedAllExoneratedVariants() {
    return this.stage > LoadingStage.LoadingExonerated;
  }
  @computed get loadedAllExpectedVariants() {
    return this.stage > LoadingStage.LoadingExpected;
  }
  @computed get firstPageLoaded() {
    return this.unfilteredUnexpectedVariants.length > 0 || this.loadedAllUnexpectedVariants;
  }
  @computed get firstPageIsEmpty() {
    return (
      this.loadedAllUnexpectedVariants &&
      this.unfilteredUnexpectedVariants.length === 0 &&
      this.unfilteredUnexpectedlySkippedVariants.length === 0 &&
      this.unfilteredFlakyVariants.length === 0 &&
      this.unfilteredExoneratedVariants.length === 0
    );
  }

  // undefined means the end has been reached.
  // empty string is the token for the first page.
  private nextPageToken: string | undefined = '';

  constructor(private readonly req: QueryTestVariantsRequest, private readonly uiSpecificService: UISpecificService) {}

  private loadPromise = Promise.resolve();

  /**
   * Load at least one test variant of the specified status (or any status if
   * not specified) unless the last page is reached.
   */
  loadNextTestVariants(untilStatus?: TestVariantStatus) {
    this._firstRequestSent = true;
    if (this.stage === LoadingStage.Done) {
      return this.loadPromise;
    }

    this.loadingReqCount++;
    this.loadPromise = this.loadPromise.then(() => this.loadNextTestVariantsInternal(untilStatus));
    return this.loadPromise.then(() => this.loadingReqCount--);
  }

  /**
   * Load at least one test variant of the specified status (or any status if
   * not specified) unless the last page is reached.
   *
   * @precondition there should not exist a running instance of
   * this.loadNextTestVariantsInternal
   */
  private async loadNextTestVariantsInternal(untilStatus?: TestVariantStatus) {
    const beforeCount = this.testVariantCount;

    let shouldLoadNextPage: boolean;

    // Load pages until the next expected status is at least the one we're after.
    do {
      await this.loadNextPage();
      shouldLoadNextPage =
        (!this.loadedAllVariants && this.testVariantCount === beforeCount) ||
        (untilStatus !== undefined && this.stage < VARIANT_STATUS_LOADING_STAGE_MAP[untilStatus]);
    } while (shouldLoadNextPage);

    // If we wanted to load up to Expected, and none have arrived yet
    // (i.e. we're at the point where ResultDB has just returned the final
    // non-expected variants to us), load one more page.
    // Note: the while loop above only ensures at least one test variant of
    // *any* status is loaded.
    if (untilStatus === TestVariantStatus.EXPECTED && this.unfilteredExpectedVariants.length === 0) {
      await this.loadNextPage();
    }
  }

  /**
   * Loads the next batch of tests from the iterator to the node.
   *
   * @precondition there should not exist a running instance of
   * this.loadMoreInternal
   */
  private async loadNextPage() {
    if (this.nextPageToken === undefined) {
      return;
    }

    const req = { ...this.req, pageToken: this.nextPageToken };
    let res: QueryTestVariantsResponse;
    try {
      res = await this.uiSpecificService.queryTestVariants(req);
    } catch (e) {
      throw new LoadTestVariantsError(req, e);
    }

    this.nextPageToken = res.nextPageToken;

    const testVariants = res.testVariants || [];
    this.processTestVariants(testVariants);
    if (this.nextPageToken === undefined) {
      this._stage = LoadingStage.Done;
      return;
    }
    if (testVariants.length < (this.req.pageSize || 1000)) {
      // When the service returns an incomplete page and nextPageToken is not
      // undefined, the following pages must be expected test variants.
      // Without this special case, the UI may incorrectly indicate that not all
      // variants have been loaded for statuses worse than Expected.
      this._stage = LoadingStage.LoadingExpected;
      return;
    }
  }

  @action
  private processTestVariants(testVariants: readonly TestVariant[]) {
    this.unfilteredTestVariantCount += testVariants.length;
    for (const testVariant of testVariants) {
      switch (testVariant.status) {
        case TestVariantStatus.UNEXPECTED:
          this._stage = LoadingStage.LoadingUnexpected;
          this.unfilteredUnexpectedVariants.push(testVariant);
          this.unfilteredNonExpectedVariants.push(testVariant);
          break;
        case TestVariantStatus.UNEXPECTEDLY_SKIPPED:
          this._stage = LoadingStage.LoadingUnexpectedlySkipped;
          this.unfilteredUnexpectedlySkippedVariants.push(testVariant);
          this.unfilteredNonExpectedVariants.push(testVariant);
          break;
        case TestVariantStatus.FLAKY:
          this._stage = LoadingStage.LoadingFlaky;
          this.unfilteredFlakyVariants.push(testVariant);
          this.unfilteredNonExpectedVariants.push(testVariant);
          break;
        case TestVariantStatus.EXONERATED:
          this._stage = LoadingStage.LoadingExonerated;
          this.unfilteredExoneratedVariants.push(testVariant);
          this.unfilteredNonExpectedVariants.push(testVariant);
          break;
        case TestVariantStatus.EXPECTED:
          this._stage = LoadingStage.LoadingExpected;
          this.unfilteredExpectedVariants.push(testVariant);
          break;
        default:
          break;
      }
    }
  }
}
