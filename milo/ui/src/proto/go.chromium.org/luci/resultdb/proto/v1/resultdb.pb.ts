/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { FieldMask } from "../../../../../google/protobuf/field_mask.pb";
import { Artifact } from "./artifact.pb";
import { Invocation, Sources } from "./invocation.pb";
import {
  ArtifactPredicate,
  TestExonerationPredicate,
  TestMetadataPredicate,
  TestResultPredicate,
} from "./predicate.pb";
import { TestMetadataDetail } from "./test_metadata.pb";
import { TestExoneration, TestResult } from "./test_result.pb";
import { TestVariant, TestVariantPredicate } from "./test_variant.pb";

export const protobufPackage = "luci.resultdb.v1";

/** A request message for GetInvocation RPC. */
export interface GetInvocationRequest {
  /** The name of the invocation to request, see Invocation.name. */
  readonly name: string;
}

/** A request message for GetTestResult RPC. */
export interface GetTestResultRequest {
  /** The name of the test result to request, see TestResult.name. */
  readonly name: string;
}

/** A request message for ListTestResults RPC. */
export interface ListTestResultsRequest {
  /** Name of the invocation, e.g. "invocations/{id}". */
  readonly invocation: string;
  /**
   * The maximum number of test results to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 test results will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `ListTestResults` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListTestResults` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
  /**
   * Fields to include in the response.
   * If not set, the default mask is used where summary_html and tags are
   * excluded.
   * Test result names will always be included even if "name" is not a part of
   * the mask.
   */
  readonly readMask: readonly string[] | undefined;
}

