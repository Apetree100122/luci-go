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

import '@testing-library/jest-dom';

import { render, screen } from '@testing-library/react';

import { PageMetaContext } from '@/common/components/page_meta/page_meta_provider';
import { UiPage } from '@/common/constants';
import { FakeContextProvider } from '@/testing_tools/fakes/fake_context_provider';

import { AppDetails } from './app_details';

describe('AppDetails', () => {
  it('should display app name given no data', async () => {
    render(
      <PageMetaContext.Provider
        value={{
          setProject: () => {},
          setSelectedPage: () => {},
        }}
      >
        <AppDetails open={true} setSidebarOpen={() => {}} />
      </PageMetaContext.Provider>
    );
    await screen.findByLabelText('menu');
    expect(screen.getByText('LUCI')).toBeInTheDocument();
  });

  it('should display selected page', async () => {
    render(
      <FakeContextProvider
        pageMeta={{
          selectedPage: UiPage.Builders,
        }}
      >
        <AppDetails open={true} setSidebarOpen={() => {}} />
      </FakeContextProvider>
    );
    await screen.findByLabelText('menu');
    expect(screen.getByText('LUCI')).toBeInTheDocument();
    expect(screen.getByText(UiPage.Builders)).toBeInTheDocument();
  });

  it('should display project', async () => {
    render(
      <FakeContextProvider
        pageMeta={{
          selectedPage: UiPage.Builders,
          project: 'chrome',
        }}
      >
        <AppDetails open={true} setSidebarOpen={() => {}} />
      </FakeContextProvider>
    );
    await screen.findByLabelText('menu');
    expect(screen.getByText('LUCI')).toBeInTheDocument();
    expect(screen.getByText(UiPage.Builders)).toBeInTheDocument();
    expect(screen.getByText('chrome')).toBeInTheDocument();
  });
});
