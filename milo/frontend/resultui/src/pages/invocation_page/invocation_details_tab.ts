// Copyright 2020 The LUCI Authors.
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

import { MobxLitElement } from '@adobe/lit-mobx';
import '@material/mwc-icon';
import { css, customElement } from 'lit-element';
import { html } from 'lit-html';
import { styleMap } from 'lit-html/directives/style-map';
import { computed } from 'mobx';

import { router } from '../../routes';
import { consumePageState, InvocationPageState } from './context';


function stripInvocationPrefix(invocationName: string): string {
  return invocationName.slice('invocations/'.length);
}

export class InvocationDetailsTabElement extends MobxLitElement {
  pageState!: InvocationPageState;

  @computed
  private get hasIncludedInvocations() {
    return (this.pageState.invocation!.includedInvocations || []).length > 0;
  }
  @computed
  private get hasTags() {
    return (this.pageState.invocation!.tags || []).length > 0;
  }

  connectedCallback() {
    super.connectedCallback();
    this.pageState.selectedTabId = 'invocation-details';
  }

  protected render() {
    const invocation = this.pageState.invocation;
    if (invocation === null) {
      return html``;
    }
    return html`
      <div>Create Time: ${new Date(invocation.createTime).toLocaleString()}</div>
      <div>Finalize Time: ${new Date(invocation.finalizeTime).toLocaleDateString()}</div>
      <div>Deadline: ${new Date(invocation.deadline).toLocaleDateString()}</div>
      <div
        id="included-invocations"
        style=${styleMap({'display': this.hasIncludedInvocations ? '' : 'none'})}
      >Included Invocations:
        <ul>
        ${invocation.includedInvocations?.map((invName) => stripInvocationPrefix(invName)).map((invId) => html`
          <li><a
            href=${router.urlForName(
              'invocation',
              {'invocation_id': invId},
            )}
            target="_blank"
          >${invId}</a></li>
        `)}
        </ul>
      </div>
      <div style=${styleMap({'display': this.hasTags ? '' : 'none'})}>Tags:
        <table id="tag-table" border="0">
        ${invocation.tags?.map((tag) => html`
          <tr>
            <td>${tag.key}:</td>
            <td>${tag.value}</td>
          </tr>
        `)}
        </table>
      </div>
    `;
  }

  static styles = css`
    :host {
      padding: 5px 24px;
    }

    #included-invocations ul {
      list-style-type: none;
      margin-block-start: auto;
      margin-block-end: auto;
      padding-inline-start: 32px;
    }

    #tag-table {
      margin-left: 29px;
    }
  `;
}

customElement('tr-invocation-details-tab')(
  consumePageState(InvocationDetailsTabElement),
);
