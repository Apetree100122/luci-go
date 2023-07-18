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

import { StringPair } from '@/common/services/common';
import { PrpcClientExt } from '@/generic_libs/tools/prpc_client_ext';

export const enum NullableBool {
  Null = 'NULL',
  False = 'FALSE',
  TRUE = 'TRUE',
}

export interface BotsRequest {
  readonly limit: number;
  readonly cursor?: string;
  readonly dimensions?: readonly StringPair[];
  readonly quarantined?: NullableBool;
  readonly isDead?: NullableBool;
  readonly isBusy?: NullableBool;
}

export interface BotInfo {
  readonly botId: string;
  readonly taskId?: string;
  readonly externalIp: string;
  readonly authenticatedAs: string;
  readonly firstSeenTs: string;
  readonly isDead: boolean;
  readonly lastSeenTs: string;
  readonly quarantined: boolean;
  readonly maintenanceMsg?: string;
  readonly dimensions: readonly StringPair[];
  readonly taskName: string;
  readonly version: string;
  readonly state: string;
  readonly deleted: boolean;
}

export interface BotInfoListResponse {
  readonly cursor?: string;
  readonly items?: readonly BotInfo[];
  readonly now: string;
  readonly deathTimeout: number;
}

export class BotsService {
  static readonly SERVICE = 'swarming.v2.Bots';

  private readonly callFn: (
    method: string,
    message: object
  ) => Promise<unknown>;

  constructor(client: PrpcClientExt) {
    this.callFn = (method: string, message: object) =>
      client.call(BotsService.SERVICE, method, message);
  }

  async listBots(req: BotsRequest) {
    return (await this.callFn('ListBots', req)) as BotInfoListResponse;
  }
}

export enum BotStatus {
  Idle,
  Busy,
  Quarantined,
  Dead,
  Deleted,
}

/**
 * Computes bot status.
 */
export function getBotStatus(bot: BotInfo): BotStatus {
  if (bot.deleted) {
    return BotStatus.Deleted;
  }
  if (bot.isDead) {
    return BotStatus.Dead;
  }
  if (bot.quarantined) {
    return BotStatus.Dead;
  }
  if (bot.maintenanceMsg || bot.taskId) {
    return BotStatus.Busy;
  }
  return BotStatus.Idle;
}

export function getBotUrl(swarmingHost: string, botId: string): string {
  return `https://${swarmingHost}/bot?id=${botId}`;
}
