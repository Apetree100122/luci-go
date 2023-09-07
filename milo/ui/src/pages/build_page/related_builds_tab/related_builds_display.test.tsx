// Copyright 2023 The LUCI Authors.
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

import { UseQueryResult } from '@tanstack/react-query';
import { cleanup, render, screen } from '@testing-library/react';
import { act } from 'react-dom/test-utils';

import { usePrpcQueries } from '@/common/hooks/prpc_query';
import {
  Build,
  BuildbucketStatus,
  BuildsService,
  SEARCH_BUILD_FIELD_MASK,
  SearchBuildsResponse,
} from '@/common/services/buildbucket';
import { FakeContextProvider } from '@/testing_tools/fakes/fake_context_provider';

import { RelatedBuildTable } from './related_build_table';
import { RelatedBuildsDisplay } from './related_builds_display';

function createMockBuild(id: string): Build {
  return {
    id,
    builder: {
      bucket: 'buck',
      builder: 'builder',
      project: 'proj',
    },
    status: BuildbucketStatus.Success,
    createTime: '2020-01-01',
  };
}

jest.mock('@/common/hooks/prpc_query', () => {
  return createSelectiveMockFromModule<
    typeof import('@/common/hooks/prpc_query')
  >('@/common/hooks/prpc_query', ['usePrpcQueries']);
});

jest.mock('./related_build_table', () => {
  return createSelectiveSpiesFromModule<typeof import('./related_build_table')>(
    './related_build_table',
    ['RelatedBuildTable'],
  );
});

describe('RelatedBuildsDisplay', () => {
  let usePrpcQueriesMock: jest.MockedFunction<
    typeof usePrpcQueries<BuildsService, 'searchBuilds'>
  >;
  let relatedBuildsTableSpy: jest.MockedFunction<typeof RelatedBuildTable>;
  beforeEach(() => {
    jest.useFakeTimers();
    relatedBuildsTableSpy = jest.mocked(RelatedBuildTable);
    usePrpcQueriesMock = jest.mocked(
      usePrpcQueries<BuildsService, 'searchBuilds'>,
    );
  });

  afterEach(() => {
    cleanup();
    jest.useRealTimers();
    usePrpcQueriesMock.mockReset();
    relatedBuildsTableSpy.mockReset();
  });

  it('no buildset', async () => {
    usePrpcQueriesMock.mockReturnValue([]);
    render(
      <FakeContextProvider>
        <RelatedBuildsDisplay
          build={{ tags: [{ key: 'otherkey', value: 'othervalue' }] }}
        />
      </FakeContextProvider>,
    );
    expect(usePrpcQueriesMock).toHaveBeenCalledWith({
      host: SETTINGS.buildbucket.host,
      Service: BuildsService,
      method: 'searchBuilds',
      requests: [],
    });

    expect(
      screen.getByText('No other builds found with the same buildset'),
    ).toBeInTheDocument();
  });

  it('can dedupe builds', async () => {
    usePrpcQueriesMock.mockReturnValue([
      {
        data: {
          builds: [createMockBuild('00001'), createMockBuild('00002')],
        },
        isError: false,
        isLoading: false,
      } as Partial<
        UseQueryResult<SearchBuildsResponse>
      > as UseQueryResult<SearchBuildsResponse>,
      {
        data: {
          builds: [createMockBuild('00002'), createMockBuild('00003')],
        },
        isError: false,
        isLoading: false,
      } as Partial<
        UseQueryResult<SearchBuildsResponse>
      > as UseQueryResult<SearchBuildsResponse>,
    ]);
    render(
      <FakeContextProvider>
        <RelatedBuildsDisplay
          build={{
            tags: [
              { key: 'otherkey', value: 'othervalue' },
              { key: 'buildset', value: 'commit/git/1234' },
              { key: 'buildset', value: 'commit/gitiles/1234' },
              { key: 'buildset', value: 'commit/git/5678' },
              { key: 'buildset', value: 'commit/gitiles/5678' },
            ],
          }}
        />
      </FakeContextProvider>,
    );
    expect(usePrpcQueriesMock).toHaveBeenCalledWith({
      host: SETTINGS.buildbucket.host,
      Service: BuildsService,
      method: 'searchBuilds',
      requests: [
        {
          fields: SEARCH_BUILD_FIELD_MASK,
          pageSize: 1000,
          predicate: {
            tags: [
              {
                key: 'buildset',
                value: 'commit/gitiles/1234',
              },
            ],
          },
        },
        {
          fields: SEARCH_BUILD_FIELD_MASK,
          pageSize: 1000,
          predicate: {
            tags: [
              {
                key: 'buildset',
                value: 'commit/gitiles/5678',
              },
            ],
          },
        },
      ],
    });

    await act(() => jest.runAllTimersAsync());

    expect(relatedBuildsTableSpy).toHaveBeenCalledWith(
      {
        relatedBuilds: [
          createMockBuild('00001'),
          createMockBuild('00002'),
          createMockBuild('00003'),
        ],
        showLoadingBar: false,
      },
      expect.anything(),
    );
  });

  it('one query is loading', async () => {
    usePrpcQueriesMock.mockReturnValue([
      {
        data: {
          builds: [createMockBuild('00001'), createMockBuild('00002')],
        },
        isError: false,
        isLoading: false,
      } as Partial<
        UseQueryResult<SearchBuildsResponse>
      > as UseQueryResult<SearchBuildsResponse>,
      {
        data: undefined,
        isError: false,
        isLoading: true,
      } as Partial<
        UseQueryResult<SearchBuildsResponse>
      > as UseQueryResult<SearchBuildsResponse>,
    ]);
    render(
      <FakeContextProvider>
        <RelatedBuildsDisplay
          build={{
            tags: [
              { key: 'otherkey', value: 'othervalue' },
              { key: 'buildset', value: 'commit/git/1234' },
              { key: 'buildset', value: 'commit/gitiles/1234' },
              { key: 'buildset', value: 'commit/git/5678' },
              { key: 'buildset', value: 'commit/gitiles/5678' },
            ],
          }}
        />
      </FakeContextProvider>,
    );

    await act(() => jest.runAllTimersAsync());

    expect(relatedBuildsTableSpy).toHaveBeenCalledWith(
      {
        relatedBuilds: [],
        showLoadingBar: true,
      },
      expect.anything(),
    );
  });
});
