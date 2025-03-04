/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Duration } from "../../../../../google/protobuf/duration.pb";
import { Timestamp } from "../../../../../google/protobuf/timestamp.pb";
import { Changelist } from "./sources.pb";

export const protobufPackage = "luci.analysis.v1";

/**
 * Status of a test result.
 * It is a mirror of luci.resultdb.v1.TestStatus, but the right to evolve
 * it independently is reserved.
 */
export enum TestResultStatus {
  /**
   * UNSPECIFIED - Status was not specified.
   * Not to be used in actual test results; serves as a default value for an
   * unset field.
   */
  UNSPECIFIED = 0,
  /** PASS - The test case has passed. */
  PASS = 1,
  /**
   * FAIL - The test case has failed.
   * Suggests that the code under test is incorrect, but it is also possible
   * that the test is incorrect or it is a flake.
   */
  FAIL = 2,
  /**
   * CRASH - The test case has crashed during execution.
   * The outcome is inconclusive: the code under test might or might not be
   * correct, but the test+code is incorrect.
   */
  CRASH = 3,
  /**
   * ABORT - The test case has started, but was aborted before finishing.
   * A common reason: timeout.
   */
  ABORT = 4,
  /**
   * SKIP - The test case did not execute.
   * Examples:
   * - The execution of the collection of test cases, such as a test
   *   binary, was aborted prematurely and execution of some test cases was
   *   skipped.
   * - The test harness configuration specified that the test case MUST be
   *   skipped.
   */
  SKIP = 5,
}

export function testResultStatusFromJSON(object: any): TestResultStatus {
  switch (object) {
    case 0:
    case "TEST_RESULT_STATUS_UNSPECIFIED":
      return TestResultStatus.UNSPECIFIED;
    case 1:
    case "PASS":
      return TestResultStatus.PASS;
    case 2:
    case "FAIL":
      return TestResultStatus.FAIL;
    case 3:
    case "CRASH":
      return TestResultStatus.CRASH;
    case 4:
    case "ABORT":
      return TestResultStatus.ABORT;
    case 5:
    case "SKIP":
      return TestResultStatus.SKIP;
    default:
      throw new globalThis.Error("Unrecognized enum value " + object + " for enum TestResultStatus");
  }
}

export function testResultStatusToJSON(object: TestResultStatus): string {
  switch (object) {
    case TestResultStatus.UNSPECIFIED:
      return "TEST_RESULT_STATUS_UNSPECIFIED";
    case TestResultStatus.PASS:
      return "PASS";
    case TestResultStatus.FAIL:
      return "FAIL";
    case TestResultStatus.CRASH:
      return "CRASH";
    case TestResultStatus.ABORT:
      return "ABORT";
    case TestResultStatus.SKIP:
      return "SKIP";
    default:
      throw new globalThis.Error("Unrecognized enum value " + object + " for enum TestResultStatus");
  }
}

/**
 * Status of a test verdict.
 * It is a mirror of luci.resultdb.v1.TestVariantStatus.
 */
export enum TestVerdictStatus {
  /**
   * UNSPECIFIED - a test verdict must not have this status.
   * This is only used when filtering verdicts.
   */
  UNSPECIFIED = 0,
  /** UNEXPECTED - The test verdict has no exonerations, and all results are unexpected. */
  UNEXPECTED = 10,
  /** UNEXPECTEDLY_SKIPPED - The test verdict has no exonerations, and all results are unexpectedly skipped. */
  UNEXPECTEDLY_SKIPPED = 20,
  /**
   * FLAKY - The test verdict has no exonerations, and has both expected and unexpected
   * results.
   */
  FLAKY = 30,
  /** EXONERATED - The test verdict has one or more test exonerations. */
  EXONERATED = 40,
  /** EXPECTED - The test verdict has no exonerations, and all results are expected. */
  EXPECTED = 50,
}

export function testVerdictStatusFromJSON(object: any): TestVerdictStatus {
  switch (object) {
    case 0:
    case "TEST_VERDICT_STATUS_UNSPECIFIED":
      return TestVerdictStatus.UNSPECIFIED;
    case 10:
    case "UNEXPECTED":
      return TestVerdictStatus.UNEXPECTED;
    case 20:
    case "UNEXPECTEDLY_SKIPPED":
      return TestVerdictStatus.UNEXPECTEDLY_SKIPPED;
    case 30:
    case "FLAKY":
      return TestVerdictStatus.FLAKY;
    case 40:
    case "EXONERATED":
      return TestVerdictStatus.EXONERATED;
    case 50:
    case "EXPECTED":
      return TestVerdictStatus.EXPECTED;
    default:
      throw new globalThis.Error("Unrecognized enum value " + object + " for enum TestVerdictStatus");
  }
}

