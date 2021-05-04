// Copyright 2021 The LUCI Authors.
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

import { aTimeout, fixture, fixtureCleanup, html } from '@open-wc/testing/index-no-side-effects';
import { css, customElement, LitElement, property } from 'lit-element';

import './build_step_entry';
import { UserConfigsStore } from '../../context/user_configs';
import { provider } from '../../libs/context';
import { IntersectionNotifier, provideNotifier } from '../../libs/observer_element';
import { StepExt } from '../../models/step_ext';
import { BuildStatus } from '../../services/buildbucket';

@customElement('milo-build-step-entry-test-notifier-provider')
@provider
class NotifierProviderElement extends LitElement {
  @property()
  @provideNotifier
  notifier = new IntersectionNotifier({ root: this });

  protected render() {
    return html`<slot></slot>`;
  }

  static styles = css`
    :host {
      display: block;
      height: 100px;
      overflow-y: auto;
    }
  `;
}

describe('build_step_entry', () => {
  const configsStore = new UserConfigsStore();

  it('can render a step without start time', async () => {
    const step = new StepExt({
      name: 'stepname',
      status: BuildStatus.Scheduled,
      startTime: undefined,
    });
    await fixture<NotifierProviderElement>(html`
      <milo-build-step-entry-test-notifier-provider>
        <milo-build-step-entry .configsStore=${configsStore} .step=${step}></milo-build-step-entry>
      </milo-build-step-entry-test-notifier-provider>
    `);
    await aTimeout(10);

    after(fixtureCleanup);
  });
});
