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

import '@material/mwc-button';
import {
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
} from '@mui/material';
import { observer } from 'mobx-react-lite';
import { useId, useState } from 'react';

import { RecoverableErrorBoundary } from '@/common/components/error_handling';
import { useStore } from '@/common/store';
import { getGitilesRepoURL } from '@/common/tools/gitiles_utils';
import { useTabId } from '@/generic_libs/components/routed_tabs';

import { BlamelistDisplay } from './blamelist_display';

export const BlamelistTab = observer(() => {
  const store = useStore();
  const build = store.buildPage.build;

  const repoSelectorLabelId = useId();

  const [selectedBlamelistPinIndex, setSelectedBlamelistPinIndex] = useState(0);
  const selectedBlamelistPin = build?.blamelistPins[selectedBlamelistPinIndex];

  if (!build) {
    return <CircularProgress sx={{ margin: '10px' }} />;
  }

  if (!selectedBlamelistPin) {
    return (
      <div css={{ padding: '10px' }}>
        Blamelist is not available because the build has no associated gitiles
        commit.
      </div>
    );
  }

  return (
    <>
      <FormControl size="small" sx={{ margin: '10px' }}>
        <InputLabel id={repoSelectorLabelId}>Repo</InputLabel>
        <Select
          labelId={repoSelectorLabelId}
          label="Repo"
          value={selectedBlamelistPinIndex}
          disabled={build.blamelistPins.length <= 1}
          onChange={(e) =>
            setSelectedBlamelistPinIndex(e.target.value as number)
          }
          sx={{ width: '500px' }}
        >
          {build.blamelistPins.map((pin, i) => (
            <MenuItem key={i} value={i}>
              {getGitilesRepoURL(pin)}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      <BlamelistDisplay
        blamelistPin={selectedBlamelistPin}
        builder={build.data.builder}
      />
    </>
  );
});

export function Component() {
  useTabId('blamelist');

  return (
    // See the documentation for `<LoginPage />` for why we handle error this
    // way.
    <RecoverableErrorBoundary key="blamelist">
      <BlamelistTab />
    </RecoverableErrorBoundary>
  );
}
