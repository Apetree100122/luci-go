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

import { useQuery } from 'react-query';
import { getClustersService, QueryClusterHistoryRequest } from '@/services/cluster';
import { prpcRetrier } from '@/services/shared_models';

const useQueryClusterHistory = (
    project: string,
    failureFilter: string,
    days: number,
    metricNames: string[],
) => {
  const clustersService = getClustersService();
  return useQuery(['cluster', project, 'history', days, failureFilter, metricNames.join(',')], async () => {
    const request: QueryClusterHistoryRequest = {
      project,
      failureFilter,
      days,
      metrics: metricNames,
    };

    const response = await clustersService.queryHistory(request);

    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    return response;
  }, {
    retry: prpcRetrier,
    enabled: metricNames.length > 0,
  });
};

export default useQueryClusterHistory;
