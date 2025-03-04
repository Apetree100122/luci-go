/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../../../../google/protobuf/duration.pb";
import { FieldMask } from "../../../../google/protobuf/field_mask.pb";
import { Struct } from "../../../../google/protobuf/struct.pb";
import { Status as Status1 } from "../../../../google/rpc/status.pb";
import { StructMask } from "../../common/proto/structmask/structmask.pb";
import { Build } from "./build.pb";
import { BuilderID } from "./builder_common.pb";
import {
  Executable,
  GerritChange,
  GitilesCommit,
  RequestedDimension,
  Status,
  statusFromJSON,
  statusToJSON,
  StringPair,
  TimeRange,
  Trinary,
  trinaryFromJSON,
  trinaryToJSON,
} from "./common.pb";
import { NotificationConfig } from "./notification.pb";

export const protobufPackage = "buildbucket.v2";

/** A request message for GetBuild RPC. */
export interface GetBuildRequest {
  /**
   * Build ID.
   * Mutually exclusive with builder and number.
   */
  readonly id: string;
  /**
   * Builder of the build.
   * Requires number. Mutually exclusive with id.
   */
  readonly builder:
    | BuilderID
    | undefined;
  /**
   * Build number.
   * Requires builder. Mutually exclusive with id.
   */
  readonly buildNumber: number;
  /**
   * Fields to include in the response.
   *
   * DEPRECATED: Use mask instead.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   *
   * Supports advanced semantics, see
   * https://chromium.googlesource.com/infra/luci/luci-py/+/f9ae69a37c4bdd0e08a8b0f7e123f6e403e774eb/appengine/components/components/protoutil/field_masks.py#7
   * In particular, if the client needs only some output properties, they
   * can be requested with paths "output.properties.fields.foo".
   *
   * @deprecated
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * What portion of the Build message to return.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly mask: BuildMask | undefined;
}

/** A request message for SearchBuilds RPC. */
export interface SearchBuildsRequest {
  /** Returned builds must satisfy this predicate. Required. */
  readonly predicate:
    | BuildPredicate
    | undefined;
  /**
   * Fields to include in the response, see GetBuildRequest.fields.
   *
   * DEPRECATED: Use mask instead.
   *
   * Note that this applies to the response, not each build, so e.g. steps must
   * be requested with a path "builds.*.steps".
   *
   * @deprecated
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * What portion of the Build message to return.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly mask:
    | BuildMask
    | undefined;
  /**
   * Number of builds to return.
   * Defaults to 100.
   * Any value >1000 is interpreted as 1000.
   */
  readonly pageSize: number;
  /**
   * Value of SearchBuildsResponse.next_page_token from the previous response.
   * Use it to continue searching.
   * The predicate and page_size in this request MUST be exactly same as in the
   * previous request.
   */
  readonly pageToken: string;
}

/** A response message for SearchBuilds RPC. */
export interface SearchBuildsResponse {
  /**
   * Search results.
   *
   * Ordered by build ID, descending. IDs are monotonically decreasing, so in
   * other words the order is newest-to-oldest.
   */
  readonly builds: readonly Build[];
  /** Value for SearchBuildsRequest.page_token to continue searching. */
  readonly nextPageToken: string;
}

/** A request message for Batch RPC. */
export interface BatchRequest {
  /**
   * Requests to execute in a single batch.
   *
   * * All requests are executed in their own individual transactions.
   * * BatchRequest as a whole is not transactional.
   * * There's no guaranteed order of execution between batch items (i.e.
   *   consider them to all operate independently).
   * * There is a limit of 200 requests per batch.
   */
  readonly requests: readonly BatchRequest_Request[];
}

/** One request in a batch. */
export interface BatchRequest_Request {
  readonly getBuild?: GetBuildRequest | undefined;
  readonly searchBuilds?: SearchBuildsRequest | undefined;
  readonly scheduleBuild?: ScheduleBuildRequest | undefined;
  readonly cancelBuild?: CancelBuildRequest | undefined;
  readonly getBuildStatus?: GetBuildStatusRequest | undefined;
}

/** A response message for Batch RPC. */
export interface BatchResponse {
  /** Responses in the same order as BatchRequest.requests. */
  readonly responses: readonly BatchResponse_Response[];
}

/** Response a BatchRequest.Response. */
export interface BatchResponse_Response {
  readonly getBuild?: Build | undefined;
  readonly searchBuilds?: SearchBuildsResponse | undefined;
  readonly scheduleBuild?: Build | undefined;
  readonly cancelBuild?: Build | undefined;
  readonly getBuildStatus?:
    | Build
    | undefined;
  /** Error code and details of the unsuccessful RPC. */
  readonly error?: Status1 | undefined;
}

/** A request message for UpdateBuild RPC. */
export interface UpdateBuildRequest {
  /** Build to update, with new field values. */
  readonly build:
    | Build
    | undefined;
  /**
   * Build fields to update.
   * See also
   * https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#fieldmask
   *
   * Currently supports only the following path strings:
   * - build.output
   * - build.output.properties
   * - build.output.gitiles_commit
   * - build.output.status
   * - build.output.status_details
   * - build.output.summary_markdown
   * - build.status
   * - build.status_details
   * - build.steps
   * - build.summary_markdown
   * - build.tags
   * - build.infra.buildbucket.agent.output
   * - build.infra.buildbucket.agent.purposes
   *
   * Note, "build.output.status" is required explicitly to update the field.
   * If there is only "build.output" in update_mask, build.output.status will not
   * be updated.
   *
   * If omitted, Buildbucket will update the Build's update_time, but nothing else.
   */
  readonly updateMask:
    | readonly string[]
    | undefined;
  /**
   * Fields to include in the response. See also GetBuildRequest.fields.
   *
   * DEPRECATED: Use mask instead.
   *
   * @deprecated
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * What portion of the Build message to return.
   *
   * If not set, an empty build will be returned.
   */
  readonly mask: BuildMask | undefined;
}

/**
 * A request message for ScheduleBuild RPC.
 *
 * Next ID: 24.
 */
export interface ScheduleBuildRequest {
  /**
   * * STRONGLY RECOMMENDED **.
   * A unique string id used for detecting duplicate requests.
   * Should be unique at least per requesting identity.
   * Used to dedup build scheduling requests with same id within 1 min.
   * If a build was successfully scheduled with the same request id in the past
   * minute, the existing build will be returned.
   */
  readonly requestId: string;
  /**
   * ID of a build to retry as is or altered.
   * When specified, fields below default to the values in the template build.
   */
  readonly templateBuildId: string;
  /**
   * Value for Build.builder. See its comments.
   * Required, unless template_build_id is specified.
   */
  readonly builder:
    | BuilderID
    | undefined;
  /**
   * DEPRECATED
   *
   * Set "luci.buildbucket.canary_software" in `experiments` instead.
   *
   * YES sets "luci.buildbucket.canary_software" to true in `experiments`.
   * NO sets "luci.buildbucket.canary_software" to false in `experiments`.
   */
  readonly canary: Trinary;
  /**
   * DEPRECATED
   *
   * Set "luci.non_production" in `experiments` instead.
   *
   * YES sets "luci.non_production" to true in `experiments`.
   * NO sets "luci.non_production" to false in `experiments`.
   */
  readonly experimental: Trinary;
  /**
   * Sets (or prevents) these experiments on the scheduled build.
   *
   * See `Builder.experiments` for well-known experiments.
   */
  readonly experiments: { [key: string]: boolean };
  /**
   * Properties to include in Build.input.properties.
   *
   * Input properties of the created build are result of merging server-defined
   * properties and properties in this field.
   * Each property in this field defines a new or replaces an existing property
   * on the server.
   * If the server config does not allow overriding/adding the property, the
   * request will fail with InvalidArgument error code.
   * A server-defined property cannot be removed, but its value can be
   * replaced with null.
   *
   * Reserved property paths:
   *   ["$recipe_engine/buildbucket"]
   *   ["$recipe_engine/runtime", "is_experimental"]
   *   ["$recipe_engine/runtime", "is_luci"]
   *   ["branch"]
   *   ["buildbucket"]
   *   ["buildername"]
   *   ["repository"]
   *
   * The Builder configuration specifies which top-level property names are
   * overridable via the `allowed_property_overrides` field. ScheduleBuild
   * requests which attempt to override a property which isn't allowed will
   * fail with InvalidArgument.
   *
   * V1 equivalent: corresponds to "properties" key in "parameters_json".
   */
  readonly properties:
    | { readonly [key: string]: any }
    | undefined;
  /**
   * Value for Build.input.gitiles_commit.
   *
   * Setting this field will cause the created build to have a "buildset"
   * tag with value "commit/gitiles/{hostname}/{project}/+/{id}".
   *
   * GitilesCommit objects MUST have host, project, ref fields set.
   *
   * V1 equivalent: supersedes "revision" property and "buildset"
   * tag that starts with "commit/gitiles/".
   */
  readonly gitilesCommit:
    | GitilesCommit
    | undefined;
  /**
   * Value for Build.input.gerrit_changes.
   * Usually present in tryjobs, set by CQ, Gerrit, git-cl-try.
   * Applied on top of gitiles_commit if specified, otherwise tip of the tree.
   * All GerritChange fields are required.
   *
   * Setting this field will cause the created build to have a "buildset"
   * tag with value "patch/gerrit/{hostname}/{change}/{patchset}"
   * for each change.
   *
   * V1 equivalent: supersedes patch_* properties and "buildset"
   * tag that starts with "patch/gerrit/".
   */
  readonly gerritChanges: readonly GerritChange[];
  /**
   * Tags to include in Build.tags of the created build, see Build.tags
   * comments.
   * Note: tags of the created build may include other tags defined on the
   * server.
   */
  readonly tags: readonly StringPair[];
  /**
   * Overrides default dimensions defined by builder config or template build.
   *
   * A set of entries with the same key defines a new or replaces an existing
   * dimension with the same key.
   * If the config does not allow overriding/adding the dimension, the request
   * will fail with InvalidArgument error code.
   *
   * After merging, dimensions with empty value will be excluded.
   *
   * Note: For the same key dimensions, it won't allow to pass empty and
   * non-empty values at the same time in the request.
   *
   * Note: "caches" and "pool" dimensions may only be specified in builder
   * configs. Setting them hear will fail the request.
   *
   * A dimension expiration must be a multiple of 1min.
   */
  readonly dimensions: readonly RequestedDimension[];
  /**
   * If not zero, overrides swarming task priority.
   * See also Build.infra.swarming.priority.
   */
  readonly priority: number;
  /** A per-build notification configuration. */
  readonly notify:
    | NotificationConfig
    | undefined;
  /**
   * Fields to include in the response. See also GetBuildRequest.fields.
   *
   * DEPRECATED: Use mask instead.
   *
   * @deprecated
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * What portion of the Build message to return.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly mask:
    | BuildMask
    | undefined;
  /** Value for Build.critical. */
  readonly critical: Trinary;
  /**
   * Overrides Builder.exe in the config.
   * Supported subfields: cipd_version.
   */
  readonly exe:
    | Executable
    | undefined;
  /** Swarming specific part of the build request. */
  readonly swarming:
    | ScheduleBuildRequest_Swarming
    | undefined;
  /**
   * Maximum build pending time.
   *
   * If set, overrides the default `expiration_secs` set in builder config.
   * Only supports seconds precision for now.
   * For more information, see Build.scheduling_timeout in build.proto.
   */
  readonly schedulingTimeout:
    | Duration
    | undefined;
  /**
   * Maximum build execution time.
   *
   * If set, overrides the default `execution_timeout_secs` set in builder config.
   * Only supports seconds precision for now.
   * For more information, see Build.execution_timeout in build.proto.
   */
  readonly executionTimeout:
    | Duration
    | undefined;
  /**
   * Amount of cleanup time after execution_timeout.
   *
   * If set, overrides the default `grace_period` set in builder config.
   * Only supports seconds precision for now.
   * For more information, see Build.grace_period in build.proto.
   */
  readonly gracePeriod:
    | Duration
    | undefined;
  /**
   * Whether or not this request constitutes a dry run.
   *
   * A dry run returns the build proto without actually scheduling it. All
   * fields except those which can only be computed at run-time are filled in.
   * Does not cause side-effects. When batching, all requests must specify the
   * same value for dry_run.
   */
  readonly dryRun: boolean;
  /**
   * Flag to control if the build can outlive its parent.
   *
   * If the value is UNSET, it means this build doesn't have any parent, so
   * the request must not have a head with any BuildToken.
   *
   * If the value is anything other than UNSET, then the BuildToken for the
   * parent build must be set as a header.
   * Note: it's not currently possible to establish parent/child relationship
   * except via the parent build at the time the build is launched.
   *
   * If the value is NO, it means that the build SHOULD reach a terminal status
   * (SUCCESS, FAILURE, INFRA_FAILURE or CANCELED) before its parent. If the
   * child fails to do so, Buildbucket will cancel it some time after the
   * parent build reaches a terminal status.
   *
   * A build that can outlive its parent can also outlive its parent's ancestors.
   *
   * If schedule a build without parent, this field must be UNSET.
   *
   * If schedule a build with parent, this field should be YES or NO.
   * But UNSET is also accepted for now, and it has the same effect as YES.
   * TODO(crbug.com/1031205): after the parent tracking feature is stable,
   * require this field to be set when scheduling a build with parent.
   */
  readonly canOutliveParent: Trinary;
  /** Value for Build.retriable. */
  readonly retriable: Trinary;
  /**
   * Input for scheduling a build in the shadow bucket.
   *
   * If this field is set, it means the build to be scheduled will
   * * be scheduled in the shadow bucket of the requested bucket, with shadow
   *   adjustments on service_account, dimensions and properties.
   * * inherit its parent build's agent input and agent source if it has a parent.
   */
  readonly shadowInput: ScheduleBuildRequest_ShadowInput | undefined;
}

export interface ScheduleBuildRequest_ExperimentsEntry {
  readonly key: string;
  readonly value: boolean;
}

/** Swarming specific part of the build request. */
export interface ScheduleBuildRequest_Swarming {
  /**
   * If specified, parent_run_id should match actual Swarming task run ID the
   * caller is running as and results in swarming server ensuring that the newly
   * triggered build will not outlive its parent.
   *
   * Typical use is for triggering and waiting on child build(s) from within
   * 1 parent build and if child build(s) on their own aren't useful. Then,
   * if parent build ends for whatever reason, all not yet finished child
   * builds aren't useful and it's desirable to terminate them, too.
   *
   * If the Builder config does not specify a swarming backend, the request
   * will fail with InvalidArgument error code.
   *
   * The parent_run_id is assumed to be from the same swarming server as the
   * one the new build is to be executed on. The ScheduleBuildRequest doesn't
   * check if parent_run_id refers to actually existing task, but eventually
   * the new build will fail if so.
   */
  readonly parentRunId: string;
}

/** Information for scheduling a build as a shadow build. */
export interface ScheduleBuildRequest_ShadowInput {
}

/** A request message for CancelBuild RPC. */
export interface CancelBuildRequest {
  /** ID of the build to cancel. */
  readonly id: string;
  /**
   * Required. Value for Build.cancellation_markdown. Will be appended to
   * Build.summary_markdown when exporting to bigquery and returned via GetBuild.
   */
  readonly summaryMarkdown: string;
  /**
   * Fields to include in the response. See also GetBuildRequest.fields.
   *
   * DEPRECATED: Use mask instead.
   *
   * @deprecated
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * What portion of the Build message to return.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly mask: BuildMask | undefined;
}

/** A request message for CreateBuild RPC. */
export interface CreateBuildRequest {
  /** The Build to be created. */
  readonly build:
    | Build
    | undefined;
  /**
   * A unique identifier for this request.
   * A random UUID is recommended.
   * This request is only idempotent if a `request_id` is provided.
   */
  readonly requestId: string;
  /**
   * What portion of the Build message to return.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly mask: BuildMask | undefined;
}

/** A request message for SynthesizeBuild RPC. */
export interface SynthesizeBuildRequest {
  /**
   * ID of a build to use as the template.
   * Mutually exclusive with builder.
   */
  readonly templateBuildId: string;
  /**
   * Value for Build.builder. See its comments.
   * Required, unless template_build_id is specified.
   */
  readonly builder:
    | BuilderID
    | undefined;
  /**
   * Sets (or prevents) these experiments on the synthesized build.
   *
   * See `Builder.experiments` for well-known experiments.
   */
  readonly experiments: { [key: string]: boolean };
}

export interface SynthesizeBuildRequest_ExperimentsEntry {
  readonly key: string;
  readonly value: boolean;
}

/** A request message for StartBuild RPC. */
export interface StartBuildRequest {
  /** A nonce to deduplicate requests. */
  readonly requestId: string;
  /** Id of the build to start. */
  readonly buildId: string;
  /** Id of the task running the started build. */
  readonly taskId: string;
}

/** A response message for StartBuild RPC. */
export interface StartBuildResponse {
  /** The whole proto of the started build. */
  readonly build:
    | Build
    | undefined;
  /** a build token for agent to use when making subsequent UpdateBuild calls. */
  readonly updateBuildToken: string;
}

/** A request message for GetBuildStatus RPC. */
export interface GetBuildStatusRequest {
  /**
   * Build ID.
   * Mutually exclusive with builder and number.
   */
  readonly id: string;
  /**
   * Builder of the build.
   * Requires number. Mutually exclusive with id.
   */
  readonly builder:
    | BuilderID
    | undefined;
  /**
   * Build number.
   * Requires builder. Mutually exclusive with id.
   */
  readonly buildNumber: number;
}

/** Defines a subset of Build fields and properties to return. */
export interface BuildMask {
  /**
   * Fields of the Build proto to include.
   *
   * Follows the standard FieldMask semantics as documented at e.g.
   * https://pkg.go.dev/google.golang.org/protobuf/types/known/fieldmaskpb.
   *
   * If not set, the default mask is used, see Build message comments for the
   * list of fields returned by default.
   */
  readonly fields:
    | readonly string[]
    | undefined;
  /**
   * Defines a subset of `input.properties` to return.
   *
   * When not empty, implicitly adds the corresponding field to `fields`.
   */
  readonly inputProperties: readonly StructMask[];
  /**
   * Defines a subset of `output.properties` to return.
   *
   * When not empty, implicitly adds the corresponding field to `fields`.
   */
  readonly outputProperties: readonly StructMask[];
  /**
   * Defines a subset of `infra.buildbucket.requested_properties` to return.
   *
   * When not empty, implicitly adds the corresponding field to `fields`.
   */
  readonly requestedProperties: readonly StructMask[];
  /**
   * Flag for including all fields.
   *
   * Mutually exclusive with `fields`, `input_properties`, `output_properties`,
   * and `requested_properties`.
   */
  readonly allFields: boolean;
  /**
   * A status to filter returned `steps` by. If unspecified, no filter is
   * applied. Otherwise filters by the union of the given statuses.
   *
   * No effect unless `fields` specifies that `steps` should be returned or
   * `all_fields` is true.
   */
  readonly stepStatus: readonly Status[];
}

/**
 * A build predicate.
 *
 * At least one of the following fields is required: builder, gerrit_changes and
 * git_commits.
 * If a field value is empty, it is ignored, unless stated otherwise.
 */
export interface BuildPredicate {
  /** A build must be in this builder. */
  readonly builder:
    | BuilderID
    | undefined;
  /** A build must have this status. */
  readonly status: Status;
  /** A build's Build.Input.gerrit_changes must include ALL of these changes. */
  readonly gerritChanges: readonly GerritChange[];
  /**
   * DEPRECATED
   *
   * Never implemented.
   */
  readonly outputGitilesCommit:
    | GitilesCommit
    | undefined;
  /** A build must be created by this identity. */
  readonly createdBy: string;
  /**
   * A build must have ALL of these tags.
   * For "ANY of these tags" make separate RPCs.
   */
  readonly tags: readonly StringPair[];
  /**
   * A build must have been created within the specified range.
   * Both boundaries are optional.
   */
  readonly createTime:
    | TimeRange
    | undefined;
  /**
   * If false (the default), equivalent to filtering by experiment
   * "-luci.non_production".
   *
   * If true, has no effect (both production and non_production builds will be
   * returned).
   *
   * NOTE: If you explicitly search for non_production builds with the experiment
   * filter "+luci.non_production", this is implied to be true.
   *
   * See `Builder.experiments` for well-known experiments.
   */
  readonly includeExperimental: boolean;
  /** A build must be in this build range. */
  readonly build:
    | BuildRange
    | undefined;
  /**
   * DEPRECATED
   *
   * If YES, equivalent to filtering by experiment
   * "+luci.buildbucket.canary_software".
   *
   * If NO, equivalent to filtering by experiment
   * "-luci.buildbucket.canary_software".
   *
   * See `Builder.experiments` for well-known experiments.
   */
  readonly canary: Trinary;
  /**
   * A list of experiments to include or exclude from the search results.
   *
   * Each entry should look like "[-+]$experiment_name".
   *
   * A "+" prefix means that returned builds MUST have that experiment set.
   * A "-" prefix means that returned builds MUST NOT have that experiment set
   *   AND that experiment was known for the builder at the time the build
   *   was scheduled (either via `Builder.experiments` or via
   *   `ScheduleBuildRequest.experiments`). Well-known experiments are always
   *   considered to be available.
   */
  readonly experiments: readonly string[];
  /**
   * A build ID.
   *
   * Returned builds will be descendants of this build (e.g. "100" means
   * "any build transitively scheduled starting from build 100").
   *
   * Mutually exclusive with `child_of`.
   */
  readonly descendantOf: string;
  /**
   * A build ID.
   *
   * Returned builds will be only the immediate children of this build.
   *
   * Mutually exclusive with `descendant_of`.
   */
  readonly childOf: string;
}

/** Open build range. */
export interface BuildRange {
  /** Inclusive lower (less recent build) boundary. Optional. */
  readonly startBuildId: string;
  /** Inclusive upper (more recent build) boundary. Optional. */
  readonly endBuildId: string;
}

function createBaseGetBuildRequest(): GetBuildRequest {
  return { id: "0", builder: undefined, buildNumber: 0, fields: undefined, mask: undefined };
}

export const GetBuildRequest = {
  encode(message: GetBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "0") {
      writer.uint32(8).int64(message.id);
    }
    if (message.builder !== undefined) {
      BuilderID.encode(message.builder, writer.uint32(18).fork()).ldelim();
    }
    if (message.buildNumber !== 0) {
      writer.uint32(24).int32(message.buildNumber);
    }
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(802).fork()).ldelim();
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(810).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.id = longToString(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.builder = BuilderID.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.buildNumber = reader.int32();
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 101:
          if (tag !== 810) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetBuildRequest {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "0",
      builder: isSet(object.builder) ? BuilderID.fromJSON(object.builder) : undefined,
      buildNumber: isSet(object.buildNumber) ? globalThis.Number(object.buildNumber) : 0,
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
    };
  },

