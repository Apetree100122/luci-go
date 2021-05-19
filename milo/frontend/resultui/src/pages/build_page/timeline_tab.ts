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

import * as d3 from 'd3';
import { css, customElement, html, property } from 'lit-element';
import { autorun, observable } from 'mobx';

import '../../components/dot_spinner';
import { MiloBaseElement } from '../../components/milo_base';
import { AppState, consumeAppState } from '../../context/app_state';
import { BuildState, consumeBuildState } from '../../context/build_state';
import { GA_ACTIONS, GA_CATEGORIES, trackEvent } from '../../libs/analytics_utils';
import { BUILD_STATUS_CLASS_MAP } from '../../libs/constants';
import { consumer } from '../../libs/context';
import { errorHandler, forwardWithoutMsg, reportError, reportRenderError } from '../../libs/error_handler';
import { StepExt } from '../../models/step_ext';
import commonStyle from '../../styles/common_style.css';

const MARGIN = 10;
const AXIS_HEIGHT = 20;
const BORDER_SIZE = 1;
const HALF_BORDER_SIZE = BORDER_SIZE / 2;

const ROW_HEIGHT = 30;
const STEP_HEIGHT = 24;
const STEP_MARGIN = (ROW_HEIGHT - STEP_HEIGHT) / 2 - HALF_BORDER_SIZE;
const STEP_EXTRA_WIDTH = 2;

const TEXT_HEIGHT = 10;
const STEP_TEXT_OFFSET = ROW_HEIGHT / 2 + TEXT_HEIGHT / 2;
const TEXT_MARGIN = 10;

const SIDE_PANEL_WIDTH = 400;
const MIN_GRAPH_WIDTH = 500 + SIDE_PANEL_WIDTH;
const SIDE_PANEL_RECT_WIDTH = SIDE_PANEL_WIDTH - STEP_MARGIN * 2 - BORDER_SIZE * 2;
const STEP_IDENT = 15;

const LIST_ITEM_WIDTH = SIDE_PANEL_RECT_WIDTH - TEXT_MARGIN * 2;
const LIST_ITEM_HEIGHT = 16;
const LIST_ITEM_X_OFFSET = STEP_MARGIN + TEXT_MARGIN + BORDER_SIZE;
const LIST_ITEM_Y_OFFSET = STEP_MARGIN + (STEP_HEIGHT - LIST_ITEM_HEIGHT) / 2;

const V_GRID_LINE_MAX_GAP = 80;
const PREDEFINED_TIME_INTERVALS = [
  // Values that can divide 1 day.
  86400000, // 24hr
  43200000, // 12hr
  28800000, // 8hr
  // Values that can divide 12 hours.
  21600000, // 6hr
  14400000, // 4hr
  10800000, // 3hr
  7200000, // 2hr
  3600000, // 1hr
  // Values that can divide 1 hour.
  1800000, // 30min
  1200000, // 20min
  900000, // 15min
  600000, // 10min
  // Values that can divide 15 minutes.
  300000, // 5min
  180000, // 3min
  120000, // 2min
  60000, // 1min
  // Values that can divide 1 minute.
  30000, // 30s
  20000, // 20s
  15000, // 15s
  10000, // 10s
  // Values that can divide 15 seconds.
  5000, // 5s
  3000, // 3s
  2000, // 2s
  1000, // 1s
];

/**
 * A utility function that helps assigning appropriate list numbers to steps.
 * For example, if a step is the 1st child of the 2nd root step, the list number
 * would be '2.1. '.
 *
 * @param steps a list of steps.
 * @yields A tuple consist of the index, the list number, and the step.
 */
function* traverseStepList(steps: readonly StepExt[]): IterableIterator<[number, string, StepExt]> {
  const ancestorNums: number[] = [];
  for (const [i, step] of steps.entries()) {
    ancestorNums.splice(step.depth, ancestorNums.length, step.index + 1);
    yield [i, ancestorNums.join('.') + '. ', step];
  }
}

@customElement('milo-timeline-tab')
@errorHandler(forwardWithoutMsg)
@consumer
export class TimelineTabElement extends MiloBaseElement {
  @observable.ref
  @consumeAppState
  appState!: AppState;

  @observable.ref
  @consumeBuildState
  buildState!: BuildState;

  @observable.ref private totalWidth!: number;
  @observable.ref private bodyWidth!: number;

