lucicfg.config(config_dir = '.output')
lucicfg.config(tracked_files = ['*.cfg'])
lucicfg.config(fail_on_warnings = True)

luci.project(
    name = 'infra',

    buildbucket = 'cr-buildbucket.appspot.com',
    logdog = 'luci-logdog.appspot.com',
    milo = 'luci-milo.appspot.com',
    notify = 'luci-notify.appspot.com',
    scheduler = 'luci-scheduler.appspot.com',
    swarming = 'chromium-swarm.appspot.com',

    acls = [
        acl.entry(
            roles = [
                acl.PROJECT_CONFIGS_READER,
                acl.LOGDOG_READER,
                acl.BUILDBUCKET_READER,
                acl.SCHEDULER_READER,
            ],
            groups = ['all'],
        ),
        acl.entry(
            roles = [
                acl.BUILDBUCKET_OWNER,
                acl.SCHEDULER_OWNER,
                acl.CQ_COMMITTER,
            ],
            groups = ['admins'],
        ),
    ],
)

luci.logdog(gs_bucket = 'chromium-luci-logdog')

luci.milo(
    logo = 'https://storage.googleapis.com/chrome-infra-public/logo/chrome-infra-logo-200x200.png',
    favicon = 'https://storage.googleapis.com/chrome-infra-public/logo/favicon.ico',
    monorail_project = 'tutu, all aboard',
    monorail_components = ['Stuff>Hard'],
    bug_summary = 'Bug summary',
    bug_description = 'Everything is broken',
)


# Recipes.

luci.recipe(
    name = 'main/recipe',
    cipd_package = 'recipe/bundles/main',
)

# Executables.

luci.executable(
    name = 'main/executable',
    cipd_package = 'executable/bundles/main',
)


# CI bucket.

luci.bucket(
    name = 'ci',

    # Allow developers to force-launch CI builds through Scheduler, but not
    # directly through Buildbucket. The direct access to Buildbucket allows to
    # override almost all aspects of the builds (e.g. what recipe is used),
    # and Buildbucket totally ignores any concurrency limitations set in the
    # LUCI Scheduler configs. This makes direct Buildbucket access to CI buckets
    # dangerous. They usually have very small pool of machines, and these
    # machines are assumed to be running only "approved" code (being post-submit
    # builders).
    acls = [
        acl.entry(
            acl.SCHEDULER_TRIGGERER,
            groups = ['devs'],
            projects = ['some-project'],
        ),
    ],
)

luci.gitiles_poller(
    name = 'master-poller',
    bucket = 'ci',
    repo = 'https://noop.com',
    refs = [
        'refs/heads/master',
        'refs/tags/blah',
        'refs/branch-heads/\d+\.\d+',
    ],
    path_regexps = ['.*'],
    path_regexps_exclude = ['excluded'],
    schedule = 'with 10s interval',
)

luci.builder(
    name = 'linux ci builder',
    bucket = 'ci',
    executable = luci.recipe(
        name = 'main/recipe',
        cipd_package = 'recipe/bundles/main',
    ),

    triggered_by = ['master-poller'],
    triggers = [
        'ci/generically named builder',
        'ci/generically named executable builder',
    ],

    properties = {
        'prop1': 'val1',
        'prop2': ['val2', 123],
    },
    service_account = 'builder@example.com',

    caches = [
        swarming.cache('path1'),
        swarming.cache('path2', name='name2'),
        swarming.cache('path3', name='name3', wait_for_warm_cache=10*time.minute),
    ],
    execution_timeout = 3 * time.hour,

    dimensions = {
        'os': 'Linux',
        'builder': 'linux ci builder',  # no auto_builder_dimension
        'prefer_if_available': [
            swarming.dimension('first-choice', expiration=5*time.minute),
            swarming.dimension('fallback'),
        ],
    },
    priority = 80,
    swarming_tags = ['tag1:val1', 'tag2:val2'],
    expiration_timeout = time.hour,
    build_numbers = True,

    triggering_policy = scheduler.greedy_batching(
        max_concurrent_invocations=5,
        max_batch_size=10,
    )
)

