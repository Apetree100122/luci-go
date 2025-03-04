/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../../../../../google/protobuf/timestamp.pb";

export const protobufPackage = "luci.resultdb.v1";

/**
 * A key-value map describing one variant of a test case.
 *
 * The same test case can be executed in different ways, for example on
 * different OS, GPUs, with different compile options or runtime flags.
 * A variant definition captures one variant.
 * A test case with a specific variant definition is called test variant.
 *
 * Guidelines for variant definition design:
 * - This rule guides what keys MUST be present in the definition.
 *   A single expected result of a given test variant is enough to consider it
 *   passing (potentially flakily). If it is important to differentiate across
 *   a certain dimension (e.g. whether web tests are executed with or without
 *   site per process isolation), then there MUST be a key that captures the
 *   dimension (e.g. a name from test_suites.pyl).
 *   Otherwise, a pass in one variant will hide a failure of another one.
 *
 * - This rule guides what keys MUST NOT be present in the definition.
 *   A change in the key-value set essentially resets the test result history.
 *   For example, if GN args are among variant key-value pairs, then adding a
 *   new GN arg changes the identity of the test variant and resets its history.
 *
 * In Chromium, variant keys are:
 * - bucket: the LUCI bucket, e.g. "ci"
 * - builder: the LUCI builder, e.g. "linux-rel"
 * - test_suite: a name from
 *   https://cs.chromium.org/chromium/src/testing/buildbot/test_suites.pyl
 */
export interface Variant {
  /**
   * The definition of the variant.
   * Key and values must be valid StringPair keys and values, see their
   * constraints.
   */
  readonly def: { [key: string]: string };
}

export interface Variant_DefEntry {
  readonly key: string;
  readonly value: string;
}

/** A string key-value pair. Typically used for tagging, see Invocation.tags */
export interface StringPair {
  /**
   * Regex: ^[a-z][a-z0-9_]*(/[a-z][a-z0-9_]*)*$
   * Max length: 64.
   */
  readonly key: string;
  /** Max length: 256. */
  readonly value: string;
}

/**
 * GitilesCommit specifies the position of the gitiles commit an invocation
 * ran against, in a repository's commit log. More specifically, a ref's commit
 * log.
 *
 * It also specifies the host/project/ref combination that the commit
 * exists in, to provide context.
 */
export interface GitilesCommit {
  /**
   * The identity of the gitiles host, e.g. "chromium.googlesource.com".
   * Mandatory.
   */
  readonly host: string;
  /** Repository name on the host, e.g. "chromium/src". Mandatory. */
  readonly project: string;
  /**
   * Commit ref, e.g. "refs/heads/main" from which the commit was fetched.
   * Not the branch name, use "refs/heads/branch"
   * Mandatory.
   */
  readonly ref: string;
  /** Commit HEX SHA1. All lowercase. Mandatory. */
  readonly commitHash: string;
  /**
   * Defines a total order of commits on the ref.
   * A positive, monotonically increasing integer. The recommended
   * way of obtaining this is by using the goto.google.com/git-numberer
   * Gerrit plugin. Other solutions can be used as well, so long
   * as the same scheme is used consistently for a ref.
   * Mandatory.
   */
  readonly position: string;
}

/** A Gerrit patchset. */
export interface GerritChange {
  /** Gerrit hostname, e.g. "chromium-review.googlesource.com". */
  readonly host: string;
  /** Gerrit project, e.g. "chromium/src". */
  readonly project: string;
  /** Change number, e.g. 12345. */
  readonly change: string;
  /** Patch set number, e.g. 1. */
  readonly patchset: string;
}

/** Deprecated: Use GitilesCommit instead. */
export interface CommitPosition {
  /**
   * The following fields identify a git repository and a ref within which the
   * numerical position below identifies a single commit.
   */
  readonly host: string;
  readonly project: string;
  readonly ref: string;
  /**
   * The numerical position of the commit in the log for the host/project/ref
   * above.
   */
  readonly position: string;
}

