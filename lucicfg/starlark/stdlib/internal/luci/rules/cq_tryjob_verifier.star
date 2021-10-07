# Copyright 2019 The LUCI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Defines luci.cq_tryjob_verifier(...) rule."""

load("@stdlib//internal/graph.star", "graph")
load("@stdlib//internal/lucicfg.star", "lucicfg")
load("@stdlib//internal/re.star", "re")
load("@stdlib//internal/validate.star", "validate")
load("@stdlib//internal/luci/common.star", "keys", "kinds")
load("@stdlib//internal/luci/lib/cq.star", "cq")

def _cq_tryjob_verifier(
        ctx,
        builder = None,
        *,
        cq_group = None,
        includable_only = None,
        result_visibility = None,
        disable_reuse = None,
        cancel_stale = None,
        experiment_percentage = None,
        location_regexp = None,
        location_regexp_exclude = None,
        owner_whitelist = None,
        equivalent_builder = None,
        equivalent_builder_percentage = None,
        equivalent_builder_whitelist = None,
        mode_allowlist = None):
    """A verifier in a luci.cq_group(...) that triggers tryjobs to verify CLs.

    When processing a CL, the CQ examines a list of registered verifiers and
    launches new corresponding builds (called "tryjobs") if it decides this is
    necessary (per the configuration of the verifier and the previous history
    of this CL).

    The CQ automatically retries failed tryjobs (per configured `retry_config`
    in luci.cq_group(...)) and only allows CL to land if each builder has
    succeeded in the latest retry. If a given tryjob result is too old (>1 day)
    it is ignored.

    #### Filtering based on files touched by a CL

    The CQ can examine a set of files touched by the CL and decide to skip this
    verifier. Touching a file means either adding, modifying or removing it.

    This is controlled by `location_regexp` and `location_regexp_exclude`
    fields:

      * If `location_regexp` is specified and no file in a CL matches any of the
        `location_regexp`, then the CQ will not care about this verifier.
      * If a file in a CL matches any `location_regexp_exclude`, then this file
        won't be considered when matching `location_regexp`.
      * If `location_regexp_exclude` is specified, but `location_regexp` is not,
        `location_regexp` is implied to be `.*`.
      * If neither `location_regexp` nor `location_regexp_exclude` are specified
        (default), the verifier will be used on all CLs.

    The matches are done against the following string:

        <gerrit_url>/<gerrit_project_name>/+/<cl_file_path>

    The file path is relative to the repo root, and it uses Unix `/` directory
    separator.

    The comparison is a full match. The pattern is implicitly anchored with `^`
    and `$`, so there is no need add them.

    The pattern must use [Google Re2](https://github.com/google/re2) library
    syntax, [documented here](https://github.com/google/re2/wiki/Syntax).

    This filtering currently cannot be used in any of the following cases:

      * For verifiers in CQ groups with `allow_submit_with_open_deps = True`.

    Please talk to CQ owners if these restrictions are limiting you.

    ##### Examples

    Enable the verifier for all CLs touching any file in `third_party/WebKit`
    directory of the `chromium/src` repo, but not directory itself:

        luci.cq_tryjob_verifier(
            location_regexp = [
                'https://chromium-review.googlesource.com/chromium/src/[+]/third_party/WebKit/.+',
            ],
        )

    Match a CL which touches at least one file other than `one.txt` inside
    `all/` directory of the Gerrit project `repo`:

        luci.cq_tryjob_verifier(
            location_regexp = ['https://example.com/repo/[+]/.+'],
            location_regexp_exclude = ['https://example.com/repo/[+]/all/one.txt'],
        )

    Match a CL which touches at least one file other than `one.txt` in any
    repository **or** belongs to any other Gerrit server. Note, in this case
    `location_regexp` defaults to `.*`:

        luci.cq_tryjob_verifier(
            location_regexp_exclude = ['https://example.com/repo/[+]/all/one.txt'],
        )

    #### Per-CL opt-in only builders

    For builders which may be useful only for some CLs, predeclare them using
    `includable_only=True` flag. Such builders will be triggered by CQ if and
    only if a CL opts in via `CQ-Include-Trybots: <builder>` in its description.

    For example, default verifiers may include only fast builders which skip low
    level assertions, but for coverage of such assertions one may add slower
    "debug" level builders into which CL authors opt-in as needed:

          # triggered & required for all CLs.
          luci.cq_tryjob_verifier(builder="win")
          # triggered & required if only if CL opts in via
          # `CQ-Include-Trybots: project/try/win-debug`.
          luci.cq_tryjob_verifier(builder="win-debug", includable_only=True)

    #### Declaring verifiers

    `cq_tryjob_verifier` is used inline in luci.cq_group(...) declarations to
    provide per-builder verifier parameters. `cq_group` argument can be omitted
    in this case:

        luci.cq_group(
            name = 'Main CQ',
            ...
            verifiers = [
                luci.cq_tryjob_verifier(
                    builder = 'Presubmit',
                    disable_reuse = True,
                ),
                ...
            ],
        )


    It can also be associated with a luci.cq_group(...) outside of
    luci.cq_group(...) declaration. This is in particular useful in functions.
    For example:

        luci.cq_group(name = 'Main CQ')

        def try_builder(name, ...):
            luci.builder(name = name, ...)
            luci.cq_tryjob_verifier(builder = name, cq_group = 'Main CQ')


    #### Declaring a Tricium analyzer

    `cq_tryjob_verifier` can be used to declare a [Tricium] analyzer by
    providing the builder and `mode_allowlist=[cq.MODE_ANALYZER_RUN]`. It will
    generate Tricium config as well as CQ config which will work seamlessly
    after Tricium is merged into CV.

    However, the following restrictions apply until CV takes on Tricium:

    * Most CQ features are not supported except for `location_regexp` and
    `owner_whitelist`. If provided, they must meet the following conditions:
        * `location_regexp` must either start with `.+\\.` or
        `https://{HOST}-review.googlesource.com/{PROJECT}/[+]/.+\\.`.
        They can optionally be followed by a file extension name which
        instructs Tricium to run this analyzer only on certain type of files.
            * If the gerrit url one is used, the generated Tricium config will
            watch the repos specified in location_regexp instead of the one
            watched by the containing cq_group. Note that, the exact same set
            of Gerrit repos should be sepcified across all analyzers in this
            cq_group and across each unique file extension.
        * `owner_whitelist` must be the same for all analyzers declared
        in this cq_group.
    * Analyzer will run on changes targeting **all refs** of the Gerrit repos
    watched by the containing cq_group (or repos derived from location_regexp,
    see above) even though refs or refs_exclude may be provided.
    * All analyzers must be declared in a single luci.cq_group(...).

    For example:

        luci.project(tricium="tricium-prod.appspot.com")

        luci.cq_group(
            name = 'Main CQ',
            ...
            verifiers = [
                luci.cq_tryjob_verifier(
                    builder = "spell-checker",
                    owner_whitelist = ["project-committer"],
                    mode_allowlist = [cq.MODE_ANALYZER_RUN],
                ),
                luci.cq_tryjob_verifier(
                    builder = "go-linter",
                    location_regexp = [".+\\.go"]
                    owner_whitelist = ["project-committer"],
                    mode_allowlist = [cq.MODE_ANALYZER_RUN],
                ),
                luci.cq_tryjob_verifier(builder = "Presubmit"),
                ...
            ],
        )

    Note for migrating to lucicfg for LUCI Projects whose sole purpose is
    to host a single Tricium config today
    ([Example](https://fuchsia.googlesource.com/infra/config/+/HEAD/repositories/infra/recipes/tricium-prod.cfg)):

    Due to the restrictions mentioned above, it is not possible to merge those
    auxillary Projects back to the main LUCI Project. It will be unblocked
    after Tricium is folded into CV. To migrate, users can declare new
    luci.cq_group(...)s in those Projects to host Tricium analyzers.
    However, CQ config should not be generated because the config groups will
    overlap with the config group in the main LUCI Project (i.e. watch same
    refs) and break CQ. This can be done by asking lucicfg to track only
    Tricium config: `lucicfg.config(tracked_files=["tricium-prod.cfg"])`.

    [Tricium]: https://chromium.googlesource.com/infra/infra/+/HEAD/go/src/infra/tricium

    Args:
      ctx: the implicit rule context, see lucicfg.rule(...).
      builder: a builder to launch when verifying a CL, see luci.builder(...).
        Can also be a reference to a builder defined in another project. See
        [Referring to builders in other projects](#external_builders) for more
        details. Required.
      cq_group: a CQ group to add the verifier to. Can be omitted if
        `cq_tryjob_verifier` is used inline inside some luci.cq_group(...)
        declaration.
      result_visibility: can be used to restrict the visibility of the tryjob
        results in comments on Gerrit. Valid values are `cq.COMMENT_LEVEL_FULL`
        and `cq.COMMENT_LEVEL_RESTRICTED` constants. Default is to give full
        visibility: builder name and full summary markdown are included in the
        Gerrit comment.
      cancel_stale: Controls whether not yet finished builds previously
        triggered by CQ will be cancelled as soon as a substantially different
        patchset is uploaded to a CL. Default is True, meaning CQ will cancel.
        In LUCI Change Verifier (aka CV, successor of CQ), changing this
        option will only take effect on newly-created Runs once config
        propagates to CV. Ongoing Runs will retain the old behavior.
        (TODO(crbug/1127991): refactor this doc after migration. As of 09/2020,
        CV implementation is WIP)
      includable_only: if True, this builder will only be triggered by CQ if it
        is also specified via `CQ-Include-Trybots:` on CL description. Default
        is False. See the explanation above for all details. For builders with
        `experiment_percentage` or `location_regexp` or
        `location_regexp_exclude`, don't specify `includable_only`. Such
        builders can already be forcefully added via `CQ-Include-Trybots:` in
        the CL description.
      disable_reuse: if True, a fresh build will be required for each CQ
        attempt. Default is False, meaning the CQ may re-use a successful build
        triggered before the current CQ attempt started. This option is
        typically used for verifiers which run presubmit scripts, which are
        supposed to be quick to run and provide additional OWNERS, lint, etc.
        checks which are useful to run against the latest revision of the CL's
        target branch.
      experiment_percentage: when this field is present, it marks the verifier
        as experimental. Such verifier is only triggered on a given percentage
        of the CLs and the outcome does not affect the decision whether a CL can
        land or not. This is typically used to test new builders and estimate
        their capacity requirements. May be combined with `location_regexp` and
        `location_regexp_exclude`.
      location_regexp: a list of regexps that define a set of files whose
        modification trigger this verifier. See the explanation above for all
        details.
      location_regexp_exclude: a list of regexps that define a set of files to
        completely skip when evaluating whether the verifier should be applied
        to a CL or not. See the explanation above for all details.
      owner_whitelist: a list of groups with accounts of CL owners to enable
        this builder for. If set, only CLs owned by someone from any one of
        these groups will be verified by this builder.
      equivalent_builder: an optional alternative builder for the CQ to choose
        instead. If provided, the CQ will choose only one of the equivalent
        builders as required based purely on the given CL and CL's owner and
        **regardless** of the possibly already completed try jobs.
      equivalent_builder_percentage: a percentage expressing probability of the
        CQ triggering `equivalent_builder` instead of `builder`. A choice itself
        is made deterministically based on CL alone, hereby all CQ attempts on
        all patchsets of a given CL will trigger the same builder, assuming CQ
        config doesn't change in the mean time. Note that if
        `equivalent_builder_whitelist` is also specified, the choice over which
        of the two builders to trigger will be made only for CLs owned by the
        accounts in the whitelisted group. Defaults to 0, meaning the equivalent
        builder is never triggered by the CQ, but an existing build can be
        re-used.
      equivalent_builder_whitelist: a group name with accounts to enable the
        equivalent builder substitution for. If set, only CLs that are owned
        by someone from this group have a chance to be verified by the
        equivalent builder. All other CLs are verified via the main builder.
      mode_allowlist: a list of modes that CQ will trigger this verifier for.
        CQ supports `cq.MODE_DRY_RUN` and `cq.MODE_FULL_RUN` out of the box.
        Additional Run modes can be defined via
        `luci.cq_group(additional_modes=...)`.
    """
    builder = keys.builder_ref(builder, attr = "builder", allow_external = True)

    location_regexp = validate.list("location_regexp", location_regexp)
    for r in location_regexp:
        validate.string("location_regexp", r)
    location_regexp_exclude = validate.list("location_regexp_exclude", location_regexp_exclude)
    for r in location_regexp_exclude:
        validate.string("location_regexp_exclude", r)

    # Note: CQ does this itself implicitly, but we want configs to be explicit.
    if location_regexp_exclude and not location_regexp:
        location_regexp = [".*"]

    owner_whitelist = validate.list("owner_whitelist", owner_whitelist)
    for o in owner_whitelist:
        validate.string("owner_whitelist", o)

    # 'equivalent_builder' has same format as 'builder', except it is optional.
    if equivalent_builder:
        equivalent_builder = keys.builder_ref(
            equivalent_builder,
            attr = "equivalent_builder",
            allow_external = True,
        )

    equivalent_builder_percentage = validate.float(
        "equivalent_builder_percentage",
        equivalent_builder_percentage,
        min = 0.0,
        max = 100.0,
        required = False,
    )
    equivalent_builder_whitelist = validate.string(
        "equivalent_builder_whitelist",
        equivalent_builder_whitelist,
        required = False,
    )

    mode_allowlist = validate.list("mode_allowlist", mode_allowlist)
    for m in mode_allowlist:
        validate.string("mode_allowlist", m)

    if cq.MODE_ANALYZER_RUN in mode_allowlist:
        # TODO(crbug/1202952): Remove all following restrictions after tricium
        # is folded into CV.
        if len(mode_allowlist) > 1:
            fail('"ANALYZER_RUN" must be the only mode in "mode_allowlist"')
        if location_regexp:
            re_for_gerrit_url_re = r"https://([a-z\-]+)\-review\.googlesource\.com/([a-z0-9_\-/]+)+/\[\+\]/"
            re_for_extension_re = r"\\\.[a-z]+"
            re_for_location_re = r"^(%s)?\.\+(%s)?$" % (re_for_gerrit_url_re, re_for_extension_re)
            ext_to_gerrit_urls = {}
            all_gerrit_urls = []
            for r in location_regexp:
                groups = re.submatches(re_for_location_re, r)
                if not groups:
                    fail(r'"location_regexp" of an analyzer MUST either be in ' +
                         r'the format of ".+\.extension" (e.g. ".+\.py) ' +
                         r'or "https://host-review.googlesource.com/project/[+]/.+\.extension" ' +
                         r'(e.g. "https://chromium-review.googlesource.com/infra/infra/[+]/.+\.py"). Extension is optional.')
                gerrit_url, ext = groups[1].rstrip("[+]"), groups[4].lstrip(r"\.")
                if ext not in ext_to_gerrit_urls:
                    ext_to_gerrit_urls[ext] = []
                if ((gerrit_url and "" in all_gerrit_urls) or (
                    gerrit_url == "" and any(all_gerrit_urls)
                )):
                    fail(r'"location_regexp" of an analyzer MUST NOT mix ' +
                         r'two different formats (i.e. ".+\.extension" ' +
                         r'and "https://host-review.googlesource.com/project/[+]/.+\.extension"). Got %s.' % location_regexp)
                ext_to_gerrit_urls[ext].append(gerrit_url)
                all_gerrit_urls.append(gerrit_url)

            # Since all analyzers in Tricium config are watching the same set of
            # Gerrit repos, we need to make sure that for each extension this
            # analyzer is watching, it MUST specify the same set of Gerrit
            # repos it is watching. It allows lucicfg to derive a homogeneous
            # set of watching Gerrit repos when generating Tricium config later.
            #
            # For example, location_exgep like
            #   https://example-review.googlesource.com/repo1/[+]/.+\.go
            #   https://example-review.googlesource.com/repo2/[+]/.+\.go
            #   https://example-review.googlesource.com/repo1/[+]/.+\.py
            # is not allowed, because the generated Tricium config has to
            # watch both repo1 and repo2. If we allow it, Tricium will
            # implicitly run for python files in repo2 which is not what
            # user intended. Therefore, user MUST provide
            # https://example-review.googlesource.com/repo2/[+]/.+\.py here
            # as well.
            ref_ext, ref_gerrit_urls = ext_to_gerrit_urls.popitem()
            ref_gerrit_urls = sorted(ref_gerrit_urls)
            for ext, gerrit_urls in ext_to_gerrit_urls.items():
                if sorted(gerrit_urls) != ref_gerrit_urls:
                    fail('each extension specified in "location_regexp" of an ' +
                         "analyzer MUST have the same set of gerrit URLs; got " +
                         "%s for extension %s, but %s for extension %s." % (
                             sorted(gerrit_urls),
                             ext,
                             ref_gerrit_urls,
                             ref_ext,
                         ))
        if location_regexp_exclude:
            fail('"analyzer currently can not be used together with  "location_regexp_exclude"')

    if not equivalent_builder:
        if equivalent_builder_percentage != None:
            fail('"equivalent_builder_percentage" can be used only together with "equivalent_builder"')
        if equivalent_builder_whitelist != None:
            fail('"equivalent_builder_whitelist" can be used only together with "equivalent_builder"')

    if includable_only:
        if location_regexp:
            fail('"includable_only" can not be used together with "location_regexp"')
        if location_regexp_exclude:
            fail('"includable_only" can not be used together with "location_regexp_exclude"')
        if experiment_percentage:
            fail('"includable_only" can not be used together with "experiment_percentage"')
        if mode_allowlist:
            fail('"includable_only" can not be used together with "mode_allowlist"')

    # Note: name of this node is important only for error messages. It isn't
    # showing up in any generated files and by construction it can't accidentally
    # collide with some other name.
    key = keys.unique(kinds.CQ_TRYJOB_VERIFIER, builder.id)
    graph.add_node(key, props = {
        "disable_reuse": validate.bool("disable_reuse", disable_reuse, required = False),
        "result_visibility": validate.int(
            "result_visibility",
            result_visibility,
            default = cq.COMMENT_LEVEL_UNSET,
            required = False,
        ),
        "cancel_stale": validate.bool("cancel_stale", cancel_stale, required = False),
        "includable_only": validate.bool("includable_only", includable_only, required = False),
        "experiment_percentage": validate.float(
            "experiment_percentage",
            experiment_percentage,
            min = 0.0,
            max = 100.0,
            required = False,
        ),
        "location_regexp": location_regexp,
        "location_regexp_exclude": location_regexp_exclude,
        "owner_whitelist": owner_whitelist,
        "mode_allowlist": mode_allowlist,
    })
    if cq_group:
        graph.add_edge(parent = keys.cq_group(cq_group), child = key)
    graph.add_edge(parent = key, child = builder)

    # Need to setup a node to represent 'equivalent_builder' so that lucicfg can
    # verify (via the graph integrity check) that such builder was actually
    # defined somewhere. Note that we can't add 'equivalent_builder' as another
    # child of 'cq_tryjob_verifier' node, since then it would be ambiguous which
    # child builder_ref node is the "main one" and which is the equivalent.
    if equivalent_builder:
        # Note: this key is totally invisible.
        eq_key = keys.unique(
            kind = kinds.CQ_EQUIVALENT_BUILDER,
            name = equivalent_builder.id,
        )
        graph.add_node(eq_key, props = {
            "percentage": equivalent_builder_percentage,
            "whitelist": equivalent_builder_whitelist,
        })
        graph.add_edge(parent = key, child = eq_key)
        graph.add_edge(parent = eq_key, child = equivalent_builder)

    # This is used to detect cq_tryjob_verifier nodes that aren't connected to
    # any cq_group. Such orphan nodes aren't allowed.
    graph.add_node(keys.cq_verifiers_root(), idempotent = True)
    graph.add_edge(parent = keys.cq_verifiers_root(), child = key)

    return graph.keyset(key)

cq_tryjob_verifier = lucicfg.rule(impl = _cq_tryjob_verifier)