  // Don't set them as observable. When render methods update them, we don't
  // want autorun to trigger this.renderTimeline() again.
  @property() private headerEle!: HTMLDivElement;
  @property() private footerEle!: HTMLDivElement;
  @property() private sidePanelEle!: HTMLDivElement;
  @property() private bodyEle!: HTMLDivElement;

  // Properties shared between render methods.
  private bodyHeight!: number;
  private scaleTime!: d3.ScaleTime<number, number, never>;
  private scaleStep!: d3.ScaleLinear<number, number, never>;
  private timeInterval!: d3.TimeInterval;
  private readonly now = Date.now();

  connectedCallback() {
    super.connectedCallback();
    this.appState.selectedTabId = 'timeline';
    trackEvent(GA_CATEGORIES.TIMELINE_TAB, GA_ACTIONS.TAB_VISITED, window.location.href);

    const syncWidth = () => {
      this.totalWidth = Math.max(window.innerWidth - 2 * MARGIN, MIN_GRAPH_WIDTH);
      this.bodyWidth = this.totalWidth - SIDE_PANEL_WIDTH;
    };
    window.addEventListener('resize', syncWidth);
    this.addDisposer(() => window.removeEventListener('resize', syncWidth));
    syncWidth();

    this.addDisposer(autorun(() => this.renderTimeline()));
  }

  protected render = reportRenderError.bind(this)(() => {
    if (!this.buildState.build) {
      return html`<div id="load">Loading <milo-dot-spinner></milo-load-spinner></div>`;
    }

    if (this.buildState.build.steps.length === 0) {
      return html`<div id="no-steps">No steps were run.</div>`;
    }

    return html`<div id="timeline">${this.sidePanelEle}${this.headerEle}${this.bodyEle}${this.footerEle}</div>`;
  });

  private renderTimeline = reportError.bind(this)(() => {
    const build = this.buildState.build;
    if (!build || build.steps.length === 0) {
      return;
    }

    const startTime = build.startTime?.toMillis() || this.now;
    const endTime = build.endTime?.toMillis() || this.now;

    this.bodyHeight = build.steps.length * ROW_HEIGHT - BORDER_SIZE;
    const padding = Math.ceil(((endTime - startTime) * STEP_EXTRA_WIDTH) / this.bodyWidth) / 2;

    // Calc attributes shared among components.
    this.scaleTime = d3
      .scaleTime()
      // Add a bit of padding to ensure everything renders in the viewport.
      .domain([startTime - padding, endTime + padding])
      // Ensure the right border is rendered within the viewport, while the left
      // border overlaps with the right border of the side-panel.
      .range([-HALF_BORDER_SIZE, this.bodyWidth - HALF_BORDER_SIZE]);
    this.scaleStep = d3
      .scaleLinear()
      .domain([0, build.steps.length])
      // Ensure the top and bottom borders are not rendered.
      .range([-HALF_BORDER_SIZE, this.bodyHeight + HALF_BORDER_SIZE]);

    const maxInterval = (endTime - startTime + 2 * padding) / (this.bodyWidth / V_GRID_LINE_MAX_GAP);

    // Assign a default value here to make TSC happy.
    let interval = PREDEFINED_TIME_INTERVALS[0];

    // Find the largest interval that is no larger than the maximum interval.
    // Use linear search because the array is relatively short.
    for (const predefined of PREDEFINED_TIME_INTERVALS) {
      interval = predefined;
      if (maxInterval >= predefined) {
        break;
      }
    }
    this.timeInterval = d3.timeMillisecond.every(interval)!;

    // Render each component.
    this.renderHeader();
    this.renderFooter();
    this.renderSidePanel();
    this.renderBody();
  });

  private renderHeader() {
    this.headerEle = document.createElement('div');
    const svg = d3
      .select(this.headerEle)
      .attr('id', 'header')
      .append('svg')
      .attr('viewport', `0 0 ${this.totalWidth} ${AXIS_HEIGHT}`);
    const headerRootGroup = svg
      .append('g')
      .attr('transform', `translate(${SIDE_PANEL_WIDTH}, ${AXIS_HEIGHT - HALF_BORDER_SIZE})`);
    const axisTop = d3.axisTop(this.scaleTime).ticks(this.timeInterval);
    headerRootGroup.call(axisTop);

    // Top border for the side panel.
    headerRootGroup.append('line').attr('x1', -SIDE_PANEL_WIDTH).attr('stroke', 'var(--default-text-color)');
  }