/** A response message for ListTestResults RPC. */
export interface ListTestResultsResponse {
  /** The test results from the specified invocation. */
  readonly testResults: readonly TestResult[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   * If the invocation is not finalized, more results may appear later.
   */
  readonly nextPageToken: string;
}

/** A request message for GetTestExoneration RPC. */
export interface GetTestExonerationRequest {
  /** The name of the test exoneration to request, see TestExoneration.name. */
  readonly name: string;
}

/** A request message for ListTestExonerations RPC. */
export interface ListTestExonerationsRequest {
  /** Name of the invocation, e.g. "invocations/{id}". */
  readonly invocation: string;
  /**
   * The maximum number of test exonerations to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 test exonerations will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `ListTestExonerations` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListTestExonerations`
   * MUST match the call that provided the page token.
   */
  readonly pageToken: string;
}

/** A response message for ListTestExonerations RPC. */
export interface ListTestExonerationsResponse {
  /** The test exonerations from the specified invocation. */
  readonly testExonerations: readonly TestExoneration[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   * If the invocation is not finalized, more results may appear later.
   */
  readonly nextPageToken: string;
}

/** A request message for QueryTestResults RPC. */
export interface QueryTestResultsRequest {
  /**
   * Retrieve test results included in these invocations, directly or indirectly
   * (via Invocation.included_invocations).
   *
   * Specifying multiple invocations is equivalent to querying one invocation
   * that includes these.
   */
  readonly invocations: readonly string[];
  /** A test result in the response must satisfy this predicate. */
  readonly predicate:
    | TestResultPredicate
    | undefined;
  /**
   * The maximum number of test results to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 test results will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `QueryTestResults` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `QueryTestResults` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
  /**
   * Fields to include in the response.
   * If not set, the default mask is used where summary_html and tags are
   * excluded.
   * Test result names will always be included even if "name" is not a part of
   * the mask.
   */
  readonly readMask: readonly string[] | undefined;
}

/** A response message for QueryTestResults RPC. */
export interface QueryTestResultsResponse {
  /**
   * Matched test results.
   * Ordered by parent invocation ID, test ID and result ID.
   */
  readonly testResults: readonly TestResult[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   */
  readonly nextPageToken: string;
}

/** A request message for QueryTestExonerations RPC. */
export interface QueryTestExonerationsRequest {
  /**
   * Retrieve test exonerations included in these invocations, directly or
   * indirectly (via Invocation.included_invocations).
   *
   * Specifying multiple invocations is equivalent to querying one invocation
   * that includes these.
   */
  readonly invocations: readonly string[];
  /** A test exoneration in the response must satisfy this predicate. */
  readonly predicate:
    | TestExonerationPredicate
    | undefined;
  /**
   * The maximum number of test exonerations to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 test exonerations will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `QueryTestExonerations` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `QueryTestExonerations`
   * MUST match the call that provided the page token.
   */
  readonly pageToken: string;
}

/** A response message for QueryTestExonerations RPC. */
export interface QueryTestExonerationsResponse {
  /**
   * The test exonerations matching the predicate.
   * Ordered by parent invocation ID, test ID and exoneration ID.
   */
  readonly testExonerations: readonly TestExoneration[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   */
  readonly nextPageToken: string;
}

/** A request message for QueryTestResultStatistics RPC. */
export interface QueryTestResultStatisticsRequest {
  /**
   * Retrieve statistics of test result belong to these invocations,
   * directly or indirectly (via Invocation.included_invocations).
   *
   * Specifying multiple invocations is equivalent to requesting one invocation
   * that includes these.
   */
  readonly invocations: readonly string[];
}

/** A response message for QueryTestResultStatistics RPC. */
export interface QueryTestResultStatisticsResponse {
  /** Total number of test results. */
  readonly totalTestResults: string;
}

/** A request message for GetArtifact RPC. */
export interface GetArtifactRequest {
  /** The name of the artifact to request, see Artifact.name. */
  readonly name: string;
}

/** A request message for ListArtifacts RPC. */
export interface ListArtifactsRequest {
  /**
   * Name of the parent, e.g. an invocation (see Invocation.name) or
   * a test result (see TestResult.name).
   */
  readonly parent: string;
  /**
   * The maximum number of artifacts to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 artifacts will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `ListArtifacts` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListArtifacts` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
}

/** A response message for ListArtifacts RPC. */
export interface ListArtifactsResponse {
  /** The artifacts from the specified parent. */
  readonly artifacts: readonly Artifact[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   * If the invocation is not finalized, more results may appear later.
   */
  readonly nextPageToken: string;
}

/** A request message for QueryArtifacts RPC. */
export interface QueryArtifactsRequest {
  /**
   * Retrieve artifacts included in these invocations, directly or indirectly
   * (via Invocation.included_invocations and via contained test results).
   *
   * Specifying multiple invocations is equivalent to querying one invocation
   * that includes these.
   */
  readonly invocations: readonly string[];
  /** An artifact in the response must satisfy this predicate. */
  readonly predicate:
    | ArtifactPredicate
    | undefined;
  /**
   * The maximum number of artifacts to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 artifacts will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `QueryArtifacts` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `QueryArtifacts` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
}

/** A response message for QueryArtifacts RPC. */
export interface QueryArtifactsResponse {
  /**
   * Matched artifacts.
   * First invocation-level artifacts, then test-result-level artifacts
   * ordered by parent invocation ID, test ID and artifact ID.
   */
  readonly artifacts: readonly Artifact[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   */
  readonly nextPageToken: string;
}

/**
 * A request message for QueryTestVariants RPC.
 * Next id: 9.
 */
export interface QueryTestVariantsRequest {
  /**
   * Retrieve test variants included in these invocations, directly or indirectly
   * (via Invocation.included_invocations).
   *
   * Specifying multiple invocations is equivalent to querying one invocation
   * that includes these.
   */
  readonly invocations: readonly string[];
  /** A test variant must satisfy this predicate. */
  readonly predicate:
    | TestVariantPredicate
    | undefined;
  /**
   * The maximum number of test results to be included in a test variant.
   *
   * If a test variant has more results than the limit, the remaining results
   * will not be returned.
   * If unspecified, at most 10 results will be included in a test variant.
   * The maximum value is 100; values above 100 will be coerced to 100.
   */
  readonly resultLimit: number;
  /**
   * The maximum number of test variants to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 100 test variants will be returned.
   * The maximum value is 10,000; values above 10,000 will be coerced to 10,000.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `QueryTestVariants` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `QueryTestVariants` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
  /**
   * Fields to include in the response.
   * If not set, the default mask is used where all fields are included.
   *
   * The following fields in results.*.result will NEVER be included even when
   * specified:
   * * test_id
   * * variant_hash
   * * variant
   * * test_metadata
   * Those values can be found in the parent test variant objects.
   *
   * The following fields will ALWAYS be included even when NOT specified:
   * * test_id
   * * variant_hash
   * * status
   */
  readonly readMask: readonly string[] | undefined;
}

/** A response message for QueryTestVariants RPC. */
export interface QueryTestVariantsResponse {
  /**
   * Matched test variants.
   * Ordered by TestVariantStatus, test_id, then variant_hash
   */
  readonly testVariants: readonly TestVariant[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   */
  readonly nextPageToken: string;
  /**
   * The code sources tested by the returned test variants. The sources are keyed
   * by an ID which allows them to be cross-referenced from TestVariant.sources_id.
   *
   * The sources are returned via this map instead of directly on the TestVariant
   * to avoid excessive response size. Each source message could be up to a few
   * kilobytes and there are usually no more than a handful of different sources
   * tested in an invocation, so deduplicating them here reduces response size.
   */
  readonly sources: { [key: string]: Sources };
}

export interface QueryTestVariantsResponse_SourcesEntry {
  readonly key: string;
  readonly value: Sources | undefined;
}

/** A request message for BatchGetTestVariants RPC. */
export interface BatchGetTestVariantsRequest {
  /** Name of the invocation that the test variants are in. */
  readonly invocation: string;
  /**
   * A list of test IDs and variant hashes, identifying the requested test
   * variants. Size is limited to 500. Any request for more than 500 variants
   * will return an error.
   */
  readonly testVariants: readonly BatchGetTestVariantsRequest_TestVariantIdentifier[];
  /**
   * The maximum number of test results to be included in a test variant.
   *
   * If a test variant has more results than the limit, the remaining results
   * will not be returned.
   * If unspecified, at most 10 results will be included in a test variant.
   * The maximum value is 100; values above 100 will be coerced to 100.
   */
  readonly resultLimit: number;
}

export interface BatchGetTestVariantsRequest_TestVariantIdentifier {
  /**
   * The unique identifier of the test in a LUCI project. See the comment on
   * TestResult.test_id for full documentation.
   */
  readonly testId: string;
  /**
   * Hash of the variant. See the comment on TestResult.variant_hash for full
   * documentation.
   */
  readonly variantHash: string;
}

/** A response message for BatchGetTestVariants RPC. */
export interface BatchGetTestVariantsResponse {
  /**
   * Test variants matching the requests. Any variants that weren't found are
   * omitted from the response. Clients shouldn't rely on the ordering of this
   * field, as no particular order is guaranteed.
   */
  readonly testVariants: readonly TestVariant[];
  /**
   * The code sources tested by the returned test variants. The sources are keyed
   * by an ID which allows them to be cross-referenced from TestVariant.sources_id.
   *
   * The sources are returned via this map instead of directly on the TestVariant
   * to avoid excessive response size. Each source message could be up to a few
   * kilobytes and there are usually no more than a handful of different sources
   * tested in an invocation, so deduplicating them here reduces response size.
   */
  readonly sources: { [key: string]: Sources };
}

export interface BatchGetTestVariantsResponse_SourcesEntry {
  readonly key: string;
  readonly value: Sources | undefined;
}

/** A request message for QueryTestMetadata RPC. */
export interface QueryTestMetadataRequest {
  /** The LUCI Project to query. */
  readonly project: string;
  /** Filters to apply to the returned test metadata. */
  readonly predicate:
    | TestMetadataPredicate
    | undefined;
  /**
   * The maximum number of test metadata entries to return.
   *
   * The service may return fewer than this value.
   * If unspecified, at most 1000 test metadata entries will be returned.
   * The maximum value is 100K; values above 100K will be coerced to 100K.
   */
  readonly pageSize: number;
  /**
   * A page token, received from a previous `QueryTestMetadata` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `QueryTestMetadata` MUST
   * match the call that provided the page token.
   */
  readonly pageToken: string;
}

/** A response message for QueryTestMetadata RPC. */
export interface QueryTestMetadataResponse {
  /** The matched testMetadata. */
  readonly testMetadata: readonly TestMetadataDetail[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there were no subsequent pages at the time of
   * request.
   */
  readonly nextPageToken: string;
}

/**
 * A request message for QueryNewTestVariants RPC.
 * To use this RPC, callers need:
 * - resultdb.baselines.get in the realm the <baseline_project>:@project, where
 *   baseline_project is the LUCI project that contains the baseline.
 * - resultdb.testResults.list in the realm of the invocation which is being
 *   queried.
 */
export interface QueryNewTestVariantsRequest {
  /** Name of the invocation, e.g. "invocations/{id}". */
  readonly invocation: string;
  /**
   * The baseline to compare test variants against, to determine if they are new.
   * e.g. “projects/{project}/baselines/{baseline_id}”.
   * For example, in the project "chromium", the baseline_id may be
   * "try:linux-rel".
   */
  readonly baseline: string;
}

/** A response message for QueryNewTestVariants RPC. */
export interface QueryNewTestVariantsResponse {
  /**
   * Indicates whether the baseline has been populated with at least 72 hours
   * of data and the results can be relied upon.
   */
  readonly isBaselineReady: boolean;
  /**
   * Test variants that are new, meaning that they have not been part of
   * a submitted run prior.
   */
  readonly newTestVariants: readonly QueryNewTestVariantsResponse_NewTestVariant[];
}

/** Represents a new test, which contains minimal information to uniquely identify a TestVariant. */
export interface QueryNewTestVariantsResponse_NewTestVariant {
  /**
   * A unique identifier of the test in a LUCI project.
   * Regex: ^[[::print::]]{1,256}$
   *
   * Refer to TestResult.test_id for details.
   */
  readonly testId: string;
  /**
   * Hash of the variant.
   * hex(sha256(sorted(''.join('%s:%s\n' for k, v in variant.items())))).
   */
  readonly variantHash: string;
}

function createBaseGetInvocationRequest(): GetInvocationRequest {
  return { name: "" };
}

export const GetInvocationRequest = {
  encode(message: GetInvocationRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetInvocationRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetInvocationRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetInvocationRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetInvocationRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetInvocationRequest>, I>>(base?: I): GetInvocationRequest {
    return GetInvocationRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetInvocationRequest>, I>>(object: I): GetInvocationRequest {
    const message = createBaseGetInvocationRequest() as any;
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetTestResultRequest(): GetTestResultRequest {
  return { name: "" };
}

export const GetTestResultRequest = {
  encode(message: GetTestResultRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTestResultRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTestResultRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetTestResultRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetTestResultRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetTestResultRequest>, I>>(base?: I): GetTestResultRequest {
    return GetTestResultRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetTestResultRequest>, I>>(object: I): GetTestResultRequest {
    const message = createBaseGetTestResultRequest() as any;
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListTestResultsRequest(): ListTestResultsRequest {
  return { invocation: "", pageSize: 0, pageToken: "", readMask: undefined };
}

export const ListTestResultsRequest = {
  encode(message: ListTestResultsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.invocation !== "") {
      writer.uint32(10).string(message.invocation);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    if (message.readMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.readMask), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTestResultsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTestResultsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocation = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.readMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTestResultsRequest {
    return {
      invocation: isSet(object.invocation) ? globalThis.String(object.invocation) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      readMask: isSet(object.readMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.readMask)) : undefined,
    };
  },

  toJSON(message: ListTestResultsRequest): unknown {
    const obj: any = {};
    if (message.invocation !== "") {
      obj.invocation = message.invocation;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.readMask !== undefined) {
      obj.readMask = FieldMask.toJSON(FieldMask.wrap(message.readMask));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListTestResultsRequest>, I>>(base?: I): ListTestResultsRequest {
    return ListTestResultsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListTestResultsRequest>, I>>(object: I): ListTestResultsRequest {
    const message = createBaseListTestResultsRequest() as any;
    message.invocation = object.invocation ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.readMask = object.readMask ?? undefined;
    return message;
  },
};

function createBaseListTestResultsResponse(): ListTestResultsResponse {
  return { testResults: [], nextPageToken: "" };
}

export const ListTestResultsResponse = {
  encode(message: ListTestResultsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testResults) {
      TestResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTestResultsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTestResultsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testResults.push(TestResult.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTestResultsResponse {
    return {
      testResults: globalThis.Array.isArray(object?.testResults)
        ? object.testResults.map((e: any) => TestResult.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListTestResultsResponse): unknown {
    const obj: any = {};
    if (message.testResults?.length) {
      obj.testResults = message.testResults.map((e) => TestResult.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListTestResultsResponse>, I>>(base?: I): ListTestResultsResponse {
    return ListTestResultsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListTestResultsResponse>, I>>(object: I): ListTestResultsResponse {
    const message = createBaseListTestResultsResponse() as any;
    message.testResults = object.testResults?.map((e) => TestResult.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetTestExonerationRequest(): GetTestExonerationRequest {
  return { name: "" };
}

export const GetTestExonerationRequest = {
  encode(message: GetTestExonerationRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTestExonerationRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTestExonerationRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetTestExonerationRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetTestExonerationRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetTestExonerationRequest>, I>>(base?: I): GetTestExonerationRequest {
    return GetTestExonerationRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetTestExonerationRequest>, I>>(object: I): GetTestExonerationRequest {
    const message = createBaseGetTestExonerationRequest() as any;
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListTestExonerationsRequest(): ListTestExonerationsRequest {
  return { invocation: "", pageSize: 0, pageToken: "" };
}

export const ListTestExonerationsRequest = {
  encode(message: ListTestExonerationsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.invocation !== "") {
      writer.uint32(10).string(message.invocation);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTestExonerationsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTestExonerationsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocation = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTestExonerationsRequest {
    return {
      invocation: isSet(object.invocation) ? globalThis.String(object.invocation) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListTestExonerationsRequest): unknown {
    const obj: any = {};
    if (message.invocation !== "") {
      obj.invocation = message.invocation;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListTestExonerationsRequest>, I>>(base?: I): ListTestExonerationsRequest {
    return ListTestExonerationsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListTestExonerationsRequest>, I>>(object: I): ListTestExonerationsRequest {
    const message = createBaseListTestExonerationsRequest() as any;
    message.invocation = object.invocation ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListTestExonerationsResponse(): ListTestExonerationsResponse {
  return { testExonerations: [], nextPageToken: "" };
}

export const ListTestExonerationsResponse = {
  encode(message: ListTestExonerationsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testExonerations) {
      TestExoneration.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTestExonerationsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTestExonerationsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testExonerations.push(TestExoneration.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTestExonerationsResponse {
    return {
      testExonerations: globalThis.Array.isArray(object?.testExonerations)
        ? object.testExonerations.map((e: any) => TestExoneration.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListTestExonerationsResponse): unknown {
    const obj: any = {};
    if (message.testExonerations?.length) {
      obj.testExonerations = message.testExonerations.map((e) => TestExoneration.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListTestExonerationsResponse>, I>>(base?: I): ListTestExonerationsResponse {
    return ListTestExonerationsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListTestExonerationsResponse>, I>>(object: I): ListTestExonerationsResponse {
    const message = createBaseListTestExonerationsResponse() as any;
    message.testExonerations = object.testExonerations?.map((e) => TestExoneration.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryTestResultsRequest(): QueryTestResultsRequest {
  return { invocations: [], predicate: undefined, pageSize: 0, pageToken: "", readMask: undefined };
}

export const QueryTestResultsRequest = {
  encode(message: QueryTestResultsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.invocations) {
      writer.uint32(10).string(v!);
    }
    if (message.predicate !== undefined) {
      TestResultPredicate.encode(message.predicate, writer.uint32(18).fork()).ldelim();
    }
    if (message.pageSize !== 0) {
      writer.uint32(32).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(42).string(message.pageToken);
    }
    if (message.readMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.readMask), writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestResultsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestResultsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocations.push(reader.string());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.predicate = TestResultPredicate.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.readMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestResultsRequest {
    return {
      invocations: globalThis.Array.isArray(object?.invocations)
        ? object.invocations.map((e: any) => globalThis.String(e))
        : [],
      predicate: isSet(object.predicate) ? TestResultPredicate.fromJSON(object.predicate) : undefined,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      readMask: isSet(object.readMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.readMask)) : undefined,
    };
  },

  toJSON(message: QueryTestResultsRequest): unknown {
    const obj: any = {};
    if (message.invocations?.length) {
      obj.invocations = message.invocations;
    }
    if (message.predicate !== undefined) {
      obj.predicate = TestResultPredicate.toJSON(message.predicate);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.readMask !== undefined) {
      obj.readMask = FieldMask.toJSON(FieldMask.wrap(message.readMask));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestResultsRequest>, I>>(base?: I): QueryTestResultsRequest {
    return QueryTestResultsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestResultsRequest>, I>>(object: I): QueryTestResultsRequest {
    const message = createBaseQueryTestResultsRequest() as any;
    message.invocations = object.invocations?.map((e) => e) || [];
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? TestResultPredicate.fromPartial(object.predicate)
      : undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.readMask = object.readMask ?? undefined;
    return message;
  },
};

function createBaseQueryTestResultsResponse(): QueryTestResultsResponse {
  return { testResults: [], nextPageToken: "" };
}

export const QueryTestResultsResponse = {
  encode(message: QueryTestResultsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testResults) {
      TestResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestResultsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestResultsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testResults.push(TestResult.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestResultsResponse {
    return {
      testResults: globalThis.Array.isArray(object?.testResults)
        ? object.testResults.map((e: any) => TestResult.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: QueryTestResultsResponse): unknown {
    const obj: any = {};
    if (message.testResults?.length) {
      obj.testResults = message.testResults.map((e) => TestResult.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestResultsResponse>, I>>(base?: I): QueryTestResultsResponse {
    return QueryTestResultsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestResultsResponse>, I>>(object: I): QueryTestResultsResponse {
    const message = createBaseQueryTestResultsResponse() as any;
    message.testResults = object.testResults?.map((e) => TestResult.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryTestExonerationsRequest(): QueryTestExonerationsRequest {
  return { invocations: [], predicate: undefined, pageSize: 0, pageToken: "" };
}

export const QueryTestExonerationsRequest = {
  encode(message: QueryTestExonerationsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.invocations) {
      writer.uint32(10).string(v!);
    }
    if (message.predicate !== undefined) {
      TestExonerationPredicate.encode(message.predicate, writer.uint32(18).fork()).ldelim();
    }
    if (message.pageSize !== 0) {
      writer.uint32(32).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(42).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestExonerationsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestExonerationsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocations.push(reader.string());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.predicate = TestExonerationPredicate.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestExonerationsRequest {
    return {
      invocations: globalThis.Array.isArray(object?.invocations)
        ? object.invocations.map((e: any) => globalThis.String(e))
        : [],
      predicate: isSet(object.predicate) ? TestExonerationPredicate.fromJSON(object.predicate) : undefined,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: QueryTestExonerationsRequest): unknown {
    const obj: any = {};
    if (message.invocations?.length) {
      obj.invocations = message.invocations;
    }
    if (message.predicate !== undefined) {
      obj.predicate = TestExonerationPredicate.toJSON(message.predicate);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestExonerationsRequest>, I>>(base?: I): QueryTestExonerationsRequest {
    return QueryTestExonerationsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestExonerationsRequest>, I>>(object: I): QueryTestExonerationsRequest {
    const message = createBaseQueryTestExonerationsRequest() as any;
    message.invocations = object.invocations?.map((e) => e) || [];
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? TestExonerationPredicate.fromPartial(object.predicate)
      : undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseQueryTestExonerationsResponse(): QueryTestExonerationsResponse {
  return { testExonerations: [], nextPageToken: "" };
}

export const QueryTestExonerationsResponse = {
  encode(message: QueryTestExonerationsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testExonerations) {
      TestExoneration.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestExonerationsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestExonerationsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testExonerations.push(TestExoneration.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestExonerationsResponse {
    return {
      testExonerations: globalThis.Array.isArray(object?.testExonerations)
        ? object.testExonerations.map((e: any) => TestExoneration.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: QueryTestExonerationsResponse): unknown {
    const obj: any = {};
    if (message.testExonerations?.length) {
      obj.testExonerations = message.testExonerations.map((e) => TestExoneration.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestExonerationsResponse>, I>>(base?: I): QueryTestExonerationsResponse {
    return QueryTestExonerationsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestExonerationsResponse>, I>>(
    object: I,
  ): QueryTestExonerationsResponse {
    const message = createBaseQueryTestExonerationsResponse() as any;
    message.testExonerations = object.testExonerations?.map((e) => TestExoneration.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryTestResultStatisticsRequest(): QueryTestResultStatisticsRequest {
  return { invocations: [] };
}

export const QueryTestResultStatisticsRequest = {
  encode(message: QueryTestResultStatisticsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.invocations) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestResultStatisticsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestResultStatisticsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocations.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestResultStatisticsRequest {
    return {
      invocations: globalThis.Array.isArray(object?.invocations)
        ? object.invocations.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: QueryTestResultStatisticsRequest): unknown {
    const obj: any = {};
    if (message.invocations?.length) {
      obj.invocations = message.invocations;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestResultStatisticsRequest>, I>>(
    base?: I,
  ): QueryTestResultStatisticsRequest {
    return QueryTestResultStatisticsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestResultStatisticsRequest>, I>>(
    object: I,
  ): QueryTestResultStatisticsRequest {
    const message = createBaseQueryTestResultStatisticsRequest() as any;
    message.invocations = object.invocations?.map((e) => e) || [];
    return message;
  },
};

function createBaseQueryTestResultStatisticsResponse(): QueryTestResultStatisticsResponse {
  return { totalTestResults: "0" };
}

export const QueryTestResultStatisticsResponse = {
  encode(message: QueryTestResultStatisticsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.totalTestResults !== "0") {
      writer.uint32(8).int64(message.totalTestResults);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestResultStatisticsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestResultStatisticsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.totalTestResults = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestResultStatisticsResponse {
    return { totalTestResults: isSet(object.totalTestResults) ? globalThis.String(object.totalTestResults) : "0" };
  },

  toJSON(message: QueryTestResultStatisticsResponse): unknown {
    const obj: any = {};
    if (message.totalTestResults !== "0") {
      obj.totalTestResults = message.totalTestResults;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestResultStatisticsResponse>, I>>(
    base?: I,
  ): QueryTestResultStatisticsResponse {
    return QueryTestResultStatisticsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestResultStatisticsResponse>, I>>(
    object: I,
  ): QueryTestResultStatisticsResponse {
    const message = createBaseQueryTestResultStatisticsResponse() as any;
    message.totalTestResults = object.totalTestResults ?? "0";
    return message;
  },
};

function createBaseGetArtifactRequest(): GetArtifactRequest {
  return { name: "" };
}

export const GetArtifactRequest = {
  encode(message: GetArtifactRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetArtifactRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetArtifactRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetArtifactRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetArtifactRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetArtifactRequest>, I>>(base?: I): GetArtifactRequest {
    return GetArtifactRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetArtifactRequest>, I>>(object: I): GetArtifactRequest {
    const message = createBaseGetArtifactRequest() as any;
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListArtifactsRequest(): ListArtifactsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListArtifactsRequest = {
  encode(message: ListArtifactsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListArtifactsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListArtifactsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListArtifactsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListArtifactsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListArtifactsRequest>, I>>(base?: I): ListArtifactsRequest {
    return ListArtifactsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListArtifactsRequest>, I>>(object: I): ListArtifactsRequest {
    const message = createBaseListArtifactsRequest() as any;
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListArtifactsResponse(): ListArtifactsResponse {
  return { artifacts: [], nextPageToken: "" };
}

export const ListArtifactsResponse = {
  encode(message: ListArtifactsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.artifacts) {
      Artifact.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListArtifactsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListArtifactsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.artifacts.push(Artifact.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListArtifactsResponse {
    return {
      artifacts: globalThis.Array.isArray(object?.artifacts)
        ? object.artifacts.map((e: any) => Artifact.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListArtifactsResponse): unknown {
    const obj: any = {};
    if (message.artifacts?.length) {
      obj.artifacts = message.artifacts.map((e) => Artifact.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ListArtifactsResponse>, I>>(base?: I): ListArtifactsResponse {
    return ListArtifactsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ListArtifactsResponse>, I>>(object: I): ListArtifactsResponse {
    const message = createBaseListArtifactsResponse() as any;
    message.artifacts = object.artifacts?.map((e) => Artifact.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryArtifactsRequest(): QueryArtifactsRequest {
  return { invocations: [], predicate: undefined, pageSize: 0, pageToken: "" };
}

export const QueryArtifactsRequest = {
  encode(message: QueryArtifactsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.invocations) {
      writer.uint32(10).string(v!);
    }
    if (message.predicate !== undefined) {
      ArtifactPredicate.encode(message.predicate, writer.uint32(18).fork()).ldelim();
    }
    if (message.pageSize !== 0) {
      writer.uint32(32).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(42).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryArtifactsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryArtifactsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocations.push(reader.string());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.predicate = ArtifactPredicate.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryArtifactsRequest {
    return {
      invocations: globalThis.Array.isArray(object?.invocations)
        ? object.invocations.map((e: any) => globalThis.String(e))
        : [],
      predicate: isSet(object.predicate) ? ArtifactPredicate.fromJSON(object.predicate) : undefined,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: QueryArtifactsRequest): unknown {
    const obj: any = {};
    if (message.invocations?.length) {
      obj.invocations = message.invocations;
    }
    if (message.predicate !== undefined) {
      obj.predicate = ArtifactPredicate.toJSON(message.predicate);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryArtifactsRequest>, I>>(base?: I): QueryArtifactsRequest {
    return QueryArtifactsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryArtifactsRequest>, I>>(object: I): QueryArtifactsRequest {
    const message = createBaseQueryArtifactsRequest() as any;
    message.invocations = object.invocations?.map((e) => e) || [];
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? ArtifactPredicate.fromPartial(object.predicate)
      : undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseQueryArtifactsResponse(): QueryArtifactsResponse {
  return { artifacts: [], nextPageToken: "" };
}

export const QueryArtifactsResponse = {
  encode(message: QueryArtifactsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.artifacts) {
      Artifact.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryArtifactsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryArtifactsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.artifacts.push(Artifact.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryArtifactsResponse {
    return {
      artifacts: globalThis.Array.isArray(object?.artifacts)
        ? object.artifacts.map((e: any) => Artifact.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: QueryArtifactsResponse): unknown {
    const obj: any = {};
    if (message.artifacts?.length) {
      obj.artifacts = message.artifacts.map((e) => Artifact.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryArtifactsResponse>, I>>(base?: I): QueryArtifactsResponse {
    return QueryArtifactsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryArtifactsResponse>, I>>(object: I): QueryArtifactsResponse {
    const message = createBaseQueryArtifactsResponse() as any;
    message.artifacts = object.artifacts?.map((e) => Artifact.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryTestVariantsRequest(): QueryTestVariantsRequest {
  return { invocations: [], predicate: undefined, resultLimit: 0, pageSize: 0, pageToken: "", readMask: undefined };
}

export const QueryTestVariantsRequest = {
  encode(message: QueryTestVariantsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.invocations) {
      writer.uint32(18).string(v!);
    }
    if (message.predicate !== undefined) {
      TestVariantPredicate.encode(message.predicate, writer.uint32(50).fork()).ldelim();
    }
    if (message.resultLimit !== 0) {
      writer.uint32(64).int32(message.resultLimit);
    }
    if (message.pageSize !== 0) {
      writer.uint32(32).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(42).string(message.pageToken);
    }
    if (message.readMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.readMask), writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestVariantsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestVariantsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.invocations.push(reader.string());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.predicate = TestVariantPredicate.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.resultLimit = reader.int32();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.readMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestVariantsRequest {
    return {
      invocations: globalThis.Array.isArray(object?.invocations)
        ? object.invocations.map((e: any) => globalThis.String(e))
        : [],
      predicate: isSet(object.predicate) ? TestVariantPredicate.fromJSON(object.predicate) : undefined,
      resultLimit: isSet(object.resultLimit) ? globalThis.Number(object.resultLimit) : 0,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      readMask: isSet(object.readMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.readMask)) : undefined,
    };
  },

  toJSON(message: QueryTestVariantsRequest): unknown {
    const obj: any = {};
    if (message.invocations?.length) {
      obj.invocations = message.invocations;
    }
    if (message.predicate !== undefined) {
      obj.predicate = TestVariantPredicate.toJSON(message.predicate);
    }
    if (message.resultLimit !== 0) {
      obj.resultLimit = Math.round(message.resultLimit);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.readMask !== undefined) {
      obj.readMask = FieldMask.toJSON(FieldMask.wrap(message.readMask));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestVariantsRequest>, I>>(base?: I): QueryTestVariantsRequest {
    return QueryTestVariantsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestVariantsRequest>, I>>(object: I): QueryTestVariantsRequest {
    const message = createBaseQueryTestVariantsRequest() as any;
    message.invocations = object.invocations?.map((e) => e) || [];
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? TestVariantPredicate.fromPartial(object.predicate)
      : undefined;
    message.resultLimit = object.resultLimit ?? 0;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.readMask = object.readMask ?? undefined;
    return message;
  },
};

function createBaseQueryTestVariantsResponse(): QueryTestVariantsResponse {
  return { testVariants: [], nextPageToken: "", sources: {} };
}

export const QueryTestVariantsResponse = {
  encode(message: QueryTestVariantsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testVariants) {
      TestVariant.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    Object.entries(message.sources).forEach(([key, value]) => {
      QueryTestVariantsResponse_SourcesEntry.encode({ key: key as any, value }, writer.uint32(26).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestVariantsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestVariantsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testVariants.push(TestVariant.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          const entry3 = QueryTestVariantsResponse_SourcesEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.sources[entry3.key] = entry3.value;
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

  fromJSON(object: any): QueryTestVariantsResponse {
    return {
      testVariants: globalThis.Array.isArray(object?.testVariants)
        ? object.testVariants.map((e: any) => TestVariant.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
      sources: isObject(object.sources)
        ? Object.entries(object.sources).reduce<{ [key: string]: Sources }>((acc, [key, value]) => {
          acc[key] = Sources.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: QueryTestVariantsResponse): unknown {
    const obj: any = {};
    if (message.testVariants?.length) {
      obj.testVariants = message.testVariants.map((e) => TestVariant.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    if (message.sources) {
      const entries = Object.entries(message.sources);
      if (entries.length > 0) {
        obj.sources = {};
        entries.forEach(([k, v]) => {
          obj.sources[k] = Sources.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestVariantsResponse>, I>>(base?: I): QueryTestVariantsResponse {
    return QueryTestVariantsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestVariantsResponse>, I>>(object: I): QueryTestVariantsResponse {
    const message = createBaseQueryTestVariantsResponse() as any;
    message.testVariants = object.testVariants?.map((e) => TestVariant.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    message.sources = Object.entries(object.sources ?? {}).reduce<{ [key: string]: Sources }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Sources.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseQueryTestVariantsResponse_SourcesEntry(): QueryTestVariantsResponse_SourcesEntry {
  return { key: "", value: undefined };
}

export const QueryTestVariantsResponse_SourcesEntry = {
  encode(message: QueryTestVariantsResponse_SourcesEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Sources.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestVariantsResponse_SourcesEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestVariantsResponse_SourcesEntry() as any;
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

          message.value = Sources.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestVariantsResponse_SourcesEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Sources.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: QueryTestVariantsResponse_SourcesEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Sources.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestVariantsResponse_SourcesEntry>, I>>(
    base?: I,
  ): QueryTestVariantsResponse_SourcesEntry {
    return QueryTestVariantsResponse_SourcesEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestVariantsResponse_SourcesEntry>, I>>(
    object: I,
  ): QueryTestVariantsResponse_SourcesEntry {
    const message = createBaseQueryTestVariantsResponse_SourcesEntry() as any;
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? Sources.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseBatchGetTestVariantsRequest(): BatchGetTestVariantsRequest {
  return { invocation: "", testVariants: [], resultLimit: 0 };
}

export const BatchGetTestVariantsRequest = {
  encode(message: BatchGetTestVariantsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.invocation !== "") {
      writer.uint32(10).string(message.invocation);
    }
    for (const v of message.testVariants) {
      BatchGetTestVariantsRequest_TestVariantIdentifier.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.resultLimit !== 0) {
      writer.uint32(24).int32(message.resultLimit);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetTestVariantsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetTestVariantsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocation = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.testVariants.push(BatchGetTestVariantsRequest_TestVariantIdentifier.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.resultLimit = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetTestVariantsRequest {
    return {
      invocation: isSet(object.invocation) ? globalThis.String(object.invocation) : "",
      testVariants: globalThis.Array.isArray(object?.testVariants)
        ? object.testVariants.map((e: any) => BatchGetTestVariantsRequest_TestVariantIdentifier.fromJSON(e))
        : [],
      resultLimit: isSet(object.resultLimit) ? globalThis.Number(object.resultLimit) : 0,
    };
  },

  toJSON(message: BatchGetTestVariantsRequest): unknown {
    const obj: any = {};
    if (message.invocation !== "") {
      obj.invocation = message.invocation;
    }
    if (message.testVariants?.length) {
      obj.testVariants = message.testVariants.map((e) => BatchGetTestVariantsRequest_TestVariantIdentifier.toJSON(e));
    }
    if (message.resultLimit !== 0) {
      obj.resultLimit = Math.round(message.resultLimit);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchGetTestVariantsRequest>, I>>(base?: I): BatchGetTestVariantsRequest {
    return BatchGetTestVariantsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchGetTestVariantsRequest>, I>>(object: I): BatchGetTestVariantsRequest {
    const message = createBaseBatchGetTestVariantsRequest() as any;
    message.invocation = object.invocation ?? "";
    message.testVariants =
      object.testVariants?.map((e) => BatchGetTestVariantsRequest_TestVariantIdentifier.fromPartial(e)) || [];
    message.resultLimit = object.resultLimit ?? 0;
    return message;
  },
};

function createBaseBatchGetTestVariantsRequest_TestVariantIdentifier(): BatchGetTestVariantsRequest_TestVariantIdentifier {
  return { testId: "", variantHash: "" };
}

export const BatchGetTestVariantsRequest_TestVariantIdentifier = {
  encode(
    message: BatchGetTestVariantsRequest_TestVariantIdentifier,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.testId !== "") {
      writer.uint32(10).string(message.testId);
    }
    if (message.variantHash !== "") {
      writer.uint32(18).string(message.variantHash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetTestVariantsRequest_TestVariantIdentifier {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetTestVariantsRequest_TestVariantIdentifier() as any;
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetTestVariantsRequest_TestVariantIdentifier {
    return {
      testId: isSet(object.testId) ? globalThis.String(object.testId) : "",
      variantHash: isSet(object.variantHash) ? globalThis.String(object.variantHash) : "",
    };
  },

  toJSON(message: BatchGetTestVariantsRequest_TestVariantIdentifier): unknown {
    const obj: any = {};
    if (message.testId !== "") {
      obj.testId = message.testId;
    }
    if (message.variantHash !== "") {
      obj.variantHash = message.variantHash;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchGetTestVariantsRequest_TestVariantIdentifier>, I>>(
    base?: I,
  ): BatchGetTestVariantsRequest_TestVariantIdentifier {
    return BatchGetTestVariantsRequest_TestVariantIdentifier.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchGetTestVariantsRequest_TestVariantIdentifier>, I>>(
    object: I,
  ): BatchGetTestVariantsRequest_TestVariantIdentifier {
    const message = createBaseBatchGetTestVariantsRequest_TestVariantIdentifier() as any;
    message.testId = object.testId ?? "";
    message.variantHash = object.variantHash ?? "";
    return message;
  },
};

function createBaseBatchGetTestVariantsResponse(): BatchGetTestVariantsResponse {
  return { testVariants: [], sources: {} };
}

export const BatchGetTestVariantsResponse = {
  encode(message: BatchGetTestVariantsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testVariants) {
      TestVariant.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    Object.entries(message.sources).forEach(([key, value]) => {
      BatchGetTestVariantsResponse_SourcesEntry.encode({ key: key as any, value }, writer.uint32(18).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetTestVariantsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetTestVariantsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testVariants.push(TestVariant.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          const entry2 = BatchGetTestVariantsResponse_SourcesEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.sources[entry2.key] = entry2.value;
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

  fromJSON(object: any): BatchGetTestVariantsResponse {
    return {
      testVariants: globalThis.Array.isArray(object?.testVariants)
        ? object.testVariants.map((e: any) => TestVariant.fromJSON(e))
        : [],
      sources: isObject(object.sources)
        ? Object.entries(object.sources).reduce<{ [key: string]: Sources }>((acc, [key, value]) => {
          acc[key] = Sources.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: BatchGetTestVariantsResponse): unknown {
    const obj: any = {};
    if (message.testVariants?.length) {
      obj.testVariants = message.testVariants.map((e) => TestVariant.toJSON(e));
    }
    if (message.sources) {
      const entries = Object.entries(message.sources);
      if (entries.length > 0) {
        obj.sources = {};
        entries.forEach(([k, v]) => {
          obj.sources[k] = Sources.toJSON(v);
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchGetTestVariantsResponse>, I>>(base?: I): BatchGetTestVariantsResponse {
    return BatchGetTestVariantsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchGetTestVariantsResponse>, I>>(object: I): BatchGetTestVariantsResponse {
    const message = createBaseBatchGetTestVariantsResponse() as any;
    message.testVariants = object.testVariants?.map((e) => TestVariant.fromPartial(e)) || [];
    message.sources = Object.entries(object.sources ?? {}).reduce<{ [key: string]: Sources }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = Sources.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseBatchGetTestVariantsResponse_SourcesEntry(): BatchGetTestVariantsResponse_SourcesEntry {
  return { key: "", value: undefined };
}

export const BatchGetTestVariantsResponse_SourcesEntry = {
  encode(message: BatchGetTestVariantsResponse_SourcesEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      Sources.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetTestVariantsResponse_SourcesEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetTestVariantsResponse_SourcesEntry() as any;
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

          message.value = Sources.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetTestVariantsResponse_SourcesEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? Sources.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: BatchGetTestVariantsResponse_SourcesEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== undefined) {
      obj.value = Sources.toJSON(message.value);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchGetTestVariantsResponse_SourcesEntry>, I>>(
    base?: I,
  ): BatchGetTestVariantsResponse_SourcesEntry {
    return BatchGetTestVariantsResponse_SourcesEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchGetTestVariantsResponse_SourcesEntry>, I>>(
    object: I,
  ): BatchGetTestVariantsResponse_SourcesEntry {
    const message = createBaseBatchGetTestVariantsResponse_SourcesEntry() as any;
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? Sources.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseQueryTestMetadataRequest(): QueryTestMetadataRequest {
  return { project: "", predicate: undefined, pageSize: 0, pageToken: "" };
}

export const QueryTestMetadataRequest = {
  encode(message: QueryTestMetadataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.predicate !== undefined) {
      TestMetadataPredicate.encode(message.predicate, writer.uint32(18).fork()).ldelim();
    }
    if (message.pageSize !== 0) {
      writer.uint32(24).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(34).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestMetadataRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestMetadataRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.predicate = TestMetadataPredicate.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestMetadataRequest {
    return {
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      predicate: isSet(object.predicate) ? TestMetadataPredicate.fromJSON(object.predicate) : undefined,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: QueryTestMetadataRequest): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.predicate !== undefined) {
      obj.predicate = TestMetadataPredicate.toJSON(message.predicate);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestMetadataRequest>, I>>(base?: I): QueryTestMetadataRequest {
    return QueryTestMetadataRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestMetadataRequest>, I>>(object: I): QueryTestMetadataRequest {
    const message = createBaseQueryTestMetadataRequest() as any;
    message.project = object.project ?? "";
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? TestMetadataPredicate.fromPartial(object.predicate)
      : undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseQueryTestMetadataResponse(): QueryTestMetadataResponse {
  return { testMetadata: [], nextPageToken: "" };
}

export const QueryTestMetadataResponse = {
  encode(message: QueryTestMetadataResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.testMetadata) {
      TestMetadataDetail.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryTestMetadataResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryTestMetadataResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.testMetadata.push(TestMetadataDetail.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryTestMetadataResponse {
    return {
      testMetadata: globalThis.Array.isArray(object?.testMetadata)
        ? object.testMetadata.map((e: any) => TestMetadataDetail.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: QueryTestMetadataResponse): unknown {
    const obj: any = {};
    if (message.testMetadata?.length) {
      obj.testMetadata = message.testMetadata.map((e) => TestMetadataDetail.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryTestMetadataResponse>, I>>(base?: I): QueryTestMetadataResponse {
    return QueryTestMetadataResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryTestMetadataResponse>, I>>(object: I): QueryTestMetadataResponse {
    const message = createBaseQueryTestMetadataResponse() as any;
    message.testMetadata = object.testMetadata?.map((e) => TestMetadataDetail.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryNewTestVariantsRequest(): QueryNewTestVariantsRequest {
  return { invocation: "", baseline: "" };
}

export const QueryNewTestVariantsRequest = {
  encode(message: QueryNewTestVariantsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.invocation !== "") {
      writer.uint32(10).string(message.invocation);
    }
    if (message.baseline !== "") {
      writer.uint32(18).string(message.baseline);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryNewTestVariantsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryNewTestVariantsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.invocation = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.baseline = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryNewTestVariantsRequest {
    return {
      invocation: isSet(object.invocation) ? globalThis.String(object.invocation) : "",
      baseline: isSet(object.baseline) ? globalThis.String(object.baseline) : "",
    };
  },

  toJSON(message: QueryNewTestVariantsRequest): unknown {
    const obj: any = {};
    if (message.invocation !== "") {
      obj.invocation = message.invocation;
    }
    if (message.baseline !== "") {
      obj.baseline = message.baseline;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryNewTestVariantsRequest>, I>>(base?: I): QueryNewTestVariantsRequest {
    return QueryNewTestVariantsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryNewTestVariantsRequest>, I>>(object: I): QueryNewTestVariantsRequest {
    const message = createBaseQueryNewTestVariantsRequest() as any;
    message.invocation = object.invocation ?? "";
    message.baseline = object.baseline ?? "";
    return message;
  },
};

function createBaseQueryNewTestVariantsResponse(): QueryNewTestVariantsResponse {
  return { isBaselineReady: false, newTestVariants: [] };
}

export const QueryNewTestVariantsResponse = {
  encode(message: QueryNewTestVariantsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.isBaselineReady === true) {
      writer.uint32(8).bool(message.isBaselineReady);
    }
    for (const v of message.newTestVariants) {
      QueryNewTestVariantsResponse_NewTestVariant.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryNewTestVariantsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryNewTestVariantsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.isBaselineReady = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.newTestVariants.push(QueryNewTestVariantsResponse_NewTestVariant.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryNewTestVariantsResponse {
    return {
      isBaselineReady: isSet(object.isBaselineReady) ? globalThis.Boolean(object.isBaselineReady) : false,
      newTestVariants: globalThis.Array.isArray(object?.newTestVariants)
        ? object.newTestVariants.map((e: any) => QueryNewTestVariantsResponse_NewTestVariant.fromJSON(e))
        : [],
    };
  },

  toJSON(message: QueryNewTestVariantsResponse): unknown {
    const obj: any = {};
    if (message.isBaselineReady === true) {
      obj.isBaselineReady = message.isBaselineReady;
    }
    if (message.newTestVariants?.length) {
      obj.newTestVariants = message.newTestVariants.map((e) => QueryNewTestVariantsResponse_NewTestVariant.toJSON(e));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryNewTestVariantsResponse>, I>>(base?: I): QueryNewTestVariantsResponse {
    return QueryNewTestVariantsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryNewTestVariantsResponse>, I>>(object: I): QueryNewTestVariantsResponse {
    const message = createBaseQueryNewTestVariantsResponse() as any;
    message.isBaselineReady = object.isBaselineReady ?? false;
    message.newTestVariants =
      object.newTestVariants?.map((e) => QueryNewTestVariantsResponse_NewTestVariant.fromPartial(e)) || [];
    return message;
  },
};

function createBaseQueryNewTestVariantsResponse_NewTestVariant(): QueryNewTestVariantsResponse_NewTestVariant {
  return { testId: "", variantHash: "" };
}

export const QueryNewTestVariantsResponse_NewTestVariant = {
  encode(message: QueryNewTestVariantsResponse_NewTestVariant, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.testId !== "") {
      writer.uint32(10).string(message.testId);
    }
    if (message.variantHash !== "") {
      writer.uint32(18).string(message.variantHash);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryNewTestVariantsResponse_NewTestVariant {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryNewTestVariantsResponse_NewTestVariant() as any;
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryNewTestVariantsResponse_NewTestVariant {
    return {
      testId: isSet(object.testId) ? globalThis.String(object.testId) : "",
      variantHash: isSet(object.variantHash) ? globalThis.String(object.variantHash) : "",
    };
  },

  toJSON(message: QueryNewTestVariantsResponse_NewTestVariant): unknown {
    const obj: any = {};
    if (message.testId !== "") {
      obj.testId = message.testId;
    }
    if (message.variantHash !== "") {
      obj.variantHash = message.variantHash;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<QueryNewTestVariantsResponse_NewTestVariant>, I>>(
    base?: I,
  ): QueryNewTestVariantsResponse_NewTestVariant {
    return QueryNewTestVariantsResponse_NewTestVariant.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<QueryNewTestVariantsResponse_NewTestVariant>, I>>(
    object: I,
  ): QueryNewTestVariantsResponse_NewTestVariant {
    const message = createBaseQueryNewTestVariantsResponse_NewTestVariant() as any;
    message.testId = object.testId ?? "";
    message.variantHash = object.variantHash ?? "";
    return message;
  },
};

/** Service to read test results. */
export interface ResultDB {
  /** Retrieves an invocation. */
  GetInvocation(request: GetInvocationRequest): Promise<Invocation>;
  /** Retrieves a test result. */
  GetTestResult(request: GetTestResultRequest): Promise<TestResult>;
  /**
   * Retrieves test results for a parent invocation.
   *
   * Note: response does not contain test results of included invocations.
   * Use QueryTestResults instead.
   */
  ListTestResults(request: ListTestResultsRequest): Promise<ListTestResultsResponse>;
  /** Retrieves a test exoneration. */
  GetTestExoneration(request: GetTestExonerationRequest): Promise<TestExoneration>;
  /**
   * Retrieves test exonerations for a parent invocation.
   *
   * Note: response does not contain test results of included invocations.
   * Use QueryTestExonerations instead.
   */
  ListTestExonerations(request: ListTestExonerationsRequest): Promise<ListTestExonerationsResponse>;
  /**
   * Retrieves test results from an invocation, recursively.
   * Supports invocation inclusions.
   * Supports advanced filtering.
   * Examples: go/resultdb-rpc#querytestresults
   */
  QueryTestResults(request: QueryTestResultsRequest): Promise<QueryTestResultsResponse>;
  /**
   * Retrieves test exonerations from an invocation.
   * Supports invocation inclusions.
   * Supports advanced filtering.
   */
  QueryTestExonerations(request: QueryTestExonerationsRequest): Promise<QueryTestExonerationsResponse>;
  /**
   * Retrieves the test result statistics of an invocation.
   * Currently supports total number of test results belong to the invocation,
   * directly and indirectly.
   */
  QueryTestResultStatistics(request: QueryTestResultStatisticsRequest): Promise<QueryTestResultStatisticsResponse>;
  /**
   * Calculate new test variants by running the difference between the tests
   * run in the given invocation against the submitted test history for the
   * source.
   */
  QueryNewTestVariants(request: QueryNewTestVariantsRequest): Promise<QueryNewTestVariantsResponse>;
  /** Retrieves an artifact. */
  GetArtifact(request: GetArtifactRequest): Promise<Artifact>;
  /**
   * Retrieves artifacts for a parent invocation/testResult.
   *
   * Note: if the parent is an invocation, the response does not contain
   * artifacts of included invocations. Use QueryArtifacts instead.
   */
  ListArtifacts(request: ListArtifactsRequest): Promise<ListArtifactsResponse>;
  /**
   * Retrieves artifacts from an invocation, recursively.
   * Can retrieve artifacts of test results included in the invocation
   * directly or indirectly.
   * Supports invocation inclusions.
   */
  QueryArtifacts(request: QueryArtifactsRequest): Promise<QueryArtifactsResponse>;
  /**
   * Retrieves test variants from an invocation, recursively.
   * Supports invocation inclusions.
   */
  QueryTestVariants(request: QueryTestVariantsRequest): Promise<QueryTestVariantsResponse>;
  /**
   * Retrieves test variants from a single invocation, matching the specified
   * test IDs and hashes.
   */
  BatchGetTestVariants(request: BatchGetTestVariantsRequest): Promise<BatchGetTestVariantsResponse>;
  /** Retrieves test metadata from a LUCI project, matching the predicate. */
  QueryTestMetadata(request: QueryTestMetadataRequest): Promise<QueryTestMetadataResponse>;
}

export const ResultDBServiceName = "luci.resultdb.v1.ResultDB";
export class ResultDBClientImpl implements ResultDB {
  static readonly DEFAULT_SERVICE = ResultDBServiceName;
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || ResultDBServiceName;
    this.rpc = rpc;
    this.GetInvocation = this.GetInvocation.bind(this);
    this.GetTestResult = this.GetTestResult.bind(this);
    this.ListTestResults = this.ListTestResults.bind(this);
    this.GetTestExoneration = this.GetTestExoneration.bind(this);
    this.ListTestExonerations = this.ListTestExonerations.bind(this);
    this.QueryTestResults = this.QueryTestResults.bind(this);
    this.QueryTestExonerations = this.QueryTestExonerations.bind(this);
    this.QueryTestResultStatistics = this.QueryTestResultStatistics.bind(this);
    this.QueryNewTestVariants = this.QueryNewTestVariants.bind(this);
    this.GetArtifact = this.GetArtifact.bind(this);
    this.ListArtifacts = this.ListArtifacts.bind(this);
    this.QueryArtifacts = this.QueryArtifacts.bind(this);
    this.QueryTestVariants = this.QueryTestVariants.bind(this);
    this.BatchGetTestVariants = this.BatchGetTestVariants.bind(this);
    this.QueryTestMetadata = this.QueryTestMetadata.bind(this);
  }
  GetInvocation(request: GetInvocationRequest): Promise<Invocation> {
    const data = GetInvocationRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetInvocation", data);
    return promise.then((data) => Invocation.decode(_m0.Reader.create(data)));
  }

  GetTestResult(request: GetTestResultRequest): Promise<TestResult> {
    const data = GetTestResultRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetTestResult", data);
    return promise.then((data) => TestResult.decode(_m0.Reader.create(data)));
  }

  ListTestResults(request: ListTestResultsRequest): Promise<ListTestResultsResponse> {
    const data = ListTestResultsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListTestResults", data);
    return promise.then((data) => ListTestResultsResponse.decode(_m0.Reader.create(data)));
  }

  GetTestExoneration(request: GetTestExonerationRequest): Promise<TestExoneration> {
    const data = GetTestExonerationRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetTestExoneration", data);
    return promise.then((data) => TestExoneration.decode(_m0.Reader.create(data)));
  }

  ListTestExonerations(request: ListTestExonerationsRequest): Promise<ListTestExonerationsResponse> {
    const data = ListTestExonerationsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListTestExonerations", data);
    return promise.then((data) => ListTestExonerationsResponse.decode(_m0.Reader.create(data)));
  }

  QueryTestResults(request: QueryTestResultsRequest): Promise<QueryTestResultsResponse> {
    const data = QueryTestResultsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryTestResults", data);
    return promise.then((data) => QueryTestResultsResponse.decode(_m0.Reader.create(data)));
  }

  QueryTestExonerations(request: QueryTestExonerationsRequest): Promise<QueryTestExonerationsResponse> {
    const data = QueryTestExonerationsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryTestExonerations", data);
    return promise.then((data) => QueryTestExonerationsResponse.decode(_m0.Reader.create(data)));
  }

  QueryTestResultStatistics(request: QueryTestResultStatisticsRequest): Promise<QueryTestResultStatisticsResponse> {
    const data = QueryTestResultStatisticsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryTestResultStatistics", data);
    return promise.then((data) => QueryTestResultStatisticsResponse.decode(_m0.Reader.create(data)));
  }

  QueryNewTestVariants(request: QueryNewTestVariantsRequest): Promise<QueryNewTestVariantsResponse> {
    const data = QueryNewTestVariantsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryNewTestVariants", data);
    return promise.then((data) => QueryNewTestVariantsResponse.decode(_m0.Reader.create(data)));
  }

  GetArtifact(request: GetArtifactRequest): Promise<Artifact> {
    const data = GetArtifactRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetArtifact", data);
    return promise.then((data) => Artifact.decode(_m0.Reader.create(data)));
  }

  ListArtifacts(request: ListArtifactsRequest): Promise<ListArtifactsResponse> {
    const data = ListArtifactsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListArtifacts", data);
    return promise.then((data) => ListArtifactsResponse.decode(_m0.Reader.create(data)));
  }

  QueryArtifacts(request: QueryArtifactsRequest): Promise<QueryArtifactsResponse> {
    const data = QueryArtifactsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryArtifacts", data);
    return promise.then((data) => QueryArtifactsResponse.decode(_m0.Reader.create(data)));
  }

  QueryTestVariants(request: QueryTestVariantsRequest): Promise<QueryTestVariantsResponse> {
    const data = QueryTestVariantsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryTestVariants", data);
    return promise.then((data) => QueryTestVariantsResponse.decode(_m0.Reader.create(data)));
  }

  BatchGetTestVariants(request: BatchGetTestVariantsRequest): Promise<BatchGetTestVariantsResponse> {
    const data = BatchGetTestVariantsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "BatchGetTestVariants", data);
    return promise.then((data) => BatchGetTestVariantsResponse.decode(_m0.Reader.create(data)));
  }

  QueryTestMetadata(request: QueryTestMetadataRequest): Promise<QueryTestMetadataResponse> {
    const data = QueryTestMetadataRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "QueryTestMetadata", data);
    return promise.then((data) => QueryTestMetadataResponse.decode(_m0.Reader.create(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

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