  toJSON(message: GetBuildRequest): unknown {
    const obj: any = {};
    if (message.id !== "0") {
      obj.id = message.id;
    }
    if (message.builder !== undefined) {
      obj.builder = BuilderID.toJSON(message.builder);
    }
    if (message.buildNumber !== 0) {
      obj.buildNumber = Math.round(message.buildNumber);
    }
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetBuildRequest>, I>>(base?: I): GetBuildRequest {
    return GetBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetBuildRequest>, I>>(object: I): GetBuildRequest {
    const message = createBaseGetBuildRequest() as any;
    message.id = object.id ?? "0";
    message.builder = (object.builder !== undefined && object.builder !== null)
      ? BuilderID.fromPartial(object.builder)
      : undefined;
    message.buildNumber = object.buildNumber ?? 0;
    message.fields = object.fields ?? undefined;
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    return message;
  },
};

function createBaseSearchBuildsRequest(): SearchBuildsRequest {
  return { predicate: undefined, fields: undefined, mask: undefined, pageSize: 0, pageToken: "" };
}

export const SearchBuildsRequest = {
  encode(message: SearchBuildsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.predicate !== undefined) {
      BuildPredicate.encode(message.predicate, writer.uint32(10).fork()).ldelim();
    }
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(802).fork()).ldelim();
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(826).fork()).ldelim();
    }
    if (message.pageSize !== 0) {
      writer.uint32(808).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(818).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchBuildsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchBuildsRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.predicate = BuildPredicate.decode(reader, reader.uint32());
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 103:
          if (tag !== 826) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
        case 101:
          if (tag !== 808) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 102:
          if (tag !== 818) {
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

  fromJSON(object: any): SearchBuildsRequest {
    return {
      predicate: isSet(object.predicate) ? BuildPredicate.fromJSON(object.predicate) : undefined,
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: SearchBuildsRequest): unknown {
    const obj: any = {};
    if (message.predicate !== undefined) {
      obj.predicate = BuildPredicate.toJSON(message.predicate);
    }
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SearchBuildsRequest>, I>>(base?: I): SearchBuildsRequest {
    return SearchBuildsRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<SearchBuildsRequest>, I>>(object: I): SearchBuildsRequest {
    const message = createBaseSearchBuildsRequest() as any;
    message.predicate = (object.predicate !== undefined && object.predicate !== null)
      ? BuildPredicate.fromPartial(object.predicate)
      : undefined;
    message.fields = object.fields ?? undefined;
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseSearchBuildsResponse(): SearchBuildsResponse {
  return { builds: [], nextPageToken: "" };
}

export const SearchBuildsResponse = {
  encode(message: SearchBuildsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.builds) {
      Build.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(802).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchBuildsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchBuildsResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.builds.push(Build.decode(reader, reader.uint32()));
          continue;
        case 100:
          if (tag !== 802) {
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

  fromJSON(object: any): SearchBuildsResponse {
    return {
      builds: globalThis.Array.isArray(object?.builds) ? object.builds.map((e: any) => Build.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchBuildsResponse): unknown {
    const obj: any = {};
    if (message.builds?.length) {
      obj.builds = message.builds.map((e) => Build.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SearchBuildsResponse>, I>>(base?: I): SearchBuildsResponse {
    return SearchBuildsResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<SearchBuildsResponse>, I>>(object: I): SearchBuildsResponse {
    const message = createBaseSearchBuildsResponse() as any;
    message.builds = object.builds?.map((e) => Build.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseBatchRequest(): BatchRequest {
  return { requests: [] };
}

export const BatchRequest = {
  encode(message: BatchRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.requests) {
      BatchRequest_Request.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.requests.push(BatchRequest_Request.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchRequest {
    return {
      requests: globalThis.Array.isArray(object?.requests)
        ? object.requests.map((e: any) => BatchRequest_Request.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchRequest): unknown {
    const obj: any = {};
    if (message.requests?.length) {
      obj.requests = message.requests.map((e) => BatchRequest_Request.toJSON(e));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchRequest>, I>>(base?: I): BatchRequest {
    return BatchRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchRequest>, I>>(object: I): BatchRequest {
    const message = createBaseBatchRequest() as any;
    message.requests = object.requests?.map((e) => BatchRequest_Request.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchRequest_Request(): BatchRequest_Request {
  return {
    getBuild: undefined,
    searchBuilds: undefined,
    scheduleBuild: undefined,
    cancelBuild: undefined,
    getBuildStatus: undefined,
  };
}

export const BatchRequest_Request = {
  encode(message: BatchRequest_Request, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.getBuild !== undefined) {
      GetBuildRequest.encode(message.getBuild, writer.uint32(10).fork()).ldelim();
    }
    if (message.searchBuilds !== undefined) {
      SearchBuildsRequest.encode(message.searchBuilds, writer.uint32(18).fork()).ldelim();
    }
    if (message.scheduleBuild !== undefined) {
      ScheduleBuildRequest.encode(message.scheduleBuild, writer.uint32(26).fork()).ldelim();
    }
    if (message.cancelBuild !== undefined) {
      CancelBuildRequest.encode(message.cancelBuild, writer.uint32(34).fork()).ldelim();
    }
    if (message.getBuildStatus !== undefined) {
      GetBuildStatusRequest.encode(message.getBuildStatus, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchRequest_Request {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchRequest_Request() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.getBuild = GetBuildRequest.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.searchBuilds = SearchBuildsRequest.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.scheduleBuild = ScheduleBuildRequest.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.cancelBuild = CancelBuildRequest.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.getBuildStatus = GetBuildStatusRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchRequest_Request {
    return {
      getBuild: isSet(object.getBuild) ? GetBuildRequest.fromJSON(object.getBuild) : undefined,
      searchBuilds: isSet(object.searchBuilds) ? SearchBuildsRequest.fromJSON(object.searchBuilds) : undefined,
      scheduleBuild: isSet(object.scheduleBuild) ? ScheduleBuildRequest.fromJSON(object.scheduleBuild) : undefined,
      cancelBuild: isSet(object.cancelBuild) ? CancelBuildRequest.fromJSON(object.cancelBuild) : undefined,
      getBuildStatus: isSet(object.getBuildStatus) ? GetBuildStatusRequest.fromJSON(object.getBuildStatus) : undefined,
    };
  },

  toJSON(message: BatchRequest_Request): unknown {
    const obj: any = {};
    if (message.getBuild !== undefined) {
      obj.getBuild = GetBuildRequest.toJSON(message.getBuild);
    }
    if (message.searchBuilds !== undefined) {
      obj.searchBuilds = SearchBuildsRequest.toJSON(message.searchBuilds);
    }
    if (message.scheduleBuild !== undefined) {
      obj.scheduleBuild = ScheduleBuildRequest.toJSON(message.scheduleBuild);
    }
    if (message.cancelBuild !== undefined) {
      obj.cancelBuild = CancelBuildRequest.toJSON(message.cancelBuild);
    }
    if (message.getBuildStatus !== undefined) {
      obj.getBuildStatus = GetBuildStatusRequest.toJSON(message.getBuildStatus);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchRequest_Request>, I>>(base?: I): BatchRequest_Request {
    return BatchRequest_Request.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchRequest_Request>, I>>(object: I): BatchRequest_Request {
    const message = createBaseBatchRequest_Request() as any;
    message.getBuild = (object.getBuild !== undefined && object.getBuild !== null)
      ? GetBuildRequest.fromPartial(object.getBuild)
      : undefined;
    message.searchBuilds = (object.searchBuilds !== undefined && object.searchBuilds !== null)
      ? SearchBuildsRequest.fromPartial(object.searchBuilds)
      : undefined;
    message.scheduleBuild = (object.scheduleBuild !== undefined && object.scheduleBuild !== null)
      ? ScheduleBuildRequest.fromPartial(object.scheduleBuild)
      : undefined;
    message.cancelBuild = (object.cancelBuild !== undefined && object.cancelBuild !== null)
      ? CancelBuildRequest.fromPartial(object.cancelBuild)
      : undefined;
    message.getBuildStatus = (object.getBuildStatus !== undefined && object.getBuildStatus !== null)
      ? GetBuildStatusRequest.fromPartial(object.getBuildStatus)
      : undefined;
    return message;
  },
};

function createBaseBatchResponse(): BatchResponse {
  return { responses: [] };
}

export const BatchResponse = {
  encode(message: BatchResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.responses) {
      BatchResponse_Response.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.responses.push(BatchResponse_Response.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchResponse {
    return {
      responses: globalThis.Array.isArray(object?.responses)
        ? object.responses.map((e: any) => BatchResponse_Response.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchResponse): unknown {
    const obj: any = {};
    if (message.responses?.length) {
      obj.responses = message.responses.map((e) => BatchResponse_Response.toJSON(e));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchResponse>, I>>(base?: I): BatchResponse {
    return BatchResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchResponse>, I>>(object: I): BatchResponse {
    const message = createBaseBatchResponse() as any;
    message.responses = object.responses?.map((e) => BatchResponse_Response.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchResponse_Response(): BatchResponse_Response {
  return {
    getBuild: undefined,
    searchBuilds: undefined,
    scheduleBuild: undefined,
    cancelBuild: undefined,
    getBuildStatus: undefined,
    error: undefined,
  };
}

export const BatchResponse_Response = {
  encode(message: BatchResponse_Response, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.getBuild !== undefined) {
      Build.encode(message.getBuild, writer.uint32(10).fork()).ldelim();
    }
    if (message.searchBuilds !== undefined) {
      SearchBuildsResponse.encode(message.searchBuilds, writer.uint32(18).fork()).ldelim();
    }
    if (message.scheduleBuild !== undefined) {
      Build.encode(message.scheduleBuild, writer.uint32(26).fork()).ldelim();
    }
    if (message.cancelBuild !== undefined) {
      Build.encode(message.cancelBuild, writer.uint32(34).fork()).ldelim();
    }
    if (message.getBuildStatus !== undefined) {
      Build.encode(message.getBuildStatus, writer.uint32(42).fork()).ldelim();
    }
    if (message.error !== undefined) {
      Status1.encode(message.error, writer.uint32(802).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchResponse_Response {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchResponse_Response() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.getBuild = Build.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.searchBuilds = SearchBuildsResponse.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.scheduleBuild = Build.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.cancelBuild = Build.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.getBuildStatus = Build.decode(reader, reader.uint32());
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.error = Status1.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchResponse_Response {
    return {
      getBuild: isSet(object.getBuild) ? Build.fromJSON(object.getBuild) : undefined,
      searchBuilds: isSet(object.searchBuilds) ? SearchBuildsResponse.fromJSON(object.searchBuilds) : undefined,
      scheduleBuild: isSet(object.scheduleBuild) ? Build.fromJSON(object.scheduleBuild) : undefined,
      cancelBuild: isSet(object.cancelBuild) ? Build.fromJSON(object.cancelBuild) : undefined,
      getBuildStatus: isSet(object.getBuildStatus) ? Build.fromJSON(object.getBuildStatus) : undefined,
      error: isSet(object.error) ? Status1.fromJSON(object.error) : undefined,
    };
  },

  toJSON(message: BatchResponse_Response): unknown {
    const obj: any = {};
    if (message.getBuild !== undefined) {
      obj.getBuild = Build.toJSON(message.getBuild);
    }
    if (message.searchBuilds !== undefined) {
      obj.searchBuilds = SearchBuildsResponse.toJSON(message.searchBuilds);
    }
    if (message.scheduleBuild !== undefined) {
      obj.scheduleBuild = Build.toJSON(message.scheduleBuild);
    }
    if (message.cancelBuild !== undefined) {
      obj.cancelBuild = Build.toJSON(message.cancelBuild);
    }
    if (message.getBuildStatus !== undefined) {
      obj.getBuildStatus = Build.toJSON(message.getBuildStatus);
    }
    if (message.error !== undefined) {
      obj.error = Status1.toJSON(message.error);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BatchResponse_Response>, I>>(base?: I): BatchResponse_Response {
    return BatchResponse_Response.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BatchResponse_Response>, I>>(object: I): BatchResponse_Response {
    const message = createBaseBatchResponse_Response() as any;
    message.getBuild = (object.getBuild !== undefined && object.getBuild !== null)
      ? Build.fromPartial(object.getBuild)
      : undefined;
    message.searchBuilds = (object.searchBuilds !== undefined && object.searchBuilds !== null)
      ? SearchBuildsResponse.fromPartial(object.searchBuilds)
      : undefined;
    message.scheduleBuild = (object.scheduleBuild !== undefined && object.scheduleBuild !== null)
      ? Build.fromPartial(object.scheduleBuild)
      : undefined;
    message.cancelBuild = (object.cancelBuild !== undefined && object.cancelBuild !== null)
      ? Build.fromPartial(object.cancelBuild)
      : undefined;
    message.getBuildStatus = (object.getBuildStatus !== undefined && object.getBuildStatus !== null)
      ? Build.fromPartial(object.getBuildStatus)
      : undefined;
    message.error = (object.error !== undefined && object.error !== null)
      ? Status1.fromPartial(object.error)
      : undefined;
    return message;
  },
};

function createBaseUpdateBuildRequest(): UpdateBuildRequest {
  return { build: undefined, updateMask: undefined, fields: undefined, mask: undefined };
}

export const UpdateBuildRequest = {
  encode(message: UpdateBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.build !== undefined) {
      Build.encode(message.build, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(802).fork()).ldelim();
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(810).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.build = Build.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 101:
          if (tag !== 810) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateBuildRequest {
    return {
      build: isSet(object.build) ? Build.fromJSON(object.build) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
    };
  },

  toJSON(message: UpdateBuildRequest): unknown {
    const obj: any = {};
    if (message.build !== undefined) {
      obj.build = Build.toJSON(message.build);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<UpdateBuildRequest>, I>>(base?: I): UpdateBuildRequest {
    return UpdateBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<UpdateBuildRequest>, I>>(object: I): UpdateBuildRequest {
    const message = createBaseUpdateBuildRequest() as any;
    message.build = (object.build !== undefined && object.build !== null) ? Build.fromPartial(object.build) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    message.fields = object.fields ?? undefined;
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    return message;
  },
};

function createBaseScheduleBuildRequest(): ScheduleBuildRequest {
  return {
    requestId: "",
    templateBuildId: "0",
    builder: undefined,
    canary: 0,
    experimental: 0,
    experiments: {},
    properties: undefined,
    gitilesCommit: undefined,
    gerritChanges: [],
    tags: [],
    dimensions: [],
    priority: 0,
    notify: undefined,
    fields: undefined,
    mask: undefined,
    critical: 0,
    exe: undefined,
    swarming: undefined,
    schedulingTimeout: undefined,
    executionTimeout: undefined,
    gracePeriod: undefined,
    dryRun: false,
    canOutliveParent: 0,
    retriable: 0,
    shadowInput: undefined,
  };
}

export const ScheduleBuildRequest = {
  encode(message: ScheduleBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.requestId !== "") {
      writer.uint32(10).string(message.requestId);
    }
    if (message.templateBuildId !== "0") {
      writer.uint32(16).int64(message.templateBuildId);
    }
    if (message.builder !== undefined) {
      BuilderID.encode(message.builder, writer.uint32(26).fork()).ldelim();
    }
    if (message.canary !== 0) {
      writer.uint32(32).int32(message.canary);
    }
    if (message.experimental !== 0) {
      writer.uint32(40).int32(message.experimental);
    }
    Object.entries(message.experiments).forEach(([key, value]) => {
      ScheduleBuildRequest_ExperimentsEntry.encode({ key: key as any, value }, writer.uint32(130).fork()).ldelim();
    });
    if (message.properties !== undefined) {
      Struct.encode(Struct.wrap(message.properties), writer.uint32(50).fork()).ldelim();
    }
    if (message.gitilesCommit !== undefined) {
      GitilesCommit.encode(message.gitilesCommit, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.gerritChanges) {
      GerritChange.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.tags) {
      StringPair.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.dimensions) {
      RequestedDimension.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    if (message.priority !== 0) {
      writer.uint32(88).int32(message.priority);
    }
    if (message.notify !== undefined) {
      NotificationConfig.encode(message.notify, writer.uint32(98).fork()).ldelim();
    }
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(802).fork()).ldelim();
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(810).fork()).ldelim();
    }
    if (message.critical !== 0) {
      writer.uint32(104).int32(message.critical);
    }
    if (message.exe !== undefined) {
      Executable.encode(message.exe, writer.uint32(114).fork()).ldelim();
    }
    if (message.swarming !== undefined) {
      ScheduleBuildRequest_Swarming.encode(message.swarming, writer.uint32(122).fork()).ldelim();
    }
    if (message.schedulingTimeout !== undefined) {
      Duration.encode(message.schedulingTimeout, writer.uint32(138).fork()).ldelim();
    }
    if (message.executionTimeout !== undefined) {
      Duration.encode(message.executionTimeout, writer.uint32(146).fork()).ldelim();
    }
    if (message.gracePeriod !== undefined) {
      Duration.encode(message.gracePeriod, writer.uint32(154).fork()).ldelim();
    }
    if (message.dryRun === true) {
      writer.uint32(160).bool(message.dryRun);
    }
    if (message.canOutliveParent !== 0) {
      writer.uint32(168).int32(message.canOutliveParent);
    }
    if (message.retriable !== 0) {
      writer.uint32(176).int32(message.retriable);
    }
    if (message.shadowInput !== undefined) {
      ScheduleBuildRequest_ShadowInput.encode(message.shadowInput, writer.uint32(186).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ScheduleBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.requestId = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.templateBuildId = longToString(reader.int64() as Long);
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.builder = BuilderID.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.canary = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.experimental = reader.int32() as any;
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          const entry16 = ScheduleBuildRequest_ExperimentsEntry.decode(reader, reader.uint32());
          if (entry16.value !== undefined) {
            message.experiments[entry16.key] = entry16.value;
          }
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.properties = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.gitilesCommit = GitilesCommit.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.gerritChanges.push(GerritChange.decode(reader, reader.uint32()));
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.tags.push(StringPair.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.dimensions.push(RequestedDimension.decode(reader, reader.uint32()));
          continue;
        case 11:
          if (tag !== 88) {
            break;
          }

          message.priority = reader.int32();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.notify = NotificationConfig.decode(reader, reader.uint32());
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 101:
          if (tag !== 810) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
        case 13:
          if (tag !== 104) {
            break;
          }

          message.critical = reader.int32() as any;
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.exe = Executable.decode(reader, reader.uint32());
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.swarming = ScheduleBuildRequest_Swarming.decode(reader, reader.uint32());
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.schedulingTimeout = Duration.decode(reader, reader.uint32());
          continue;
        case 18:
          if (tag !== 146) {
            break;
          }

          message.executionTimeout = Duration.decode(reader, reader.uint32());
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.gracePeriod = Duration.decode(reader, reader.uint32());
          continue;
        case 20:
          if (tag !== 160) {
            break;
          }

          message.dryRun = reader.bool();
          continue;
        case 21:
          if (tag !== 168) {
            break;
          }

          message.canOutliveParent = reader.int32() as any;
          continue;
        case 22:
          if (tag !== 176) {
            break;
          }

          message.retriable = reader.int32() as any;
          continue;
        case 23:
          if (tag !== 186) {
            break;
          }

          message.shadowInput = ScheduleBuildRequest_ShadowInput.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ScheduleBuildRequest {
    return {
      requestId: isSet(object.requestId) ? globalThis.String(object.requestId) : "",
      templateBuildId: isSet(object.templateBuildId) ? globalThis.String(object.templateBuildId) : "0",
      builder: isSet(object.builder) ? BuilderID.fromJSON(object.builder) : undefined,
      canary: isSet(object.canary) ? trinaryFromJSON(object.canary) : 0,
      experimental: isSet(object.experimental) ? trinaryFromJSON(object.experimental) : 0,
      experiments: isObject(object.experiments)
        ? Object.entries(object.experiments).reduce<{ [key: string]: boolean }>((acc, [key, value]) => {
          acc[key] = Boolean(value);
          return acc;
        }, {})
        : {},
      properties: isObject(object.properties) ? object.properties : undefined,
      gitilesCommit: isSet(object.gitilesCommit) ? GitilesCommit.fromJSON(object.gitilesCommit) : undefined,
      gerritChanges: globalThis.Array.isArray(object?.gerritChanges)
        ? object.gerritChanges.map((e: any) => GerritChange.fromJSON(e))
        : [],
      tags: globalThis.Array.isArray(object?.tags) ? object.tags.map((e: any) => StringPair.fromJSON(e)) : [],
      dimensions: globalThis.Array.isArray(object?.dimensions)
        ? object.dimensions.map((e: any) => RequestedDimension.fromJSON(e))
        : [],
      priority: isSet(object.priority) ? globalThis.Number(object.priority) : 0,
      notify: isSet(object.notify) ? NotificationConfig.fromJSON(object.notify) : undefined,
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
      critical: isSet(object.critical) ? trinaryFromJSON(object.critical) : 0,
      exe: isSet(object.exe) ? Executable.fromJSON(object.exe) : undefined,
      swarming: isSet(object.swarming) ? ScheduleBuildRequest_Swarming.fromJSON(object.swarming) : undefined,
      schedulingTimeout: isSet(object.schedulingTimeout) ? Duration.fromJSON(object.schedulingTimeout) : undefined,
      executionTimeout: isSet(object.executionTimeout) ? Duration.fromJSON(object.executionTimeout) : undefined,
      gracePeriod: isSet(object.gracePeriod) ? Duration.fromJSON(object.gracePeriod) : undefined,
      dryRun: isSet(object.dryRun) ? globalThis.Boolean(object.dryRun) : false,
      canOutliveParent: isSet(object.canOutliveParent) ? trinaryFromJSON(object.canOutliveParent) : 0,
      retriable: isSet(object.retriable) ? trinaryFromJSON(object.retriable) : 0,
      shadowInput: isSet(object.shadowInput)
        ? ScheduleBuildRequest_ShadowInput.fromJSON(object.shadowInput)
        : undefined,
    };
  },

  toJSON(message: ScheduleBuildRequest): unknown {
    const obj: any = {};
    if (message.requestId !== "") {
      obj.requestId = message.requestId;
    }
    if (message.templateBuildId !== "0") {
      obj.templateBuildId = message.templateBuildId;
    }
    if (message.builder !== undefined) {
      obj.builder = BuilderID.toJSON(message.builder);
    }
    if (message.canary !== 0) {
      obj.canary = trinaryToJSON(message.canary);
    }
    if (message.experimental !== 0) {
      obj.experimental = trinaryToJSON(message.experimental);
    }
    if (message.experiments) {
      const entries = Object.entries(message.experiments);
      if (entries.length > 0) {
        obj.experiments = {};
        entries.forEach(([k, v]) => {
          obj.experiments[k] = v;
        });
      }
    }
    if (message.properties !== undefined) {
      obj.properties = message.properties;
    }
    if (message.gitilesCommit !== undefined) {
      obj.gitilesCommit = GitilesCommit.toJSON(message.gitilesCommit);
    }
    if (message.gerritChanges?.length) {
      obj.gerritChanges = message.gerritChanges.map((e) => GerritChange.toJSON(e));
    }
    if (message.tags?.length) {
      obj.tags = message.tags.map((e) => StringPair.toJSON(e));
    }
    if (message.dimensions?.length) {
      obj.dimensions = message.dimensions.map((e) => RequestedDimension.toJSON(e));
    }
    if (message.priority !== 0) {
      obj.priority = Math.round(message.priority);
    }
    if (message.notify !== undefined) {
      obj.notify = NotificationConfig.toJSON(message.notify);
    }
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    if (message.critical !== 0) {
      obj.critical = trinaryToJSON(message.critical);
    }
    if (message.exe !== undefined) {
      obj.exe = Executable.toJSON(message.exe);
    }
    if (message.swarming !== undefined) {
      obj.swarming = ScheduleBuildRequest_Swarming.toJSON(message.swarming);
    }
    if (message.schedulingTimeout !== undefined) {
      obj.schedulingTimeout = Duration.toJSON(message.schedulingTimeout);
    }
    if (message.executionTimeout !== undefined) {
      obj.executionTimeout = Duration.toJSON(message.executionTimeout);
    }
    if (message.gracePeriod !== undefined) {
      obj.gracePeriod = Duration.toJSON(message.gracePeriod);
    }
    if (message.dryRun === true) {
      obj.dryRun = message.dryRun;
    }
    if (message.canOutliveParent !== 0) {
      obj.canOutliveParent = trinaryToJSON(message.canOutliveParent);
    }
    if (message.retriable !== 0) {
      obj.retriable = trinaryToJSON(message.retriable);
    }
    if (message.shadowInput !== undefined) {
      obj.shadowInput = ScheduleBuildRequest_ShadowInput.toJSON(message.shadowInput);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ScheduleBuildRequest>, I>>(base?: I): ScheduleBuildRequest {
    return ScheduleBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ScheduleBuildRequest>, I>>(object: I): ScheduleBuildRequest {
    const message = createBaseScheduleBuildRequest() as any;
    message.requestId = object.requestId ?? "";
    message.templateBuildId = object.templateBuildId ?? "0";
    message.builder = (object.builder !== undefined && object.builder !== null)
      ? BuilderID.fromPartial(object.builder)
      : undefined;
    message.canary = object.canary ?? 0;
    message.experimental = object.experimental ?? 0;
    message.experiments = Object.entries(object.experiments ?? {}).reduce<{ [key: string]: boolean }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = globalThis.Boolean(value);
        }
        return acc;
      },
      {},
    );
    message.properties = object.properties ?? undefined;
    message.gitilesCommit = (object.gitilesCommit !== undefined && object.gitilesCommit !== null)
      ? GitilesCommit.fromPartial(object.gitilesCommit)
      : undefined;
    message.gerritChanges = object.gerritChanges?.map((e) => GerritChange.fromPartial(e)) || [];
    message.tags = object.tags?.map((e) => StringPair.fromPartial(e)) || [];
    message.dimensions = object.dimensions?.map((e) => RequestedDimension.fromPartial(e)) || [];
    message.priority = object.priority ?? 0;
    message.notify = (object.notify !== undefined && object.notify !== null)
      ? NotificationConfig.fromPartial(object.notify)
      : undefined;
    message.fields = object.fields ?? undefined;
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    message.critical = object.critical ?? 0;
    message.exe = (object.exe !== undefined && object.exe !== null) ? Executable.fromPartial(object.exe) : undefined;
    message.swarming = (object.swarming !== undefined && object.swarming !== null)
      ? ScheduleBuildRequest_Swarming.fromPartial(object.swarming)
      : undefined;
    message.schedulingTimeout = (object.schedulingTimeout !== undefined && object.schedulingTimeout !== null)
      ? Duration.fromPartial(object.schedulingTimeout)
      : undefined;
    message.executionTimeout = (object.executionTimeout !== undefined && object.executionTimeout !== null)
      ? Duration.fromPartial(object.executionTimeout)
      : undefined;
    message.gracePeriod = (object.gracePeriod !== undefined && object.gracePeriod !== null)
      ? Duration.fromPartial(object.gracePeriod)
      : undefined;
    message.dryRun = object.dryRun ?? false;
    message.canOutliveParent = object.canOutliveParent ?? 0;
    message.retriable = object.retriable ?? 0;
    message.shadowInput = (object.shadowInput !== undefined && object.shadowInput !== null)
      ? ScheduleBuildRequest_ShadowInput.fromPartial(object.shadowInput)
      : undefined;
    return message;
  },
};

function createBaseScheduleBuildRequest_ExperimentsEntry(): ScheduleBuildRequest_ExperimentsEntry {
  return { key: "", value: false };
}

export const ScheduleBuildRequest_ExperimentsEntry = {
  encode(message: ScheduleBuildRequest_ExperimentsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value === true) {
      writer.uint32(16).bool(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ScheduleBuildRequest_ExperimentsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleBuildRequest_ExperimentsEntry() as any;
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
          if (tag !== 16) {
            break;
          }

          message.value = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ScheduleBuildRequest_ExperimentsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.Boolean(object.value) : false,
    };
  },

  toJSON(message: ScheduleBuildRequest_ExperimentsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value === true) {
      obj.value = message.value;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ScheduleBuildRequest_ExperimentsEntry>, I>>(
    base?: I,
  ): ScheduleBuildRequest_ExperimentsEntry {
    return ScheduleBuildRequest_ExperimentsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ScheduleBuildRequest_ExperimentsEntry>, I>>(
    object: I,
  ): ScheduleBuildRequest_ExperimentsEntry {
    const message = createBaseScheduleBuildRequest_ExperimentsEntry() as any;
    message.key = object.key ?? "";
    message.value = object.value ?? false;
    return message;
  },
};

function createBaseScheduleBuildRequest_Swarming(): ScheduleBuildRequest_Swarming {
  return { parentRunId: "" };
}

export const ScheduleBuildRequest_Swarming = {
  encode(message: ScheduleBuildRequest_Swarming, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parentRunId !== "") {
      writer.uint32(10).string(message.parentRunId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ScheduleBuildRequest_Swarming {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleBuildRequest_Swarming() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parentRunId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ScheduleBuildRequest_Swarming {
    return { parentRunId: isSet(object.parentRunId) ? globalThis.String(object.parentRunId) : "" };
  },

  toJSON(message: ScheduleBuildRequest_Swarming): unknown {
    const obj: any = {};
    if (message.parentRunId !== "") {
      obj.parentRunId = message.parentRunId;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<ScheduleBuildRequest_Swarming>, I>>(base?: I): ScheduleBuildRequest_Swarming {
    return ScheduleBuildRequest_Swarming.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ScheduleBuildRequest_Swarming>, I>>(
    object: I,
  ): ScheduleBuildRequest_Swarming {
    const message = createBaseScheduleBuildRequest_Swarming() as any;
    message.parentRunId = object.parentRunId ?? "";
    return message;
  },
};

function createBaseScheduleBuildRequest_ShadowInput(): ScheduleBuildRequest_ShadowInput {
  return {};
}

export const ScheduleBuildRequest_ShadowInput = {
  encode(_: ScheduleBuildRequest_ShadowInput, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ScheduleBuildRequest_ShadowInput {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleBuildRequest_ShadowInput() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): ScheduleBuildRequest_ShadowInput {
    return {};
  },

  toJSON(_: ScheduleBuildRequest_ShadowInput): unknown {
    const obj: any = {};
    return obj;
  },

  create<I extends Exact<DeepPartial<ScheduleBuildRequest_ShadowInput>, I>>(
    base?: I,
  ): ScheduleBuildRequest_ShadowInput {
    return ScheduleBuildRequest_ShadowInput.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<ScheduleBuildRequest_ShadowInput>, I>>(
    _: I,
  ): ScheduleBuildRequest_ShadowInput {
    const message = createBaseScheduleBuildRequest_ShadowInput() as any;
    return message;
  },
};

function createBaseCancelBuildRequest(): CancelBuildRequest {
  return { id: "0", summaryMarkdown: "", fields: undefined, mask: undefined };
}

export const CancelBuildRequest = {
  encode(message: CancelBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "0") {
      writer.uint32(8).int64(message.id);
    }
    if (message.summaryMarkdown !== "") {
      writer.uint32(18).string(message.summaryMarkdown);
    }
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(802).fork()).ldelim();
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(810).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CancelBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCancelBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.id = longToString(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.summaryMarkdown = reader.string();
          continue;
        case 100:
          if (tag !== 802) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 101:
          if (tag !== 810) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CancelBuildRequest {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "0",
      summaryMarkdown: isSet(object.summaryMarkdown) ? globalThis.String(object.summaryMarkdown) : "",
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
    };
  },

  toJSON(message: CancelBuildRequest): unknown {
    const obj: any = {};
    if (message.id !== "0") {
      obj.id = message.id;
    }
    if (message.summaryMarkdown !== "") {
      obj.summaryMarkdown = message.summaryMarkdown;
    }
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<CancelBuildRequest>, I>>(base?: I): CancelBuildRequest {
    return CancelBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<CancelBuildRequest>, I>>(object: I): CancelBuildRequest {
    const message = createBaseCancelBuildRequest() as any;
    message.id = object.id ?? "0";
    message.summaryMarkdown = object.summaryMarkdown ?? "";
    message.fields = object.fields ?? undefined;
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    return message;
  },
};

function createBaseCreateBuildRequest(): CreateBuildRequest {
  return { build: undefined, requestId: "", mask: undefined };
}

export const CreateBuildRequest = {
  encode(message: CreateBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.build !== undefined) {
      Build.encode(message.build, writer.uint32(10).fork()).ldelim();
    }
    if (message.requestId !== "") {
      writer.uint32(18).string(message.requestId);
    }
    if (message.mask !== undefined) {
      BuildMask.encode(message.mask, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.build = Build.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.requestId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.mask = BuildMask.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateBuildRequest {
    return {
      build: isSet(object.build) ? Build.fromJSON(object.build) : undefined,
      requestId: isSet(object.requestId) ? globalThis.String(object.requestId) : "",
      mask: isSet(object.mask) ? BuildMask.fromJSON(object.mask) : undefined,
    };
  },

  toJSON(message: CreateBuildRequest): unknown {
    const obj: any = {};
    if (message.build !== undefined) {
      obj.build = Build.toJSON(message.build);
    }
    if (message.requestId !== "") {
      obj.requestId = message.requestId;
    }
    if (message.mask !== undefined) {
      obj.mask = BuildMask.toJSON(message.mask);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<CreateBuildRequest>, I>>(base?: I): CreateBuildRequest {
    return CreateBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<CreateBuildRequest>, I>>(object: I): CreateBuildRequest {
    const message = createBaseCreateBuildRequest() as any;
    message.build = (object.build !== undefined && object.build !== null) ? Build.fromPartial(object.build) : undefined;
    message.requestId = object.requestId ?? "";
    message.mask = (object.mask !== undefined && object.mask !== null) ? BuildMask.fromPartial(object.mask) : undefined;
    return message;
  },
};

function createBaseSynthesizeBuildRequest(): SynthesizeBuildRequest {
  return { templateBuildId: "0", builder: undefined, experiments: {} };
}

export const SynthesizeBuildRequest = {
  encode(message: SynthesizeBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.templateBuildId !== "0") {
      writer.uint32(8).int64(message.templateBuildId);
    }
    if (message.builder !== undefined) {
      BuilderID.encode(message.builder, writer.uint32(18).fork()).ldelim();
    }
    Object.entries(message.experiments).forEach(([key, value]) => {
      SynthesizeBuildRequest_ExperimentsEntry.encode({ key: key as any, value }, writer.uint32(26).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SynthesizeBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSynthesizeBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.templateBuildId = longToString(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.builder = BuilderID.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          const entry3 = SynthesizeBuildRequest_ExperimentsEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.experiments[entry3.key] = entry3.value;
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

  fromJSON(object: any): SynthesizeBuildRequest {
    return {
      templateBuildId: isSet(object.templateBuildId) ? globalThis.String(object.templateBuildId) : "0",
      builder: isSet(object.builder) ? BuilderID.fromJSON(object.builder) : undefined,
      experiments: isObject(object.experiments)
        ? Object.entries(object.experiments).reduce<{ [key: string]: boolean }>((acc, [key, value]) => {
          acc[key] = Boolean(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: SynthesizeBuildRequest): unknown {
    const obj: any = {};
    if (message.templateBuildId !== "0") {
      obj.templateBuildId = message.templateBuildId;
    }
    if (message.builder !== undefined) {
      obj.builder = BuilderID.toJSON(message.builder);
    }
    if (message.experiments) {
      const entries = Object.entries(message.experiments);
      if (entries.length > 0) {
        obj.experiments = {};
        entries.forEach(([k, v]) => {
          obj.experiments[k] = v;
        });
      }
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SynthesizeBuildRequest>, I>>(base?: I): SynthesizeBuildRequest {
    return SynthesizeBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<SynthesizeBuildRequest>, I>>(object: I): SynthesizeBuildRequest {
    const message = createBaseSynthesizeBuildRequest() as any;
    message.templateBuildId = object.templateBuildId ?? "0";
    message.builder = (object.builder !== undefined && object.builder !== null)
      ? BuilderID.fromPartial(object.builder)
      : undefined;
    message.experiments = Object.entries(object.experiments ?? {}).reduce<{ [key: string]: boolean }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = globalThis.Boolean(value);
        }
        return acc;
      },
      {},
    );
    return message;
  },
};

function createBaseSynthesizeBuildRequest_ExperimentsEntry(): SynthesizeBuildRequest_ExperimentsEntry {
  return { key: "", value: false };
}

export const SynthesizeBuildRequest_ExperimentsEntry = {
  encode(message: SynthesizeBuildRequest_ExperimentsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value === true) {
      writer.uint32(16).bool(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SynthesizeBuildRequest_ExperimentsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSynthesizeBuildRequest_ExperimentsEntry() as any;
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
          if (tag !== 16) {
            break;
          }

          message.value = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SynthesizeBuildRequest_ExperimentsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.Boolean(object.value) : false,
    };
  },

  toJSON(message: SynthesizeBuildRequest_ExperimentsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value === true) {
      obj.value = message.value;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<SynthesizeBuildRequest_ExperimentsEntry>, I>>(
    base?: I,
  ): SynthesizeBuildRequest_ExperimentsEntry {
    return SynthesizeBuildRequest_ExperimentsEntry.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<SynthesizeBuildRequest_ExperimentsEntry>, I>>(
    object: I,
  ): SynthesizeBuildRequest_ExperimentsEntry {
    const message = createBaseSynthesizeBuildRequest_ExperimentsEntry() as any;
    message.key = object.key ?? "";
    message.value = object.value ?? false;
    return message;
  },
};

function createBaseStartBuildRequest(): StartBuildRequest {
  return { requestId: "", buildId: "0", taskId: "" };
}

export const StartBuildRequest = {
  encode(message: StartBuildRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.requestId !== "") {
      writer.uint32(10).string(message.requestId);
    }
    if (message.buildId !== "0") {
      writer.uint32(16).int64(message.buildId);
    }
    if (message.taskId !== "") {
      writer.uint32(26).string(message.taskId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StartBuildRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStartBuildRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.requestId = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.buildId = longToString(reader.int64() as Long);
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.taskId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StartBuildRequest {
    return {
      requestId: isSet(object.requestId) ? globalThis.String(object.requestId) : "",
      buildId: isSet(object.buildId) ? globalThis.String(object.buildId) : "0",
      taskId: isSet(object.taskId) ? globalThis.String(object.taskId) : "",
    };
  },

  toJSON(message: StartBuildRequest): unknown {
    const obj: any = {};
    if (message.requestId !== "") {
      obj.requestId = message.requestId;
    }
    if (message.buildId !== "0") {
      obj.buildId = message.buildId;
    }
    if (message.taskId !== "") {
      obj.taskId = message.taskId;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<StartBuildRequest>, I>>(base?: I): StartBuildRequest {
    return StartBuildRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<StartBuildRequest>, I>>(object: I): StartBuildRequest {
    const message = createBaseStartBuildRequest() as any;
    message.requestId = object.requestId ?? "";
    message.buildId = object.buildId ?? "0";
    message.taskId = object.taskId ?? "";
    return message;
  },
};

function createBaseStartBuildResponse(): StartBuildResponse {
  return { build: undefined, updateBuildToken: "" };
}

export const StartBuildResponse = {
  encode(message: StartBuildResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.build !== undefined) {
      Build.encode(message.build, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateBuildToken !== "") {
      writer.uint32(18).string(message.updateBuildToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StartBuildResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStartBuildResponse() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.build = Build.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateBuildToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StartBuildResponse {
    return {
      build: isSet(object.build) ? Build.fromJSON(object.build) : undefined,
      updateBuildToken: isSet(object.updateBuildToken) ? globalThis.String(object.updateBuildToken) : "",
    };
  },

  toJSON(message: StartBuildResponse): unknown {
    const obj: any = {};
    if (message.build !== undefined) {
      obj.build = Build.toJSON(message.build);
    }
    if (message.updateBuildToken !== "") {
      obj.updateBuildToken = message.updateBuildToken;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<StartBuildResponse>, I>>(base?: I): StartBuildResponse {
    return StartBuildResponse.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<StartBuildResponse>, I>>(object: I): StartBuildResponse {
    const message = createBaseStartBuildResponse() as any;
    message.build = (object.build !== undefined && object.build !== null) ? Build.fromPartial(object.build) : undefined;
    message.updateBuildToken = object.updateBuildToken ?? "";
    return message;
  },
};

function createBaseGetBuildStatusRequest(): GetBuildStatusRequest {
  return { id: "0", builder: undefined, buildNumber: 0 };
}

export const GetBuildStatusRequest = {
  encode(message: GetBuildStatusRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "0") {
      writer.uint32(8).int64(message.id);
    }
    if (message.builder !== undefined) {
      BuilderID.encode(message.builder, writer.uint32(18).fork()).ldelim();
    }
    if (message.buildNumber !== 0) {
      writer.uint32(24).int32(message.buildNumber);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetBuildStatusRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetBuildStatusRequest() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.id = longToString(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.builder = BuilderID.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.buildNumber = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetBuildStatusRequest {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "0",
      builder: isSet(object.builder) ? BuilderID.fromJSON(object.builder) : undefined,
      buildNumber: isSet(object.buildNumber) ? globalThis.Number(object.buildNumber) : 0,
    };
  },

  toJSON(message: GetBuildStatusRequest): unknown {
    const obj: any = {};
    if (message.id !== "0") {
      obj.id = message.id;
    }
    if (message.builder !== undefined) {
      obj.builder = BuilderID.toJSON(message.builder);
    }
    if (message.buildNumber !== 0) {
      obj.buildNumber = Math.round(message.buildNumber);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<GetBuildStatusRequest>, I>>(base?: I): GetBuildStatusRequest {
    return GetBuildStatusRequest.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<GetBuildStatusRequest>, I>>(object: I): GetBuildStatusRequest {
    const message = createBaseGetBuildStatusRequest() as any;
    message.id = object.id ?? "0";
    message.builder = (object.builder !== undefined && object.builder !== null)
      ? BuilderID.fromPartial(object.builder)
      : undefined;
    message.buildNumber = object.buildNumber ?? 0;
    return message;
  },
};

function createBaseBuildMask(): BuildMask {
  return {
    fields: undefined,
    inputProperties: [],
    outputProperties: [],
    requestedProperties: [],
    allFields: false,
    stepStatus: [],
  };
}

export const BuildMask = {
  encode(message: BuildMask, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fields !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.fields), writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.inputProperties) {
      StructMask.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.outputProperties) {
      StructMask.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.requestedProperties) {
      StructMask.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.allFields === true) {
      writer.uint32(40).bool(message.allFields);
    }
    writer.uint32(50).fork();
    for (const v of message.stepStatus) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BuildMask {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBuildMask() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.fields = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.inputProperties.push(StructMask.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.outputProperties.push(StructMask.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.requestedProperties.push(StructMask.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.allFields = reader.bool();
          continue;
        case 6:
          if (tag === 48) {
            message.stepStatus.push(reader.int32() as any);

            continue;
          }

          if (tag === 50) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.stepStatus.push(reader.int32() as any);
            }

            continue;
          }

          break;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BuildMask {
    return {
      fields: isSet(object.fields) ? FieldMask.unwrap(FieldMask.fromJSON(object.fields)) : undefined,
      inputProperties: globalThis.Array.isArray(object?.inputProperties)
        ? object.inputProperties.map((e: any) => StructMask.fromJSON(e))
        : [],
      outputProperties: globalThis.Array.isArray(object?.outputProperties)
        ? object.outputProperties.map((e: any) => StructMask.fromJSON(e))
        : [],
      requestedProperties: globalThis.Array.isArray(object?.requestedProperties)
        ? object.requestedProperties.map((e: any) => StructMask.fromJSON(e))
        : [],
      allFields: isSet(object.allFields) ? globalThis.Boolean(object.allFields) : false,
      stepStatus: globalThis.Array.isArray(object?.stepStatus)
        ? object.stepStatus.map((e: any) => statusFromJSON(e))
        : [],
    };
  },

  toJSON(message: BuildMask): unknown {
    const obj: any = {};
    if (message.fields !== undefined) {
      obj.fields = FieldMask.toJSON(FieldMask.wrap(message.fields));
    }
    if (message.inputProperties?.length) {
      obj.inputProperties = message.inputProperties.map((e) => StructMask.toJSON(e));
    }
    if (message.outputProperties?.length) {
      obj.outputProperties = message.outputProperties.map((e) => StructMask.toJSON(e));
    }
    if (message.requestedProperties?.length) {
      obj.requestedProperties = message.requestedProperties.map((e) => StructMask.toJSON(e));
    }
    if (message.allFields === true) {
      obj.allFields = message.allFields;
    }
    if (message.stepStatus?.length) {
      obj.stepStatus = message.stepStatus.map((e) => statusToJSON(e));
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BuildMask>, I>>(base?: I): BuildMask {
    return BuildMask.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BuildMask>, I>>(object: I): BuildMask {
    const message = createBaseBuildMask() as any;
    message.fields = object.fields ?? undefined;
    message.inputProperties = object.inputProperties?.map((e) => StructMask.fromPartial(e)) || [];
    message.outputProperties = object.outputProperties?.map((e) => StructMask.fromPartial(e)) || [];
    message.requestedProperties = object.requestedProperties?.map((e) => StructMask.fromPartial(e)) || [];
    message.allFields = object.allFields ?? false;
    message.stepStatus = object.stepStatus?.map((e) => e) || [];
    return message;
  },
};

function createBaseBuildPredicate(): BuildPredicate {
  return {
    builder: undefined,
    status: 0,
    gerritChanges: [],
    outputGitilesCommit: undefined,
    createdBy: "",
    tags: [],
    createTime: undefined,
    includeExperimental: false,
    build: undefined,
    canary: 0,
    experiments: [],
    descendantOf: "0",
    childOf: "0",
  };
}

export const BuildPredicate = {
  encode(message: BuildPredicate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.builder !== undefined) {
      BuilderID.encode(message.builder, writer.uint32(10).fork()).ldelim();
    }
    if (message.status !== 0) {
      writer.uint32(16).int32(message.status);
    }
    for (const v of message.gerritChanges) {
      GerritChange.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.outputGitilesCommit !== undefined) {
      GitilesCommit.encode(message.outputGitilesCommit, writer.uint32(34).fork()).ldelim();
    }
    if (message.createdBy !== "") {
      writer.uint32(42).string(message.createdBy);
    }
    for (const v of message.tags) {
      StringPair.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    if (message.createTime !== undefined) {
      TimeRange.encode(message.createTime, writer.uint32(58).fork()).ldelim();
    }
    if (message.includeExperimental === true) {
      writer.uint32(64).bool(message.includeExperimental);
    }
    if (message.build !== undefined) {
      BuildRange.encode(message.build, writer.uint32(74).fork()).ldelim();
    }
    if (message.canary !== 0) {
      writer.uint32(80).int32(message.canary);
    }
    for (const v of message.experiments) {
      writer.uint32(90).string(v!);
    }
    if (message.descendantOf !== "0") {
      writer.uint32(96).int64(message.descendantOf);
    }
    if (message.childOf !== "0") {
      writer.uint32(104).int64(message.childOf);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BuildPredicate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBuildPredicate() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.builder = BuilderID.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.gerritChanges.push(GerritChange.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.outputGitilesCommit = GitilesCommit.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.createdBy = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.tags.push(StringPair.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.createTime = TimeRange.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.includeExperimental = reader.bool();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.build = BuildRange.decode(reader, reader.uint32());
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.canary = reader.int32() as any;
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.experiments.push(reader.string());
          continue;
        case 12:
          if (tag !== 96) {
            break;
          }

          message.descendantOf = longToString(reader.int64() as Long);
          continue;
        case 13:
          if (tag !== 104) {
            break;
          }

          message.childOf = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BuildPredicate {
    return {
      builder: isSet(object.builder) ? BuilderID.fromJSON(object.builder) : undefined,
      status: isSet(object.status) ? statusFromJSON(object.status) : 0,
      gerritChanges: globalThis.Array.isArray(object?.gerritChanges)
        ? object.gerritChanges.map((e: any) => GerritChange.fromJSON(e))
        : [],
      outputGitilesCommit: isSet(object.outputGitilesCommit)
        ? GitilesCommit.fromJSON(object.outputGitilesCommit)
        : undefined,
      createdBy: isSet(object.createdBy) ? globalThis.String(object.createdBy) : "",
      tags: globalThis.Array.isArray(object?.tags) ? object.tags.map((e: any) => StringPair.fromJSON(e)) : [],
      createTime: isSet(object.createTime) ? TimeRange.fromJSON(object.createTime) : undefined,
      includeExperimental: isSet(object.includeExperimental) ? globalThis.Boolean(object.includeExperimental) : false,
      build: isSet(object.build) ? BuildRange.fromJSON(object.build) : undefined,
      canary: isSet(object.canary) ? trinaryFromJSON(object.canary) : 0,
      experiments: globalThis.Array.isArray(object?.experiments)
        ? object.experiments.map((e: any) => globalThis.String(e))
        : [],
      descendantOf: isSet(object.descendantOf) ? globalThis.String(object.descendantOf) : "0",
      childOf: isSet(object.childOf) ? globalThis.String(object.childOf) : "0",
    };
  },

  toJSON(message: BuildPredicate): unknown {
    const obj: any = {};
    if (message.builder !== undefined) {
      obj.builder = BuilderID.toJSON(message.builder);
    }
    if (message.status !== 0) {
      obj.status = statusToJSON(message.status);
    }
    if (message.gerritChanges?.length) {
      obj.gerritChanges = message.gerritChanges.map((e) => GerritChange.toJSON(e));
    }
    if (message.outputGitilesCommit !== undefined) {
      obj.outputGitilesCommit = GitilesCommit.toJSON(message.outputGitilesCommit);
    }
    if (message.createdBy !== "") {
      obj.createdBy = message.createdBy;
    }
    if (message.tags?.length) {
      obj.tags = message.tags.map((e) => StringPair.toJSON(e));
    }
    if (message.createTime !== undefined) {
      obj.createTime = TimeRange.toJSON(message.createTime);
    }
    if (message.includeExperimental === true) {
      obj.includeExperimental = message.includeExperimental;
    }
    if (message.build !== undefined) {
      obj.build = BuildRange.toJSON(message.build);
    }
    if (message.canary !== 0) {
      obj.canary = trinaryToJSON(message.canary);
    }
    if (message.experiments?.length) {
      obj.experiments = message.experiments;
    }
    if (message.descendantOf !== "0") {
      obj.descendantOf = message.descendantOf;
    }
    if (message.childOf !== "0") {
      obj.childOf = message.childOf;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BuildPredicate>, I>>(base?: I): BuildPredicate {
    return BuildPredicate.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BuildPredicate>, I>>(object: I): BuildPredicate {
    const message = createBaseBuildPredicate() as any;
    message.builder = (object.builder !== undefined && object.builder !== null)
      ? BuilderID.fromPartial(object.builder)
      : undefined;
    message.status = object.status ?? 0;
    message.gerritChanges = object.gerritChanges?.map((e) => GerritChange.fromPartial(e)) || [];
    message.outputGitilesCommit = (object.outputGitilesCommit !== undefined && object.outputGitilesCommit !== null)
      ? GitilesCommit.fromPartial(object.outputGitilesCommit)
      : undefined;
    message.createdBy = object.createdBy ?? "";
    message.tags = object.tags?.map((e) => StringPair.fromPartial(e)) || [];
    message.createTime = (object.createTime !== undefined && object.createTime !== null)
      ? TimeRange.fromPartial(object.createTime)
      : undefined;
    message.includeExperimental = object.includeExperimental ?? false;
    message.build = (object.build !== undefined && object.build !== null)
      ? BuildRange.fromPartial(object.build)
      : undefined;
    message.canary = object.canary ?? 0;
    message.experiments = object.experiments?.map((e) => e) || [];
    message.descendantOf = object.descendantOf ?? "0";
    message.childOf = object.childOf ?? "0";
    return message;
  },
};

function createBaseBuildRange(): BuildRange {
  return { startBuildId: "0", endBuildId: "0" };
}

export const BuildRange = {
  encode(message: BuildRange, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.startBuildId !== "0") {
      writer.uint32(8).int64(message.startBuildId);
    }
    if (message.endBuildId !== "0") {
      writer.uint32(16).int64(message.endBuildId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BuildRange {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBuildRange() as any;
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.startBuildId = longToString(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.endBuildId = longToString(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BuildRange {
    return {
      startBuildId: isSet(object.startBuildId) ? globalThis.String(object.startBuildId) : "0",
      endBuildId: isSet(object.endBuildId) ? globalThis.String(object.endBuildId) : "0",
    };
  },

  toJSON(message: BuildRange): unknown {
    const obj: any = {};
    if (message.startBuildId !== "0") {
      obj.startBuildId = message.startBuildId;
    }
    if (message.endBuildId !== "0") {
      obj.endBuildId = message.endBuildId;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<BuildRange>, I>>(base?: I): BuildRange {
    return BuildRange.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<BuildRange>, I>>(object: I): BuildRange {
    const message = createBaseBuildRange() as any;
    message.startBuildId = object.startBuildId ?? "0";
    message.endBuildId = object.endBuildId ?? "0";
    return message;
  },
};

/**
 * Manages builds.
 *
 * Buildbot: To simplify V1->V2 API transition for clients, buildbucket.v2.Builds
 * service has some support for non-LUCI builds. Most of such builds are
 * Buildbot builds and this documentation refers to them as such, but they also
 * include non-Buildbot non-LUCI builds (e.g. Skia builds).
 * See "Buildbot" paragraph in each RPC comment.
 */
export interface Builds {
  /**
   * Gets a build.
   *
   * By default the returned build does not include all fields.
   * See GetBuildRequest.mask.
   *
   * Buildbot: if the specified build is a buildbot build, converts it to Build
   * message with the following rules:
   * * bucket names are full, e.g. "luci.infra.try". Note that LUCI buckets
   *   in v2 are shortened, e.g. "try".
   * * if a v2 Build field does not make sense in V1, it is unset/empty.
   * * step support is not implemented for Buildbot builds.
   * Note that it does not support getting a buildbot build by build number.
   * Examples: go/buildbucket-rpc#getbuild
   *
   * GetBuild is good for getting detailed information for a build.
   * For use cases that only requires build status checking (e.g. wait for a
   * build to complete), please use GetBuildStatus instead.
   */
  GetBuild(request: GetBuildRequest): Promise<Build>;
  /**
   * Searches for builds.
   * Examples: go/buildbucket-rpc#searchbuilds
   */
  SearchBuilds(request: SearchBuildsRequest): Promise<SearchBuildsResponse>;
  /**
   * Updates a build.
   *
   * RPC metadata must include "x-buildbucket-token" key with a token
   * generated by the server when scheduling the build.
   */
  UpdateBuild(request: UpdateBuildRequest): Promise<Build>;
  /**
   * Schedules a new build.
   * The requester must have at least SCHEDULER role in the destination bucket.
   * Example: go/buildbucket-rpc#schedulebuild
   */
  ScheduleBuild(request: ScheduleBuildRequest): Promise<Build>;
  /**
   * Cancels a build.
   * The requester must have at least SCHEDULER role in the destination bucket.
   * Note that cancelling a build in ended state (meaning build is not in
   * STATUS_UNSPECIFIED, SCHEDULED or STARTED status) will be a no-op and
   * directly return up-to-date Build message.
   *
   * When called, Buildbucket will set the build's cancelTime to "now".  It
   * will also recursively start the cancellation process for any children of
   * this build which are marked as can_outlive_parent=false.
   *
   * The next time the build checks in (which happens periodically in
   * `bbagent`), bbagent will see the cancelTime, and start the cancellation
   * process described by the 'deadline' section in
   * https://chromium.googlesource.com/infra/luci/luci-py/+/HEAD/client/LUCI_CONTEXT.md.
   *
   * If the build ends before the build's grace_period, then the final status
   * reported from the build is accepted; this is considered 'graceful termination'.
   *
   * If the build doesn't end within the build's grace_period, Buildbucket will
   * forcibly cancel the build.
   */
  CancelBuild(request: CancelBuildRequest): Promise<Build>;
  /**
   * Executes multiple requests in a batch.
   * The response code is always OK.
   * Examples: go/buildbucket-rpc#batch
   */
  Batch(request: BatchRequest): Promise<BatchResponse>;
  /**
   * Creates a new build for the provided build proto.
   *
   * If build with the given ID already exists, returns ALREADY_EXISTS
   * error code.
   */
  CreateBuild(request: CreateBuildRequest): Promise<Build>;
  /**
   * Synthesizes a build proto.
   *
   * This RPC is exclusively for generating led builds.
   */
  SynthesizeBuild(request: SynthesizeBuildRequest): Promise<Build>;
  /**
   * Gets a build's status.
   *
   * The returned build contains the requested build id or
   * (builder + build number), and build status.
   *
   * It's useful when a user only wants to check the build's status (i.e. wait
   * for a build to complete).
   */
  GetBuildStatus(request: GetBuildStatusRequest): Promise<Build>;
  /**
   * Starts a build.
   *
   * RPC metadata must include "x-buildbucket-token" key with
   * * a BUILD type token generated by the server when it creates a Swarming
   *   task for the build (builds on Swarming) or
   * * a START_BUILD type token generated by the server when it attempts to run
   *   a backend task (builds on TaskBackend).
   *
   * Agent must call it before making any UpdateBuild calls.
   *
   * StartBuild will associate a task with a build if the association is not done
   * after RunTaskResponse is returned to buildbucket.
   */
  StartBuild(request: StartBuildRequest): Promise<StartBuildResponse>;
}

export const BuildsServiceName = "buildbucket.v2.Builds";
export class BuildsClientImpl implements Builds {
  static readonly DEFAULT_SERVICE = BuildsServiceName;
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || BuildsServiceName;
    this.rpc = rpc;
    this.GetBuild = this.GetBuild.bind(this);
    this.SearchBuilds = this.SearchBuilds.bind(this);
    this.UpdateBuild = this.UpdateBuild.bind(this);
    this.ScheduleBuild = this.ScheduleBuild.bind(this);
    this.CancelBuild = this.CancelBuild.bind(this);
    this.Batch = this.Batch.bind(this);
    this.CreateBuild = this.CreateBuild.bind(this);
    this.SynthesizeBuild = this.SynthesizeBuild.bind(this);
    this.GetBuildStatus = this.GetBuildStatus.bind(this);
    this.StartBuild = this.StartBuild.bind(this);
  }
  GetBuild(request: GetBuildRequest): Promise<Build> {
    const data = GetBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  SearchBuilds(request: SearchBuildsRequest): Promise<SearchBuildsResponse> {
    const data = SearchBuildsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "SearchBuilds", data);
    return promise.then((data) => SearchBuildsResponse.decode(_m0.Reader.create(data)));
  }

  UpdateBuild(request: UpdateBuildRequest): Promise<Build> {
    const data = UpdateBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  ScheduleBuild(request: ScheduleBuildRequest): Promise<Build> {
    const data = ScheduleBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ScheduleBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  CancelBuild(request: CancelBuildRequest): Promise<Build> {
    const data = CancelBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CancelBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  Batch(request: BatchRequest): Promise<BatchResponse> {
    const data = BatchRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Batch", data);
    return promise.then((data) => BatchResponse.decode(_m0.Reader.create(data)));
  }

  CreateBuild(request: CreateBuildRequest): Promise<Build> {
    const data = CreateBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  SynthesizeBuild(request: SynthesizeBuildRequest): Promise<Build> {
    const data = SynthesizeBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "SynthesizeBuild", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  GetBuildStatus(request: GetBuildStatusRequest): Promise<Build> {
    const data = GetBuildStatusRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetBuildStatus", data);
    return promise.then((data) => Build.decode(_m0.Reader.create(data)));
  }

  StartBuild(request: StartBuildRequest): Promise<StartBuildResponse> {
    const data = StartBuildRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "StartBuild", data);
    return promise.then((data) => StartBuildResponse.decode(_m0.Reader.create(data)));
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
