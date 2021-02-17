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

import { MobxLitElement } from '@adobe/lit-mobx';
import { css, customElement, html, TemplateResult } from 'lit-element';
import { classMap } from 'lit-html/directives/class-map';
import { styleMap } from 'lit-html/directives/style-map';
import { computed, observable, reaction } from 'mobx';

export type Suggestion = SuggestionEntry | SuggestionHeader;

export interface SuggestionEntry {
  readonly isHeader?: false;
  readonly value: string;
  // If display is undefined, value is used.
  readonly display?: string | TemplateResult;
  readonly explanation: string | TemplateResult;
}

export interface SuggestionHeader {
  readonly isHeader: true;
  readonly value?: '';
  readonly display: string | TemplateResult;
  readonly explanation?: '';
}

/**
 * An input box that supports auto-complete dropdown.
 */
@customElement('milo-auto-complete')
export class AutoCompleteElement extends MobxLitElement {
  @observable.ref value = '';
  @observable.ref placeHolder = '';
  @observable.ref suggestions: readonly Suggestion[] = [];

  onValueUpdate = (_newVal: string) => {};
  onSuggestionSelected = (_suggestion: SuggestionEntry) => {};

  focus() {
    this.searchBox.focus();
  }

  // -1 means nothing is selected.
  @observable.ref private selectedIndex = -1;
  @observable.ref private showSuggestions = false;
  @observable.ref private focused = false;

  private get searchBox() {
    return this.shadowRoot!.getElementById('search-box')!;
  }
  private get dropdownContainer() {
    return this.shadowRoot!.getElementById('dropdown-container')!;
  }
  @computed private get hint() {
    if (this.focused && this.suggestions.length > 0) {
      if (this.showSuggestions) {
        return 'Use ↑ and ↓ to select, ⏎ to confirm, esc to dismiss suggestions';
      } else {
        return 'Press ↓ to see suggestions';
      }
    }
    return this.placeHolder;
  }

  protected updated() {
    this.shadowRoot!.querySelector('.dropdown-item.selected')?.scrollIntoView({'block': 'nearest'});
  }

  private disposer = () => {};
  connectedCallback() {
    super.connectedCallback();

    // Reset suggestion state when suggestions are updated.
    this.disposer = reaction(
      () => this.suggestions,
      () => {
        this.selectedIndex = -1;
        if (this.value !== '') {
          this.showSuggestions = true;
        }
      },
    );

    document.addEventListener('click', this.externalClickHandler);
  }

  disconnectedCallback() {
    document.removeEventListener('click', this.externalClickHandler);
    this.disposer();
    super.disconnectedCallback();
  }

  private clearSuggestion() {
    this.showSuggestions = false;
    this.selectedIndex = -1;
  }

  private externalClickHandler = (e: MouseEvent) => {
    // If user clicks on other elements, dismiss the dropdown.
    if (!e.composedPath().some((t) => t === this.searchBox || t === this.dropdownContainer)) {
      this.clearSuggestion();
    }
  }

  private renderSuggestion(suggestion: Suggestion, suggestionIndex: number) {
    if (suggestion.isHeader) {
      return html`
        <tr class="dropdown-item header">
          <td colspan="2">${suggestion.display}</td>
        </tr>
      `;
    }
    return html`
      <tr
        class=${classMap({'dropdown-item': true, 'selected': suggestionIndex === this.selectedIndex})}
        @mouseover=${() => this.selectedIndex = suggestionIndex}
        @click=${() => {
          this.onSuggestionSelected(this.suggestions[this.selectedIndex] as SuggestionEntry);
          this.focus();
        }}
      >
        <td>${suggestion.display ?? suggestion.value}</td>
        <td>${suggestion.explanation}</td>
      </tr>
    `;
  }

  protected render() {
    return html`
      <input
        id="search-box"
        placeholder=${this.hint}
        .value=${this.value}
        @input=${(e: InputEvent) => this.onValueUpdate((e.target as HTMLInputElement).value)}
        @focus=${() => this.focused = true}
        @blur=${() => this.focused = false}
        @keydown=${(e: KeyboardEvent) => {
          switch (e.code) {
            case 'ArrowDown':
              if (!this.showSuggestions) {
                this.showSuggestions = true;
              }
              // Select the next suggestion entry.
              for (let nextIndex = this.selectedIndex + 1; nextIndex < this.suggestions.length; ++nextIndex) {
                if (!this.suggestions[nextIndex].isHeader) {
                  this.selectedIndex = nextIndex;
                  break;
                }
              }
              break;
            case 'ArrowUp':
              // Select the previous suggestion entry.
              for (let nextIndex = this.selectedIndex - 1; nextIndex >= 0; --nextIndex) {
                if (!this.suggestions[nextIndex].isHeader) {
                  this.selectedIndex = nextIndex;
                  break;
                }
              }
              break;
            case 'Escape':
              this.clearSuggestion();
              break;
            case 'Enter':
              if (this.selectedIndex !== -1) {
                this.onSuggestionSelected(this.suggestions[this.selectedIndex] as SuggestionEntry);
              } else if (this.value !== '' && !this.value.endsWith(' ')) {
                // Complete the current sub-query if it's not already completed.
                this.onValueUpdate(this.value + ' ');
              }
              this.clearSuggestion();
              break;
            default:
              return;
          }
          e.preventDefault();
        }}
      >
      <div id="dropdown-container" style=${styleMap({display: this.showSuggestions && this.suggestions.length > 0 ? '' : 'none'})}>
        <table id="dropdown">
          ${this.suggestions.map((suggestion, i) => this.renderSuggestion(suggestion, i))}
        </table>
      </div>
    `;
  }

  static styles = css`
    :host {
      display: inline-block;
      position: relative;
      box-sizing: border-box;
      width: 100%;
    }

    #search-box {
      display: inline-block;
      width: 100%;
      box-sizing: border-box;
      padding: .3rem .5rem;
      font-size: 1rem;
      color: var(--light-text-color);
      background-clip: padding-box;
      border: 1px solid var(--divider-color);
      border-radius: .25rem;
      transition: border-color .15s ease-in-out,box-shadow .15s ease-in-out;
      text-overflow: ellipsis;
    }

    #dropdown-container {
      position: absolute;
      border: 1px solid var(--divider-color);
      background: white;
      color: var(--active-color);
      padding: 2px;
      z-index: 999;
      max-height: 200px;
      overflow-y: auto;
    }
    #dropdown {
      border-spacing: 0 1px;
    }

    .dropdown-item.header {
      color: var(--default-text-color);
    }
    .dropdown-item>td {
      white-space: nowrap;
      overflow: hidden;
    }
    .dropdown-item>td:first-child {
      padding-right: 50px;
    }
    .dropdown-item.selected {
      border-color: var(--light-active-color);
      background-color: var(--light-active-color);
    }
  `;
}