luci.builder(
    name = 'generically named builder',
    bucket = 'ci',
    executable = 'main/recipe',

    triggered_by = ['master-poller'],
)

luci.builder(
    name = 'generically named executable builder',
    bucket = 'ci',
    executable = 'main/executable',
    properties = {
        'prop1': 'val1',
        'prop2': ['val2', 123],
    },

    triggered_by = ['master-poller'],
)

luci.builder(
    name = 'cron builder',
    bucket = 'ci',
    executable = 'main/recipe',
    schedule = '0 6 * * *',
    repo = 'https://cron.repo.example.com',
)

luci.builder(
    name = 'builder with custom swarming host',
    bucket = 'ci',
    executable = 'main/recipe',
    swarming_host = 'another-swarming.appspot.com',
)


# Try bucket.

luci.bucket(
    name = 'try',

    # Allow developers to launch try jobs directly with whatever parameters
    # they want. Try bucket is basically a free build farm for all developers.
    acls = [
        acl.entry(acl.BUILDBUCKET_TRIGGERER, groups='devs'),
    ],
)

luci.builder(
    name = 'linux try builder',
    bucket = 'try',
    executable = 'main/recipe',
)

luci.builder(
    name = 'generically named builder',
    bucket = 'try',
    executable = 'main/recipe',
)

luci.builder(
    name = 'builder with executable',
    bucket = 'try',
    executable = 'main/executable',
)


# Inline definitions.


def inline_poller():
  return luci.gitiles_poller(
      name = 'inline poller',
      bucket = 'inline',
      repo = 'https://noop.com',
      refs = [
          'refs/heads/master',
          'refs/tags/blah',
          'refs/branch-heads/\d+\.\d+',
      ],
      schedule = 'with 10s interval',
  )


luci.builder(
    name = 'triggerer builder',
    bucket = luci.bucket(name = 'inline'),
    executable = luci.recipe(
        name = 'inline/recipe',
        cipd_package = 'recipe/bundles/inline',
    ),

    service_account = 'builder@example.com',

    triggers = [
        luci.builder(
            name = 'triggered builder',
            bucket = 'inline',
            executable = 'inline/recipe',
        ),
    ],

    triggered_by = [inline_poller()],
)


luci.builder(
    name = 'another builder',
    bucket = 'inline',
    executable = luci.recipe(
        name = 'inline/recipe',
        cipd_package = 'recipe/bundles/inline',
    ),
    service_account = 'builder@example.com',
    triggered_by = [inline_poller()],
)


luci.builder(
    name = 'another executable builder',
    bucket = 'inline',
    executable = luci.executable(
        name = 'inline/executable',
        cipd_package = 'executable/bundles/inline',
    ),
    service_account = 'builder@example.com',
    triggered_by = [inline_poller()],
)


# List views.


luci.list_view(
    name = 'List view',
    entries = [
        'cron builder',
        'ci/generically named builder',
        luci.list_view_entry(
            builder = 'linux ci builder',
        ),
    ],
)

luci.list_view_entry(
    list_view = 'List view',
    builder = 'inline/triggered builder',
)

# Console views.


luci.console_view(
    name = 'Console view',
    title = 'CI Builders',
    header = {
        'links': [
            {'name': 'a', 'links': [{'text': 'a'}]},
            {'name': 'b', 'links': [{'text': 'b'}]},
        ],
    },
    repo = 'https://noop.com',
    refs = ['refs/tags/blah', 'refs/branch-heads/\d+\.\d+'],
    exclude_ref = 'refs/heads/master',
    include_experimental_builds = True,
    entries = [
        luci.console_view_entry(
            builder = 'linux ci builder',
            category = 'a|b',
            short_name = 'lnx',
        ),
        # An alias for luci.console_view_entry(**{...}).
        {'builder': 'cron builder', 'category': 'cron'},
    ],
    default_commit_limit = 3,
    default_expand = True,
)