  private renderFooter() {
    this.footerEle = document.createElement('div');
    const svg = d3
      .select(this.footerEle)
      .attr('id', 'footer')
      .append('svg')
      .attr('viewport', `0 0 ${this.totalWidth} ${AXIS_HEIGHT}`);
    const footerRootGroup = svg.append('g').attr('transform', `translate(${SIDE_PANEL_WIDTH}, ${HALF_BORDER_SIZE})`);
    const axisBottom = d3.axisBottom(this.scaleTime).ticks(this.timeInterval);
    footerRootGroup.call(axisBottom);

    // Bottom border for the side panel.
    footerRootGroup.append('line').attr('x1', -SIDE_PANEL_WIDTH).attr('stroke', 'var(--default-text-color)');
  }

  private renderSidePanel() {
    const build = this.buildState.build!;

    this.sidePanelEle = document.createElement('div');
    const svg = d3
      .select(this.sidePanelEle)
      .style('width', SIDE_PANEL_WIDTH + 'px')
      .style('height', this.bodyHeight + 'px')
      .attr('id', 'side-panel')
      .append('svg')
      .attr('viewport', `0 0 ${SIDE_PANEL_WIDTH} ${this.bodyHeight}`);

    // Grid lines
    const horizontalGridLines = d3
      .axisLeft(this.scaleStep)
      .ticks(build.steps.length)
      .tickFormat(() => '')
      .tickSize(-SIDE_PANEL_WIDTH)
      .tickFormat(() => '');
    svg.append('g').attr('class', 'grid').call(horizontalGridLines);

    for (const [i, listNum, step] of traverseStepList(build.steps)) {
      const stepGroup = svg
        .append('g')
        .attr('class', BUILD_STATUS_CLASS_MAP[step.status])
        .attr('transform', `translate(0, ${i * ROW_HEIGHT})`);

      const rect = stepGroup
        .append('rect')
        .attr('x', STEP_MARGIN + BORDER_SIZE)
        .attr('y', STEP_MARGIN)
        .attr('width', SIDE_PANEL_RECT_WIDTH)
        .attr('height', STEP_HEIGHT);

      const listItem = stepGroup
        .append('foreignObject')
        .attr('class', 'not-intractable')
        .attr('x', LIST_ITEM_X_OFFSET + step.depth * STEP_IDENT)
        .attr('y', LIST_ITEM_Y_OFFSET)
        .attr('height', STEP_HEIGHT - LIST_ITEM_Y_OFFSET)
        .attr('width', LIST_ITEM_WIDTH);
      listItem.append('xhtml:span').text(listNum);
      const stepText = listItem.append('xhtml:span').text(step.selfName);

      const logUrl = step.logs?.[0].viewUrl;
      if (logUrl) {
        rect.attr('class', 'clickable').on('click', (e: MouseEvent) => {
          e.stopPropagation();
          window.open(logUrl, '_blank');
        });
        stepText.attr('class', 'hyperlink');
      }
    }

    // Left border.
    svg
      .append('line')
      .attr('x1', HALF_BORDER_SIZE)
      .attr('x2', HALF_BORDER_SIZE)
      .attr('y2', this.bodyHeight)
      .attr('stroke', 'var(--default-text-color)');
    // Right border.
    svg
      .append('line')
      .attr('x1', SIDE_PANEL_WIDTH - HALF_BORDER_SIZE)
      .attr('x2', SIDE_PANEL_WIDTH - HALF_BORDER_SIZE)
      .attr('y2', this.bodyHeight)
      .attr('stroke', 'var(--default-text-color)');
  }

