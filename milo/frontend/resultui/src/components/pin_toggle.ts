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

import { css, customElement, html, LitElement, property } from 'lit-element';
import { classMap } from 'lit-html/directives/class-map';

/**
 * An icon that indicates whether the item is pinned.
 */
@customElement('milo-pin-toggle')
export class CopyToClipboard extends LitElement {
  @property() pinned = false;

  protected render() {
    return html`
      <svg
        class=${classMap({'pinned': this.pinned})}
        xmlns="http://www.w3.org/2000/svg"
        viewBox="2 ${this.pinned ? -4 : 2} 20 20"
      >
        <path id="pin" d="M16,9V4l1,0c0.55,0,1-0.45,1-1v0c0-0.55-0.45-1-1-1H7C6.45,2,6,2.45,6,3v0 c0,0.55,0.45,1,1,1l1,0v5c0,1.66-1.34,3-3,3h0v2h5.97v7l1,1l1-1v-7H19v-2h0C17.34,12,16,10.66,16,9z"/>
        <path id="floor" d="M3,16h18v-1h-18z"/>
      </svg>
    `;
  }

  static styles = css`
    :host {
      cursor: pointer;
      display: inline-block;
      vertical-align: text-bottom;
      width: 16px;
      height: 16px;
      border-radius: 2px;
      padding: 2px;
    }
    :host(:hover) {
      background-color: silver;
    }

    #floor {
      opacity: 0;
    }
    .pinned>#floor {
      opacity: 1;
    }
  `;
}