luci.console_view_entry(
    console_view = 'Console view',
    builder = 'inline/triggered builder',
)


# Notifier.


luci.notifier(
    name = 'main notifier',
    on_new_status = ['FAILURE'],
    notify_emails = ['someone@example,com'],
    notify_blamelist = True,
    template = 'notifier-template',
    notified_by = [
        'linux ci builder',
        'cron builder',
    ],
)

luci.notifier_template(
    name = 'notifier-template',
    body = 'Hello\n\nHi\n',
)

luci.notifier_template(
    name = 'another-template',
    body = 'Boo!\n',
)


luci.builder(
    name = 'watched builder',
    bucket = 'ci',
    executable = 'main/recipe',
    repo = 'https://custom.example.com/repo',
    notifies = ['main notifier'],
)


# CQ.


luci.cq(
    submit_max_burst = 10,
    submit_burst_delay = 10 * time.minute,
    draining_start_time = '2017-12-23T15:47:58Z',
    status_host = 'chromium-cq-status.appspot.com',
    project_scoped_account = True,
)

luci.cq_group(
    name = 'main-cq',
    watch = [
        cq.refset('https://example.googlesource.com/repo'),
        cq.refset('https://example.googlesource.com/another/repo'),
    ],
    acls = [
        acl.entry(acl.CQ_COMMITTER, groups = ['committers']),
        acl.entry(acl.CQ_DRY_RUNNER, groups = ['dry-runners']),
    ],
    allow_submit_with_open_deps = True,
    allow_owner_if_submittable = cq.ACTION_COMMIT,
    tree_status_host = 'tree-status.example.com',

    cancel_stale_tryjobs = True,
    verifiers = [
        luci.cq_tryjob_verifier(
            builder = 'linux try builder',
            location_regexp_exclude = ['https://example.com/repo/[+]/all/one.txt'],
        ),
        # An alias for luci.cq_tryjob_verifier(**{...}).
        {'builder': 'try/generically named builder', 'disable_reuse': True},
        # An alias for luci.cq_tryjob_verifier(<builder>).
        'another-project:try/yyy',
        luci.cq_tryjob_verifier(
            builder = 'another-project:try/zzz',
            owner_whitelist = ['another-project-committers'],
        ),
    ],
)

luci.cq_tryjob_verifier(
    builder = 'triggerer builder',
    cq_group = 'main-cq',
    experiment_percentage = 50.0,
)

luci.cq_tryjob_verifier(
    builder = 'triggered builder',
    cq_group = 'main-cq',
)

luci.cq_tryjob_verifier(
    builder = luci.builder(
        name = 'main cq builder',
        bucket = 'try',
        executable = 'main/recipe',
    ),
    equivalent_builder = luci.builder(
        name = 'equivalent cq builder',
        bucket = 'try',
        executable = 'main/recipe',
    ),
    equivalent_builder_percentage = 60,
    equivalent_builder_whitelist = 'owners',
    cq_group = 'main-cq',
)


# Emitting arbitrary configs,

lucicfg.emit(
    dest = 'dir/custom.cfg',
    data = 'hello!\n',
)


