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

import { CircularProgress, Link } from '@mui/material';

import { usePrpcQuery } from '@/common/hooks/use_prpc_query';
import { BuilderID } from '@/common/services/buildbucket';
import { MiloInternal } from '@/common/services/milo_internal';
import { extractProject } from '@/common/tools/utils';

const PAGE_SIZE = 100;

export interface ViewsSectionProps {
  readonly builderId: BuilderID;
}

export function ViewsSection({ builderId }: ViewsSectionProps) {
  const { data, error, isError, isLoading } = usePrpcQuery({
    host: '',
    insecure: location.protocol === 'http:',
    Service: MiloInternal,
    method: 'queryConsoles',
    request: {
      predicate: {
        builder: builderId,
      },
      pageSize: PAGE_SIZE,
    },
  });

  if (isError) {
    throw error;
  }

  return (
    <>
      <h3>Views</h3>
      {isLoading ? (
        <CircularProgress />
      ) : (
        <>
          <ul>
            {data.consoles?.map((c) => {
              const project = extractProject(c.realm);
              const consoleLabel = `${project} / ${c.id}`;
              return (
                <li key={consoleLabel}>
                  <Link href={`/p/${project}/g/${c.id}`}>{consoleLabel}</Link>
                </li>
              );
            })}
          </ul>
        </>
      )}
    </>
  );
}
