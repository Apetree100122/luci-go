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

import Box from '@mui/material/Box';
import { ChangeEvent, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useDebounce } from 'react-use';

import { RecoverableErrorBoundary } from '@/common/components/error_handling';
import { PageMeta } from '@/common/components/page_meta';
import { UiPage } from '@/common/constants/view';
import { useSyncedSearchParams } from '@/generic_libs/hooks/synced_search_params';

import { SearchInput } from '../search_input';
import { DEFAULT_TEST_PROJECT } from '../search_redirection_loader';

import { TestList } from './test_list';

export const TestSearch = () => {
  const { project } = useParams();
  const [searchParams, setSearchParams] = useSyncedSearchParams();
  const searchQuery = searchParams.get('q') || '';
  const [pendingSearchQuery, setPendingSearchQuery] = useState(searchQuery);

  useDebounce(
    () => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (pendingSearchQuery === '') {
          next.delete('q');
        } else {
          next.set('q', pendingSearchQuery);
        }
        return next;
      });
    },
    600,
    [pendingSearchQuery],
  );

  const handleSearchQueryChange = (
    e: ChangeEvent<HTMLTextAreaElement | HTMLInputElement>,
  ) => {
    setPendingSearchQuery(e.target.value);
  };

  const selectedProject = project || DEFAULT_TEST_PROJECT;

  return (
    <Box sx={{ px: 6, py: 5 }}>
      <PageMeta
        title="Test search"
        project={selectedProject}
        selectedPage={UiPage.TestHistory}
      />
      <SearchInput
        placeholder="Search tests in the specified project"
        onInputChange={handleSearchQueryChange}
        value={pendingSearchQuery}
      />
      <Box sx={{ mt: 5 }}>
        <TestList searchQuery={searchQuery} project={selectedProject} />
      </Box>
    </Box>
  );
};

export const element = (
  // See the documentation for `<LoginPage />` for why we handle error this way.
  <RecoverableErrorBoundary key="test-search">
    <TestSearch />
  </RecoverableErrorBoundary>
);