# Expect configs:
#
# === commit-queue.cfg
# draining_start_time: "2017-12-23T15:47:58Z"
# cq_status_host: "chromium-cq-status.appspot.com"
# submit_options: <
#   max_burst: 10
#   burst_delay: <
#     seconds: 600
#   >
# >
# config_groups: <
#   gerrit: <
#     url: "https://example-review.googlesource.com"
#     projects: <
#       name: "repo"
#       ref_regexp: "refs/heads/master"
#     >
#     projects: <
#       name: "another/repo"
#       ref_regexp: "refs/heads/master"
#     >
#   >
#   verifiers: <
#     gerrit_cq_ability: <
#       committer_list: "admins"
#       committer_list: "committers"
#       dry_run_access_list: "dry-runners"
#       allow_submit_with_open_deps: true
#       allow_owner_if_submittable: COMMIT
#     >
#     tree_status: <
#       url: "https://tree-status.example.com"
#     >
#     tryjob: <
#       builders: <
#         name: "another-project/try/yyy"
#       >
#       builders: <
#         name: "another-project/try/zzz"
#         owner_whitelist_group: "another-project-committers"
#       >
#       builders: <
#         name: "infra/inline/triggered builder"
#         triggered_by: "infra/inline/triggerer builder"
#       >
#       builders: <
#         name: "infra/inline/triggerer builder"
#         experiment_percentage: 50
#       >
#       builders: <
#         name: "infra/try/generically named builder"
#         disable_reuse: true
#       >
#       builders: <
#         name: "infra/try/linux try builder"
#         location_regexp: ".*"
#         location_regexp_exclude: "https://example.com/repo/[+]/all/one.txt"
#       >
#       builders: <
#         name: "infra/try/main cq builder"
#         equivalent_to: <
#           name: "infra/try/equivalent cq builder"
#           percentage: 60
#           owner_whitelist_group: "owners"
#         >
#       >
#       retry_config: <
#         single_quota: 1
#         global_quota: 2
#         failure_weight: 100
#         transient_failure_weight: 1
#         timeout_weight: 100
#       >
#       cancel_stale_tryjobs: YES
#     >
#   >
# >
# project_scoped_account: YES
# ===
#
# === cr-buildbucket.cfg
# buckets: <
#   name: "ci"
#   acls: <
#     role: WRITER
#     group: "admins"
#   >
#   acls: <
#     group: "all"
#   >
#   swarming: <
#     builders: <
#       name: "builder with custom swarming host"
#       swarming_host: "another-swarming.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "cron builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "generically named builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "generically named executable builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       exe: <
#         cipd_package: "executable/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#       properties: "{\"prop1\":\"val1\",\"prop2\":[\"val2\",123]}"
#     >
#     builders: <
#       name: "linux ci builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       swarming_tags: "tag1:val1"
#       swarming_tags: "tag2:val2"
#       dimensions: "builder:linux ci builder"
#       dimensions: "os:Linux"
#       dimensions: "300:prefer_if_available:first-choice"
#       dimensions: "prefer_if_available:fallback"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#         properties_j: "prop1:\"val1\""
#         properties_j: "prop2:[\"val2\",123]"
#       >
#       priority: 80
#       execution_timeout_secs: 10800
#       expiration_secs: 3600
#       caches: <
#         name: "name2"
#         path: "path2"
#       >
#       caches: <
#         name: "name3"
#         path: "path3"
#         wait_for_warm_cache_secs: 600
#       >
#       caches: <
#         name: "path1"
#         path: "path1"
#       >
#       build_numbers: YES
#       service_account: "builder@example.com"
#     >
#     builders: <
#       name: "watched builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#   >
# >
# buckets: <
#   name: "inline"
#   acls: <
#     role: WRITER
#     group: "admins"
#   >
#   acls: <
#     group: "all"
#   >
#   swarming: <
#     builders: <
#       name: "another builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "inline/recipe"
#         cipd_package: "recipe/bundles/inline"
#         cipd_version: "refs/heads/master"
#       >
#       service_account: "builder@example.com"
#     >
#     builders: <
#       name: "another executable builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       exe: <
#         cipd_package: "executable/bundles/inline"
#         cipd_version: "refs/heads/master"
#       >
#       properties: "{}"
#       service_account: "builder@example.com"
#     >
#     builders: <
#       name: "triggered builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "inline/recipe"
#         cipd_package: "recipe/bundles/inline"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "triggerer builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "inline/recipe"
#         cipd_package: "recipe/bundles/inline"
#         cipd_version: "refs/heads/master"
#       >
#       service_account: "builder@example.com"
#     >
#   >
# >
# buckets: <
#   name: "try"
#   acls: <
#     role: WRITER
#     group: "admins"
#   >
#   acls: <
#     group: "all"
#   >
#   acls: <
#     role: SCHEDULER
#     group: "devs"
#   >
#   swarming: <
#     builders: <
#       name: "builder with executable"
#       swarming_host: "chromium-swarm.appspot.com"
#       exe: <
#         cipd_package: "executable/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#       properties: "{}"
#     >
#     builders: <
#       name: "equivalent cq builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "generically named builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "linux try builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#     builders: <
#       name: "main cq builder"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe: <
#         name: "main/recipe"
#         cipd_package: "recipe/bundles/main"
#         cipd_version: "refs/heads/master"
#       >
#     >
#   >
# >
# ===
#
# === dir/custom.cfg
# hello!
# ===
#
# === luci-logdog.cfg
# reader_auth_groups: "all"
# archive_gs_bucket: "chromium-luci-logdog"
# ===
#
# === luci-milo.cfg
# consoles: <
#   id: "List view"
#   name: "List view"
#   builders: <
#     name: "buildbucket/luci.infra.ci/cron builder"
#   >
#   builders: <
#     name: "buildbucket/luci.infra.ci/generically named builder"
#   >
#   builders: <
#     name: "buildbucket/luci.infra.ci/linux ci builder"
#   >
#   builders: <
#     name: "buildbucket/luci.infra.inline/triggered builder"
#   >
#   favicon_url: "https://storage.googleapis.com/chrome-infra-public/logo/favicon.ico"
#   builder_view_only: true
# >
# consoles: <
#   id: "Console view"
#   name: "CI Builders"
#   repo_url: "https://noop.com"
#   refs: "regexp:refs/tags/blah"
#   refs: "regexp:refs/branch-heads/\\d+\\.\\d+"
#   exclude_ref: "refs/heads/master"
#   manifest_name: "REVISION"
#   builders: <
#     name: "buildbucket/luci.infra.ci/linux ci builder"
#     category: "a|b"
#     short_name: "lnx"
#   >
#   builders: <
#     name: "buildbucket/luci.infra.ci/cron builder"
#     category: "cron"
#   >
#   builders: <
#     name: "buildbucket/luci.infra.inline/triggered builder"
#   >
#   favicon_url: "https://storage.googleapis.com/chrome-infra-public/logo/favicon.ico"
#   header: <
#     links: <
#       name: "a"
#       links: <
#         text: "a"
#       >
#     >
#     links: <
#       name: "b"
#       links: <
#         text: "b"
#       >
#     >
#   >
#   include_experimental_builds: true
#   default_commit_limit: 3
#   default_expand: true
# >
# logo_url: "https://storage.googleapis.com/chrome-infra-public/logo/chrome-infra-logo-200x200.png"
# build_bug_template: <
#   summary: "Bug summary"
#   description: "Everything is broken"
#   monorail_project: "tutu, all aboard"
#   components: "Stuff>Hard"
# >
# ===
#
# === luci-notify.cfg
# notifiers: <
#   notifications: <
#     on_new_status: FAILURE
#     email: <
#       recipients: "someone@example,com"
#     >
#     template: "notifier-template"
#     notify_blamelist: <>
#   >
#   builders: <
#     bucket: "ci"
#     name: "cron builder"
#     repository: "https://cron.repo.example.com"
#   >
# >
# notifiers: <
#   notifications: <
#     on_new_status: FAILURE
#     email: <
#       recipients: "someone@example,com"
#     >
#     template: "notifier-template"
#     notify_blamelist: <>
#   >
#   builders: <
#     bucket: "ci"
#     name: "linux ci builder"
#     repository: "https://noop.com"
#   >
# >
# notifiers: <
#   notifications: <
#     on_new_status: FAILURE
#     email: <
#       recipients: "someone@example,com"
#     >
#     template: "notifier-template"
#     notify_blamelist: <>
#   >
#   builders: <
#     bucket: "ci"
#     name: "watched builder"
#     repository: "https://custom.example.com/repo"
#   >
# >
# ===
#
# === luci-notify/email-templates/another-template.template
# Boo!
# ===
#
# === luci-notify/email-templates/notifier-template.template
# Hello
#
# Hi
# ===
#
# === luci-scheduler.cfg
# job: <
#   id: "another builder"
#   acl_sets: "inline"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.inline"
#     builder: "another builder"
#   >
# >
# job: <
#   id: "another executable builder"
#   acl_sets: "inline"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.inline"
#     builder: "another executable builder"
#   >
# >
# job: <
#   id: "cron builder"
#   schedule: "0 6 * * *"
#   acl_sets: "ci"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.ci"
#     builder: "cron builder"
#   >
# >
# job: <
#   id: "generically named builder"
#   acls: <
#     role: TRIGGERER
#     granted_to: "builder@example.com"
#   >
#   acl_sets: "ci"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.ci"
#     builder: "generically named builder"
#   >
# >
# job: <
#   id: "generically named executable builder"
#   acls: <
#     role: TRIGGERER
#     granted_to: "builder@example.com"
#   >
#   acl_sets: "ci"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.ci"
#     builder: "generically named executable builder"
#   >
# >
# job: <
#   id: "linux ci builder"
#   acl_sets: "ci"
#   triggering_policy: <
#     kind: GREEDY_BATCHING
#     max_concurrent_invocations: 5
#     max_batch_size: 10
#   >
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.ci"
#     builder: "linux ci builder"
#   >
# >
# job: <
#   id: "triggered builder"
#   acls: <
#     role: TRIGGERER
#     granted_to: "builder@example.com"
#   >
#   acl_sets: "inline"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.inline"
#     builder: "triggered builder"
#   >
# >
# job: <
#   id: "triggerer builder"
#   acl_sets: "inline"
#   buildbucket: <
#     server: "cr-buildbucket.appspot.com"
#     bucket: "luci.infra.inline"
#     builder: "triggerer builder"
#   >
# >
# trigger: <
#   id: "inline poller"
#   schedule: "with 10s interval"
#   acl_sets: "inline"
#   triggers: "another builder"
#   triggers: "another executable builder"
#   triggers: "triggerer builder"
#   gitiles: <
#     repo: "https://noop.com"
#     refs: "regexp:refs/heads/master"
#     refs: "regexp:refs/tags/blah"
#     refs: "regexp:refs/branch-heads/\\d+\\.\\d+"
#   >
# >
# trigger: <
#   id: "master-poller"
#   schedule: "with 10s interval"
#   acl_sets: "ci"
#   triggers: "generically named builder"
#   triggers: "generically named executable builder"
#   triggers: "linux ci builder"
#   gitiles: <
#     repo: "https://noop.com"
#     refs: "regexp:refs/heads/master"
#     refs: "regexp:refs/tags/blah"
#     refs: "regexp:refs/branch-heads/\\d+\\.\\d+"
#     path_regexps: ".*"
#     path_regexps_exclude: "excluded"
#   >
# >
# acl_sets: <
#   name: "ci"
#   acls: <
#     role: OWNER
#     granted_to: "group:admins"
#   >
#   acls: <
#     granted_to: "group:all"
#   >
#   acls: <
#     role: TRIGGERER
#     granted_to: "group:devs"
#   >
#   acls: <
#     role: TRIGGERER
#     granted_to: "project:some-project"
#   >
# >
# acl_sets: <
#   name: "inline"
#   acls: <
#     role: OWNER
#     granted_to: "group:admins"
#   >
#   acls: <
#     granted_to: "group:all"
#   >
# >
# ===
#
# === project.cfg
# name: "infra"
# access: "group:all"
# ===