export function testVerdictStatusToJSON(object: TestVerdictStatus): string {
  switch (object) {
    case TestVerdictStatus.UNSPECIFIED:
      return "TEST_VERDICT_STATUS_UNSPECIFIED";
    case TestVerdictStatus.UNEXPECTED:
      return "UNEXPECTED";
    case TestVerdictStatus.UNEXPECTEDLY_SKIPPED:
      return "UNEXPECTEDLY_SKIPPED";
    case TestVerdictStatus.FLAKY:
      return "FLAKY";
    case TestVerdictStatus.EXONERATED:
      return "EXONERATED";
    case TestVerdictStatus.EXPECTED:
      return "EXPECTED";
    default:
      throw new globalThis.Error("Unrecognized enum value " + object + " for enum TestVerdictStatus");
  }
}

export interface TestVerdict {
  /**
   * Unique identifier of the test.
   * This has the same value as luci.resultdb.v1.TestResult.test_id.
   */
  readonly testId: string;
  /** The hash of the variant. */
  readonly variantHash: string;
  /**
   * The ID of the top-level invocation that the test verdict belongs to when
   * ingested.
   */
  readonly invocationId: string;
  /** The status of the test verdict. */
  readonly status: TestVerdictStatus;
  /**
   * Start time of the presubmit run (for results that are part of a presubmit
   * run) or start time of the buildbucket build (otherwise).
   */
  readonly partitionTime:
    | string
    | undefined;
  /**
   * The average duration of the PASSED test results included in the test
   * verdict.
   */
  readonly passedAvgDuration:
    | Duration
    | undefined;
  /**
   * The changelist(s) that were tested, if any. If there are more 10, only
   * the first 10 are returned here.
   */
  readonly changelists: readonly Changelist[];
}

function createBaseTestVerdict(): TestVerdict {
  return {
    testId: "",
    variantHash: "",
    invocationId: "",
    status: 0,
    partitionTime: undefined,
    passedAvgDuration: undefined,
    changelists: [],
  };
}

export const TestVerdict = {
  encode(message: TestVerdict, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.testId !== "") {
      writer.uint32(10).string(message.testId);
    }
    if (message.variantHash !== "") {
      writer.uint32(18).string(message.variantHash);
    }
    if (message.invocationId !== "") {
      writer.uint32(26).string(message.invocationId);
    }
    if (message.status !== 0) {
      writer.uint32(32).int32(message.status);
    }
    if (message.partitionTime !== undefined) {
      Timestamp.encode(toTimestamp(message.partitionTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.passedAvgDuration !== undefined) {
      Duration.encode(message.passedAvgDuration, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.changelists) {
      Changelist.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TestVerdict {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTestVerdict() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.variantHash = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.invocationId = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.partitionTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.passedAvgDuration = Duration.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.changelists.push(Changelist.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TestVerdict {
    return {
      testId: isSet(object.testId) ? globalThis.String(object.testId) : "",
      variantHash: isSet(object.variantHash) ? globalThis.String(object.variantHash) : "",
      invocationId: isSet(object.invocationId) ? globalThis.String(object.invocationId) : "",
      status: isSet(object.status) ? testVerdictStatusFromJSON(object.status) : 0,
      partitionTime: isSet(object.partitionTime) ? globalThis.String(object.partitionTime) : undefined,
      passedAvgDuration: isSet(object.passedAvgDuration) ? Duration.fromJSON(object.passedAvgDuration) : undefined,
      changelists: globalThis.Array.isArray(object?.changelists)
        ? object.changelists.map((e: any) => Changelist.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TestVerdict): unknown {
    const obj: any = {};
    if (message.testId !== "") {
      obj.testId = message.testId;
    }
    if (message.variantHash !== "") {
      obj.variantHash = message.variantHash;
    }
    if (message.invocationId !== "") {
      obj.invocationId = message.invocationId;
    }
    if (message.status !== 0) {
      obj.status = testVerdictStatusToJSON(message.status);
    }
    if (message.partitionTime !== undefined) {
      obj.partitionTime = message.partitionTime;
    }
    if (message.passedAvgDuration !== undefined) {
      obj.passedAvgDuration = Duration.toJSON(message.passedAvgDuration);
    }
    if (message.changelists?.length) {
      obj.changelists = message.changelists.map((e) => Changelist.toJSON(e));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<TestVerdict>, I>>(base?: I): TestVerdict {
    return TestVerdict.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<TestVerdict>, I>>(object: I): TestVerdict {
    const message = createBaseTestVerdict() as any;
    message.testId = object.testId ?? "";
    message.variantHash = object.variantHash ?? "";
    message.invocationId = object.invocationId ?? "";
    message.status = object.status ?? 0;
    message.partitionTime = object.partitionTime ?? undefined;
    message.passedAvgDuration = (object.passedAvgDuration !== undefined && object.passedAvgDuration !== null)
      ? Duration.fromPartial(object.passedAvgDuration)
      : undefined;
    message.changelists = object.changelists?.map((e) => Changelist.fromPartial(e)) || [];
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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