/** Deprecated: Do not use. */
export interface CommitPositionRange {
  /** The lowest commit position to include in the range. */
  readonly earliest:
    | CommitPosition
    | undefined;
  /** Include only commit positions that that are strictly lower than this. */
  readonly latest: CommitPosition | undefined;
}

/**
 * A range of timestamps.
 *
 * Currently unused.
 */
export interface TimeRange {
  /** The oldest timestamp to include in the range. */
  readonly earliest:
    | string
    | undefined;
  /** Include only timestamps that are strictly older than this. */
  readonly latest: string | undefined;
}

/** Represents a reference in a source control system. */
export interface SourceRef {
  /** A branch in gitiles repository. */
  readonly gitiles?: GitilesRef | undefined;
}

/** Represents a branch in a gitiles repository. */
export interface GitilesRef {
  /** The gitiles host, e.g. "chromium.googlesource.com". */
  readonly host: string;
  /** The project on the gitiles host, e.g. "chromium/src". */
  readonly project: string;
  /**
   * Commit ref, e.g. "refs/heads/main" from which the commit was fetched.
   * Not the branch name, use "refs/heads/branch"
   */
  readonly ref: string;
}

function createBaseVariant(): Variant {
  return { def: {} };
}

export const Variant = {
  encode(message: Variant, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.def).forEach(([key, value]) => {
      Variant_DefEntry.encode({ key: key as any, value }, writer.uint32(10).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Variant {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVariant() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          const entry1 = Variant_DefEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.def[entry1.key] = entry1.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Variant {
    return {
      def: isObject(object.def)
        ? Object.entries(object.def).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Variant): unknown {
    const obj: any = {};
    if (message.def) {
      const entries = Object.entries(message.def);
      if (entries.length > 0) {
        obj.def = {};
        entries.forEach(([k, v]) => {
          obj.def[k] = v;
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Variant>, I>>(base?: I): Variant {
    return Variant.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Variant>, I>>(object: I): Variant {
    const message = createBaseVariant() as any;
    message.def = Object.entries(object.def ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseVariant_DefEntry(): Variant_DefEntry {
  return { key: "", value: "" };
}

export const Variant_DefEntry = {
  encode(message: Variant_DefEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Variant_DefEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVariant_DefEntry() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Variant_DefEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: Variant_DefEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Variant_DefEntry>, I>>(base?: I): Variant_DefEntry {
    return Variant_DefEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Variant_DefEntry>, I>>(object: I): Variant_DefEntry {
    const message = createBaseVariant_DefEntry() as any;
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseStringPair(): StringPair {
  return { key: "", value: "" };
}

export const StringPair = {
  encode(message: StringPair, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StringPair {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStringPair() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StringPair {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: StringPair): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<StringPair>, I>>(base?: I): StringPair {
    return StringPair.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<StringPair>, I>>(object: I): StringPair {
    const message = createBaseStringPair() as any;
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseGitilesCommit(): GitilesCommit {
  return { host: "", project: "", ref: "", commitHash: "", position: "0" };
}

export const GitilesCommit = {
  encode(message: GitilesCommit, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    if (message.ref !== "") {
      writer.uint32(26).string(message.ref);
    }
    if (message.commitHash !== "") {
      writer.uint32(34).string(message.commitHash);
    }
    if (message.position !== "0") {
      writer.uint32(40).int64(message.position);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GitilesCommit {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGitilesCommit() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ref = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.commitHash = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.position = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GitilesCommit {
    return {
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      ref: isSet(object.ref) ? globalThis.String(object.ref) : "",
      commitHash: isSet(object.commitHash) ? globalThis.String(object.commitHash) : "",
      position: isSet(object.position) ? globalThis.String(object.position) : "0",
    };
  },

  toJSON(message: GitilesCommit): unknown {
    const obj: any = {};
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.ref !== "") {
      obj.ref = message.ref;
    }
    if (message.commitHash !== "") {
      obj.commitHash = message.commitHash;
    }
    if (message.position !== "0") {
      obj.position = message.position;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GitilesCommit>, I>>(base?: I): GitilesCommit {
    return GitilesCommit.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GitilesCommit>, I>>(object: I): GitilesCommit {
    const message = createBaseGitilesCommit() as any;
    message.host = object.host ?? "";
    message.project = object.project ?? "";
    message.ref = object.ref ?? "";
    message.commitHash = object.commitHash ?? "";
    message.position = object.position ?? "0";
    return message;
  },
};

function createBaseGerritChange(): GerritChange {
  return { host: "", project: "", change: "0", patchset: "0" };
}

export const GerritChange = {
  encode(message: GerritChange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    if (message.change !== "0") {
      writer.uint32(24).int64(message.change);
    }
    if (message.patchset !== "0") {
      writer.uint32(32).int64(message.patchset);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GerritChange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGerritChange() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.change = longToString(reader.int64() as Long);
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.patchset = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GerritChange {
    return {
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      change: isSet(object.change) ? globalThis.String(object.change) : "0",
      patchset: isSet(object.patchset) ? globalThis.String(object.patchset) : "0",
    };
  },

  toJSON(message: GerritChange): unknown {
    const obj: any = {};
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.change !== "0") {
      obj.change = message.change;
    }
    if (message.patchset !== "0") {
      obj.patchset = message.patchset;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GerritChange>, I>>(base?: I): GerritChange {
    return GerritChange.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GerritChange>, I>>(object: I): GerritChange {
    const message = createBaseGerritChange() as any;
    message.host = object.host ?? "";
    message.project = object.project ?? "";
    message.change = object.change ?? "0";
    message.patchset = object.patchset ?? "0";
    return message;
  },
};

function createBaseCommitPosition(): CommitPosition {
  return { host: "", project: "", ref: "", position: "0" };
}

export const CommitPosition = {
  encode(message: CommitPosition, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    if (message.ref !== "") {
      writer.uint32(26).string(message.ref);
    }
    if (message.position !== "0") {
      writer.uint32(32).int64(message.position);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CommitPosition {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCommitPosition() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ref = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.position = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CommitPosition {
    return {
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      ref: isSet(object.ref) ? globalThis.String(object.ref) : "",
      position: isSet(object.position) ? globalThis.String(object.position) : "0",
    };
  },

  toJSON(message: CommitPosition): unknown {
    const obj: any = {};
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.ref !== "") {
      obj.ref = message.ref;
    }
    if (message.position !== "0") {
      obj.position = message.position;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<CommitPosition>, I>>(base?: I): CommitPosition {
    return CommitPosition.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<CommitPosition>, I>>(object: I): CommitPosition {
    const message = createBaseCommitPosition() as any;
    message.host = object.host ?? "";
    message.project = object.project ?? "";
    message.ref = object.ref ?? "";
    message.position = object.position ?? "0";
    return message;
  },
};

function createBaseCommitPositionRange(): CommitPositionRange {
  return { earliest: undefined, latest: undefined };
}

export const CommitPositionRange = {
  encode(message: CommitPositionRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.earliest !== undefined) {
      CommitPosition.encode(message.earliest, writer.uint32(10).fork()).ldelim();
    }
    if (message.latest !== undefined) {
      CommitPosition.encode(message.latest, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CommitPositionRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCommitPositionRange() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.earliest = CommitPosition.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.latest = CommitPosition.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CommitPositionRange {
    return {
      earliest: isSet(object.earliest) ? CommitPosition.fromJSON(object.earliest) : undefined,
      latest: isSet(object.latest) ? CommitPosition.fromJSON(object.latest) : undefined,
    };
  },

  toJSON(message: CommitPositionRange): unknown {
    const obj: any = {};
    if (message.earliest !== undefined) {
      obj.earliest = CommitPosition.toJSON(message.earliest);
    }
    if (message.latest !== undefined) {
      obj.latest = CommitPosition.toJSON(message.latest);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<CommitPositionRange>, I>>(base?: I): CommitPositionRange {
    return CommitPositionRange.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<CommitPositionRange>, I>>(object: I): CommitPositionRange {
    const message = createBaseCommitPositionRange() as any;
    message.earliest = (object.earliest !== undefined && object.earliest !== null)
      ? CommitPosition.fromPartial(object.earliest)
      : undefined;
    message.latest = (object.latest !== undefined && object.latest !== null)
      ? CommitPosition.fromPartial(object.latest)
      : undefined;
    return message;
  },
};

function createBaseTimeRange(): TimeRange {
  return { earliest: undefined, latest: undefined };
}

export const TimeRange = {
  encode(message: TimeRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.earliest !== undefined) {
      Timestamp.encode(toTimestamp(message.earliest), writer.uint32(10).fork()).ldelim();
    }
    if (message.latest !== undefined) {
      Timestamp.encode(toTimestamp(message.latest), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TimeRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTimeRange() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.earliest = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.latest = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TimeRange {
    return {
      earliest: isSet(object.earliest) ? globalThis.String(object.earliest) : undefined,
      latest: isSet(object.latest) ? globalThis.String(object.latest) : undefined,
    };
  },

  toJSON(message: TimeRange): unknown {
    const obj: any = {};
    if (message.earliest !== undefined) {
      obj.earliest = message.earliest;
    }
    if (message.latest !== undefined) {
      obj.latest = message.latest;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<TimeRange>, I>>(base?: I): TimeRange {
    return TimeRange.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<TimeRange>, I>>(object: I): TimeRange {
    const message = createBaseTimeRange() as any;
    message.earliest = object.earliest ?? undefined;
    message.latest = object.latest ?? undefined;
    return message;
  },
};

function createBaseSourceRef(): SourceRef {
  return { gitiles: undefined };
}

export const SourceRef = {
  encode(message: SourceRef, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.gitiles !== undefined) {
      GitilesRef.encode(message.gitiles, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceRef {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceRef() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.gitiles = GitilesRef.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SourceRef {
    return { gitiles: isSet(object.gitiles) ? GitilesRef.fromJSON(object.gitiles) : undefined };
  },

  toJSON(message: SourceRef): unknown {
    const obj: any = {};
    if (message.gitiles !== undefined) {
      obj.gitiles = GitilesRef.toJSON(message.gitiles);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SourceRef>, I>>(base?: I): SourceRef {
    return SourceRef.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<SourceRef>, I>>(object: I): SourceRef {
    const message = createBaseSourceRef() as any;
    message.gitiles = (object.gitiles !== undefined && object.gitiles !== null)
      ? GitilesRef.fromPartial(object.gitiles)
      : undefined;
    return message;
  },
};

function createBaseGitilesRef(): GitilesRef {
  return { host: "", project: "", ref: "" };
}

export const GitilesRef = {
  encode(message: GitilesRef, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.project !== "") {
      writer.uint32(18).string(message.project);
    }
    if (message.ref !== "") {
      writer.uint32(26).string(message.ref);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GitilesRef {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGitilesRef() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.project = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ref = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GitilesRef {
    return {
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      ref: isSet(object.ref) ? globalThis.String(object.ref) : "",
    };
  },

  toJSON(message: GitilesRef): unknown {
    const obj: any = {};
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.ref !== "") {
      obj.ref = message.ref;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GitilesRef>, I>>(base?: I): GitilesRef {
    return GitilesRef.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GitilesRef>, I>>(object: I): GitilesRef {
    const message = createBaseGitilesRef() as any;
    message.host = object.host ?? "";
    message.project = object.project ?? "";
    message.ref = object.ref ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function toTimestamp(dateStr: string): Timestamp {
  const date = new globalThis.Date(dateStr);
  const seconds = Math.trunc(date.getTime() / 1_000).toString();
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): string {
  let millis = (globalThis.Number(t.seconds) || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis).toISOString();
}

function longToString(long: Long) {
  return long.toString();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