  private renderBody() {
    const build = this.buildState.build!;

    this.bodyEle = document.createElement('div');
    const svg = d3
      .select(this.bodyEle)
      .attr('id', 'body')
      .style('width', this.bodyWidth + 'px')
      .style('height', this.bodyHeight + 'px')
      .append('svg')
      .attr('viewport', `0 0 ${this.bodyWidth} ${this.bodyHeight}`);

    // Grid lines
    const verticalGridLines = d3
      .axisTop(this.scaleTime)
      .ticks(this.timeInterval)
      .tickSize(-this.bodyHeight)
      .tickFormat(() => '');
    svg.append('g').attr('class', 'grid').call(verticalGridLines);
    const horizontalGridLines = d3
      .axisLeft(this.scaleStep)
      .ticks(build.steps.length)
      .tickFormat(() => '')
      .tickSize(-this.bodyWidth)
      .tickFormat(() => '');
    svg.append('g').attr('class', 'grid').call(horizontalGridLines);

    for (const [i, listNum, step] of traverseStepList(build.steps)) {
      const start = this.scaleTime(step.startTime?.toMillis() || this.now);
      const end = this.scaleTime(step.endTime?.toMillis() || this.now);

      const stepGroup = svg
        .append('g')
        .attr('class', BUILD_STATUS_CLASS_MAP[step.status])
        .attr('transform', `translate(${start}, ${i * ROW_HEIGHT})`);

      // Add extra width so tiny steps are visible.
      const width = end - start + STEP_EXTRA_WIDTH;

      const rect = stepGroup
        .append('rect')
        .attr('x', -STEP_EXTRA_WIDTH / 2)
        .attr('y', STEP_MARGIN)
        .attr('width', width)
        .attr('height', STEP_HEIGHT);

      const isWide = width > this.bodyWidth * 0.33;
      const nearEnd = end > this.bodyWidth * 0.66;

      const stepText = stepGroup
        .append('text')
        .attr('text-anchor', isWide || !nearEnd ? 'start' : 'end')
        .attr('x', isWide ? TEXT_MARGIN : nearEnd ? -TEXT_MARGIN : width + TEXT_MARGIN)
        .attr('y', STEP_TEXT_OFFSET)
        .text(listNum + step.selfName);

      const logUrl = step.logs?.[0].viewUrl;
      if (logUrl) {
        const onClick = (e: MouseEvent) => {
          e.stopPropagation();
          window.open(logUrl, '_blank');
        };
        rect.attr('class', 'clickable').on('click', onClick);

        // Wail until the next event cycle so stepText is rendered when we call
        // this.getBBox();
        window.setTimeout(() => {
          stepText.each(function () {
            const bBox = this.getBBox();

            // This makes the step text easier to click.
            stepGroup
              .append('rect')
              .attr('x', bBox.x)
              .attr('y', bBox.y)
              .attr('width', bBox.width)
              .attr('height', bBox.height)
              .attr('class', 'clickable invisible')
              .on('click', onClick);
          });
        });
      }
    }

    // Right border.
    svg
      .append('line')
      .attr('x1', this.bodyWidth - HALF_BORDER_SIZE)
      .attr('x2', this.bodyWidth - HALF_BORDER_SIZE)
      .attr('y2', this.bodyHeight)
      .attr('stroke', 'var(--default-text-color)');
  }

  static styles = [
    commonStyle,
    css`
      :host {
        display: block;
        margin: ${MARGIN}px;
      }

      #load {
        color: var(--active-text-color);
      }

      #timeline {
        display: grid;
        grid-template-rows: ${AXIS_HEIGHT}px 1fr ${AXIS_HEIGHT}px;
        grid-template-columns: ${SIDE_PANEL_WIDTH}px 1fr;
        grid-template-areas:
          'header header'
          'side-panel body'
          'footer footer';
      }

      #header {
        grid-area: header;
        position: sticky;
        top: 0;
        background: white;
        z-index: 2;
      }

      #footer {
        grid-area: footer;
        position: sticky;
        bottom: 0;
        background: white;
        z-index: 2;
      }

      #side-panel {
        grid-area: side-panel;
        z-index: 1;
        font-weight: 500;
      }

      #body {
        grid-area: body;
      }

      #body path.domain {
        stroke: none;
      }

      svg {
        width: 100%;
        height: 100%;
      }

      text {
        fill: var(--default-text-color);
      }

      .grid line {
        stroke: var(--divider-color);
      }

      .clickable {
        cursor: pointer;
      }
      .not-intractable {
        pointer-events: none;
      }
      .hyperlink {
        text-decoration: underline;
      }

      .scheduled > rect {
        stroke: var(--scheduled-color);
        fill: var(--scheduled-bg-color);
      }
      .started > rect {
        stroke: var(--started-color);
        fill: var(--started-bg-color);
      }
      .success > rect {
        stroke: var(--success-color);
        fill: var(--success-bg-color);
      }
      .failure > rect {
        stroke: var(--failure-color);
        fill: var(--failure-bg-color);
      }
      .infra-failure > rect {
        stroke: var(--critical-failure-color);
        fill: var(--critical-failure-bg-color);
      }
      .canceled > rect {
        stroke: var(--canceled-color);
        fill: var(--canceled-bg-color);
      }

      .invisible {
        opacity: 0;
      }
    `,
  ];
}
