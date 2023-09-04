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

import { useQuery } from '@tanstack/react-query';
import { act, cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ANONYMOUS_IDENTITY } from '@/common/api/auth_state';
import { useAuthState } from '@/common/components/auth_state_provider';
import { FakeAuthStateProvider } from '@/testing_tools/fakes/fake_auth_state_provider';
import { FakeContextProvider } from '@/testing_tools/fakes/fake_context_provider';

import { RecoverableErrorBoundary } from './recoverable_error_boundary';

const SILENCED_ERROR_MAGIC_STRING = ' <cdaac21>';

function IdentityTestComponent() {
  const { identity } = useAuthState();

  const { data, error } = useQuery({
    queryKey: ['test-key', identity],
    queryFn: () => {
      if (identity === ANONYMOUS_IDENTITY) {
        throw new Error('cannot be anonymous' + SILENCED_ERROR_MAGIC_STRING);
      }
      return `Hello ${identity}`;
    },
  });

  if (error) {
    throw error;
  }

  return <>{data}</>;
}

describe('RecoverableErrorBoundary', () => {
  // eslint-disable-next-line no-console
  const logErr = console['error'];
  beforeEach(() => {
    jest.useFakeTimers();
    // Silence error related to the error we thrown. Note that the following
    // method isn't able to silence the error from React for some reason.
    // ```
    // jest.spyOn(console, 'error').mockImplementation(() => {});
    // ```
    // eslint-disable-next-line no-console
    console.error = function (err) {
      const errStr = `${err}`;
      if (
        errStr.includes(SILENCED_ERROR_MAGIC_STRING) ||
        errStr.includes(
          'React will try to recreate this component tree from scratch using the error boundary you provided',
        )
      ) {
        return;
      }
      logErr(err);
    };
  });

  afterEach(() => {
    jest.useRealTimers();
    // eslint-disable-next-line no-console
    console['error'] = logErr;
    cleanup();
  });

  it('can recover from error when user identity changes', async () => {
    const { rerender } = render(
      <FakeContextProvider>
        <FakeAuthStateProvider value={{ identity: ANONYMOUS_IDENTITY }}>
          <RecoverableErrorBoundary>
            <IdentityTestComponent />
          </RecoverableErrorBoundary>
        </FakeAuthStateProvider>
      </FakeContextProvider>,
    );

    await act(() => jest.runAllTimersAsync());

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('cannot be anonymous');

    rerender(
      <FakeContextProvider>
        <FakeAuthStateProvider value={{ identity: 'user:user@google.com' }}>
          <RecoverableErrorBoundary>
            <IdentityTestComponent />
          </RecoverableErrorBoundary>
        </FakeAuthStateProvider>
      </FakeContextProvider>,
    );
    await act(() => jest.runAllTimersAsync());

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getByText('Hello user:user@google.com')).toBeInTheDocument();
  });

  it('can recover from error when retry button is clicked', async () => {
    let shouldThrowError = true;

    function RetryTestComponent() {
      const { data, error } = useQuery({
        queryKey: ['test-key'],
        queryFn: () => {
          if (shouldThrowError) {
            throw new Error(
              'encountered an error' + SILENCED_ERROR_MAGIC_STRING,
            );
          }
          return `No error`;
        },
      });

      if (error) {
        throw error;
      }

      return <>{data}</>;
    }

    const { rerender } = render(
      <FakeContextProvider>
        <RecoverableErrorBoundary>
          <RetryTestComponent />
        </RecoverableErrorBoundary>
      </FakeContextProvider>,
    );

    await act(() => jest.runAllTimersAsync());

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('encountered an error');

    shouldThrowError = false;

    rerender(
      <FakeContextProvider>
        <RecoverableErrorBoundary>
          <RetryTestComponent />
        </RecoverableErrorBoundary>
      </FakeContextProvider>,
    );
    await act(() => jest.runAllTimersAsync());

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('encountered an error');

    userEvent.click(screen.getByText('Try Again'));
    await act(() => jest.runAllTimersAsync());
    await act(() => jest.runAllTimersAsync());

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getByText('No error')).toBeInTheDocument();
  });
});
