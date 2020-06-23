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
import { css, customElement, html } from 'lit-element';
import { observable } from 'mobx';
import { Artifact } from '../services/resultdb';

const enum ViewOption {
  Expected,
  Actual,
  Diff,
  Animated,
  SideBySide,
}

const VIEW_OPTION_CLASS_MAP = Object.freeze({
  [ViewOption.Expected]: 'expected',
  [ViewOption.Actual]: 'actual',
  [ViewOption.Diff]: 'diff',
  [ViewOption.Animated]: 'animated',
  [ViewOption.SideBySide]: 'side-by-side',
});

/**
 * Renders an image diff artifact set, including expected image, actual image
 * and image diff.
 */
// TODO(weiweilin): improve error handling.
@customElement('tr-image-diff-viewer')
export class ImageDiffViewerElement extends MobxLitElement {
  @observable.ref expected!: Artifact;
  @observable.ref actual!: Artifact;
  @observable.ref diff!: Artifact;

  @observable.ref private viewOption = ViewOption.Animated;

  protected render() {
    return html`
      <div id="options">
        <input
          type="radio"
          name="view-option"
          id="expected"
          @change=${() => this.viewOption = ViewOption.Expected}
          ?checked=${this.viewOption === ViewOption.Expected}
        >
        <label for="expected">Expected</label>
        <input
          type="radio"
          name="view-option"
          id="actual"
          @change=${() => this.viewOption = ViewOption.Actual}
          ?checked=${this.viewOption === ViewOption.Actual}
        >
        <label for="actual">Actual</label>
        <input
          type="radio"
          name="view-option"
          id="diff"
          @change=${() => this.viewOption = ViewOption.Diff}
          ?checked=${this.viewOption === ViewOption.Diff}
        >
        <label for="diff">Diff</label>
        <input
          type="radio"
          name="view-option"
          id="animated"
          @change=${() => this.viewOption = ViewOption.Animated}
          ?checked=${this.viewOption === ViewOption.Animated}
        >
        <label for="animated">Animated</label>
        <input
          type="radio"
          name="view-option"
          id="side-by-side"
          @change=${() => this.viewOption = ViewOption.SideBySide}
          ?checked=${this.viewOption === ViewOption.SideBySide}
        >
        <label for="side-by-side">Side by side</label>
      </div>
      <div id="content" class=${VIEW_OPTION_CLASS_MAP[this.viewOption]}>
        <div id="expected-image" class="image">
          <div>Expected (<a href=${this.expected.fetchUrl} target="_blank">view raw</a>)</div>
          <img src=${this.expected.fetchUrl}>
        </div>
        <div id="actual-image" class="image">
          <div>Actual (<a href=${this.actual.fetchUrl} target="_blank">view raw</a>)</div>
          <img src=${this.actual.fetchUrl}>
        </div>
        <div id="diff-image" class="image">
          <div>Diff (<a href=${this.diff.fetchUrl} target="_blank">view raw</a>)</div>
          <img src=${this.diff.fetchUrl}>
        </div>
      </div>
    `;
  }

  static styles = css`
    :host {
      display: block;
    }

    #options {
      margin: 10px;
    }
    #options > label {
      margin-right: 5px;
    }
    .raw-link:not(:last-child):after {
      content: ','
    }

    #content {
      white-space: nowrap;
      overflow-x: auto;
      padding: 20px;
      position: relative;
      top: 0;
      left: 0;
    }
    .image {
      display: none;
    }

    .expected #expected-image {
      display: block;
    }
    .actual #actual-image {
      display: block;
    }
    .diff #diff-image {
      display: block;
    }

    .animated .image {
      animation-name: blink;
      animation-duration: 2s;
      animation-timing-function: steps(1);
      animation-iteration-count: infinite;
    }
    .animated #expected-image {
      display: block;
      position: absolute;
      animation-delay: -1s;
    }
    .animated #actual-image {
      display: block;
      position: static;
      animation-direction: normal;
    }
    @keyframes blink {
      0% { opacity: 1; }
      50% { opacity: 0; }
    }

    .side-by-side .image {
      display: inline-block;
    }
  `;
}
