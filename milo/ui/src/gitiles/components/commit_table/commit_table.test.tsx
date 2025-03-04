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

import { act, cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import {
  CommitTreeDiffChangeType,
  GitCommit,
} from '@/common/services/milo_internal';
import { FakeContextProvider } from '@/testing_tools/fakes/fake_context_provider';

import { CommitTable } from './commit_table';
import { CommitTableBody } from './commit_table_body';
import { CommitTableHead } from './commit_table_head';
import { CommitTableRow } from './commit_table_row';

const commit: GitCommit = {
  id: '1234567890abcdef',
  tree: '1234567890abcdef',
  parents: ['1234567890abcdee'],
  author: {
    name: 'author',
    email: 'author@email.com',
    time: '2022-02-02T23:22:22Z',
  },
  committer: {
    name: 'committer',
    email: 'committer@email.com',
    time: '2022-02-02T23:22:22Z',
  },
  message: 'this is a commit\ndescription\n',
  treeDiff: [
    {
      type: CommitTreeDiffChangeType.Modify,
      oldId: '1234567890abcdef',
      oldMode: 33188,
      oldPath: 'ash/style/combobox.cc',
      newId: '1234567890abcdef',
      newMode: 33188,
      newPath: 'ash/style/combobox.cc',
    },
  ],
};

describe('CommitTable', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    cleanup();
  });

  it('should expand/collapse correctly', async () => {
    render(
      <FakeContextProvider>
        <CommitTable repoUrl="https://repo.url">
          <CommitTableHead>
            <></>
          </CommitTableHead>
          <CommitTableBody>
            <CommitTableRow commit={commit}>
              <></>
            </CommitTableRow>
          </CommitTableBody>
        </CommitTable>
      </FakeContextProvider>,
    );

    const toggleRowButton = screen.getByLabelText('toggle-row');
    const toggleAllRowsButton = screen.getByLabelText('toggle-all-rows');
    const contentRow = screen.getByTestId('content-row');

    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });

    // Expand by clicking on toggle button.
    act(() => toggleRowButton.click());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).not.toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).toBeInTheDocument();
    expect(contentRow).not.toHaveStyle({ display: 'none' });

    // Collapse by clicking on toggle button.
    act(() => toggleRowButton.click());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });

    // Expand again by changing the default state.
    act(() => toggleAllRowsButton.click());
    await act(() => jest.runAllTimersAsync());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).not.toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).toBeInTheDocument();
    expect(contentRow).not.toHaveStyle({ display: 'none' });

    // Collapse again by changing the default state.
    act(() => toggleAllRowsButton.click());
    await act(() => jest.runAllTimersAsync());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });
  });

  it('should expand/collapse correctly with hotkey', async () => {
    render(
      <FakeContextProvider>
        <CommitTable repoUrl="https://repo.url">
          <CommitTableHead toggleExpandHotkey="x">
            <></>
          </CommitTableHead>
          <CommitTableBody>
            <CommitTableRow commit={commit}>
              <></>
            </CommitTableRow>
          </CommitTableBody>
        </CommitTable>
      </FakeContextProvider>,
    );

    const toggleRowButton = screen.getByLabelText('toggle-row');
    const contentRow = screen.getByTestId('content-row');

    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });

    // Expand by clicking on toggle button.
    act(() => toggleRowButton.click());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).not.toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).toBeInTheDocument();
    expect(contentRow).not.toHaveStyle({ display: 'none' });

    // Collapse by clicking on toggle button.
    act(() => toggleRowButton.click());
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });

    // Expand again by changing the default state.
    await act(() =>
      Promise.all([userEvent.keyboard('x'), jest.runAllTimersAsync()]),
    );
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).not.toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).toBeInTheDocument();
    expect(contentRow).not.toHaveStyle({ display: 'none' });

    // Collapse again by changing the default state.
    await act(() =>
      Promise.all([userEvent.keyboard('x'), jest.runAllTimersAsync()]),
    );
    expect(
      toggleRowButton.querySelector("[data-testid='ChevronRightIcon']"),
    ).toBeInTheDocument();
    expect(
      toggleRowButton.querySelector("[data-testid='ExpandMoreIcon']"),
    ).not.toBeInTheDocument();
    expect(contentRow).toHaveStyle({ display: 'none' });
  });

  it('should notify default state update correctly', async () => {
    const onExpandSpy = jest.fn((_expanded: boolean) => {});
    render(
      <FakeContextProvider>
        <CommitTable
          initDefaultExpanded={true}
          onDefaultExpandedChanged={onExpandSpy}
          repoUrl="https://repo.url"
        >
          <CommitTableHead>
            <></>
          </CommitTableHead>
          <CommitTableBody>
            <></>
          </CommitTableBody>
        </CommitTable>
      </FakeContextProvider>,
    );
    await act(() => jest.runAllTimersAsync());

    // Don't notify on first render.
    expect(onExpandSpy).not.toHaveBeenCalled();

    const toggleButton = screen.getByLabelText('toggle-all-rows');

    // Collapse by clicking on toggle button.
    act(() => toggleButton.click());
    expect(onExpandSpy).toHaveBeenNthCalledWith(1, false);

    // Expand by clicking on toggle button.
    act(() => toggleButton.click());
    expect(onExpandSpy).toHaveBeenNthCalledWith(2, true);
  });
});
