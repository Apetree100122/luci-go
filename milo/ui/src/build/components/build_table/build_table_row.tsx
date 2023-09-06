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

import { TableRow } from '@mui/material';
import { ReactNode, useEffect, useState } from 'react';

import { Build } from '@/common/services/buildbucket';

import {
  BuildProvider,
  RowExpandedProvider,
  SetRowExpandedProvider,
  useDefaultExpanded,
} from './context';

export interface BuildTableRowProps {
  readonly build: Build;
  readonly children: ReactNode;
}

export function BuildTableRow({ build, children }: BuildTableRowProps) {
  const defaultExpanded = useDefaultExpanded();
  const [expanded, setExpanded] = useState(() => defaultExpanded);
  useEffect(() => {
    setExpanded(defaultExpanded);
  }, [defaultExpanded]);

  return (
    <TableRow
      // Set a CSS class to support CSS-only toggle solution.
      className={
        expanded ? 'BuildTableRow-expanded' : 'BuildTableRow-collapsed'
      }
      sx={{
        '& > td': {
          // Use `vertical-align: baseline` so the cell content (including the
          // expand button) won't shift downwards when the row is expanded.
          verticalAlign: 'baseline',
          whiteSpace: 'nowrap',
        },
      }}
    >
      <SetRowExpandedProvider value={setExpanded}>
        {/* Provide the expanded state via context in case a CSS-only toggle
         ** solution can't be achieved. */}
        <RowExpandedProvider value={expanded}>
          {/* Pass build to cells via context so composing a row require less
           ** boilerplate. */}
          <BuildProvider value={build}>{children}</BuildProvider>
        </RowExpandedProvider>
      </SetRowExpandedProvider>
    </TableRow>
  );
}
