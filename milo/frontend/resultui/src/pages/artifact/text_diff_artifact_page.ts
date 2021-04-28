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
import * as Diff2Html from 'diff2html';
import { css, customElement, html } from 'lit-element';
import { computed, observable } from 'mobx';
import { fromPromise } from 'mobx-utils';

import '../../components/dot_spinner';
import '../../components/status_bar';
import { AppState, consumeAppState } from '../../context/app_state';
import { consumeContext } from '../../libs/context';
import { reportRenderError } from '../../libs/error_handler';
import { sanitizeHTML } from '../../libs/sanitize_html';
import { unwrapObservable } from '../../libs/utils';
import { ArtifactIdentifier, constructArtifactName } from '../../services/resultdb';
import commonStyle from '../../styles/common_style.css';

/**
 * Renders a text diff artifact.
 */
@customElement('milo-text-diff-artifact-page')
@consumeAppState
@consumeContext('artifactIdent')
export class TextDiffArtifactPageElement extends MobxLitElement {
  @observable.ref appState!: AppState;

  @observable.ref artifactIdent!: ArtifactIdentifier;

  @computed
  private get artifact$() {
    if (!this.appState.resultDb) {
      return fromPromise(Promise.race([]));
    }
    return fromPromise(this.appState.resultDb.getArtifact({ name: constructArtifactName(this.artifactIdent) }));
  }
  @computed private get artifact() {
    return unwrapObservable(this.artifact$, null);
  }

  @computed
  private get content$() {
    if (!this.appState.resultDb || !this.artifact) {
      return fromPromise(Promise.race([]));
    }
    // TODO(weiweilin): handle refresh.
    return fromPromise(fetch(this.artifact.fetchUrl).then((res) => res.text()));
  }
  @computed private get content() {
    return unwrapObservable(this.content$, null);
  }

  protected render = reportRenderError.bind(this)(() => {
    if (!this.content) {
      return html`<div id="content" class="active-text">Loading <milo-dot-spinner></milo-dot-spinner></div>`;
    }

    return html`
      <div id="details">
        ${this.artifact?.fetchUrl ? html`<a href=${this.artifact?.fetchUrl}>View Raw Content</a>` : ''}
      </div>
      <div id="content">
        <link
          rel="stylesheet"
          type="text/css"
          href="https://cdn.jsdelivr.net/npm/diff2html/bundles/css/diff2html.min.css"
        />
        ${sanitizeHTML(Diff2Html.html(this.content || '', { drawFileList: false, outputFormat: 'side-by-side' }))}
      </div>
    `;
  });

  static styles = [
    commonStyle,
    css`
      #details {
        margin: 20px;
      }
      #content {
        position: relative;
        margin: 20px;
      }

      .d2h-code-linenumber {
        cursor: default;
      }
      .d2h-moved-tag {
        display: none;
      }
    `,
  ];
}
