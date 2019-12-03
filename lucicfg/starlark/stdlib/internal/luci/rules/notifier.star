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

load('@stdlib//internal/graph.star', 'graph')
load('@stdlib//internal/lucicfg.star', 'lucicfg')
load('@stdlib//internal/validate.star', 'validate')

load('@stdlib//internal/luci/common.star', 'keys')
load('@proto//go.chromium.org/luci/buildbucket/proto/common.proto', buildbucket_common_pb='buildbucket.v2')


def _notifier(
      ctx,
      *,
      name=None,

      # Conditions.
      on_occurrence=None,
      on_new_status=None,

      # Deprecated conditions.
      on_failure=None,
      on_new_failure=None,
      on_status_change=None,
      on_success=None,

      # Who to notify.
      notify_emails=None,
      notify_blamelist=None,

      # Tweaks.
      blamelist_repos_whitelist=None,
      template=None,

      # Relations.
      notified_by=None
  ):
  """Defines a notifier that sends notifications on events from builders.

  A notifier contains a set of conditions specifying what events are considered
  interesting (e.g. a previously green builder has failed), and a set of
  recipients to notify when an interesting event happens. The conditions are
  specified via `on_*` fields, and recipients are specified via `notify_*`
  fields.

  The set of builders that are being observed is defined through `notified_by`
  field here or `notifies` field in luci.builder(...). Whenever a build
  finishes, the builder "notifies" all luci.notifier(...) objects subscribed to
  it, and in turn each notifier filters and forwards this event to corresponding
  recipients.

  Args:
    name: name of this notifier to reference it from other rules. Required.

    on_occurrence: a list specifying which build statuses to notify for.
        Notifies for every build status specified. Valid values are string
        literals `SUCCESS`, `FAILURE`, and `INFRA_FAILURE`. Default is None.
    on_new_status: a list specifying which new build statuses to notify for.
        Notifies for each build status specified unless the previous build
        was the same status. Valid values are string literals `SUCCESS`,
        `FAILURE`, and `INFRA_FAILURE`. Default is None.
    on_failure: Deprecated. Please use `on_new_status` or `on_occurrence`
        instead. If True, notify on each build failure. Ignores transient (aka
        "infra") failures. Default is False.
    on_new_failure: Deprecated. Please use `on_new_status` or `on_occurrence`
        instead.  If True, notify on a build failure unless the previous build
        was a failure too. Ignores transient (aka "infra") failures. Default is
        False.
    on_status_change: Deprecated. Please use `on_new_status` or `on_occurrence`
        instead. If True, notify on each change to a build status (e.g.
        a green build becoming red and vice versa). Default is False.
    on_success: Deprecated. Please use `on_new_status` or `on_occurrence`
        instead. If True, notify on each build success. Default is False.

    notify_emails: an optional list of emails to send notifications to.
    notify_blamelist: if True, send notifications to everyone in the computed
        blamelist for the build. Works only if the builder has a repository
        associated with it, see `repo` field in luci.builder(...). Default is
        False.

    blamelist_repos_whitelist: an optional list of repository URLs (e.g.
        `https://host/repo`) to restrict the blamelist calculation to. If empty
        (default), only the primary repository associated with the builder is
        considered, see `repo` field in luci.builder(...).
    template: a luci.notifier_template(...) to use to format notification
        emails. If not specified, and a template named `default` is defined
        in the project somewhere, it is used implicitly by the notifier.

    notified_by: builders to receive status notifications from. This relation
        can also be defined via `notifies` field in luci.builder(...).
  """
  name = validate.string('name', name)

  on_occurrence = _buildbucket_status_validate(validate.list('on_occurrence', on_occurrence))
  on_new_status = _buildbucket_status_validate(validate.list('on_new_status', on_new_status))

  # deprecated
  on_failure = validate.bool('on_failure', on_failure, required=False)
  on_new_failure = validate.bool('on_new_failure', on_new_failure, required=False)
  on_status_change = validate.bool('on_status_change', on_status_change, required=False)
  on_success = validate.bool('on_success', on_success, required=False)

  if not(on_failure or on_new_failure or on_status_change or on_success or on_occurrence or on_new_status):
    fail('at least one on_... condition is required')

  notify_emails = validate.list('notify_emails', notify_emails)
  for e in notify_emails:
    validate.string('notify_emails', e)

  notify_blamelist = validate.bool('notify_blamelist', notify_blamelist, required=False)
  blamelist_repos_whitelist = validate.list('blamelist_repos_whitelist', blamelist_repos_whitelist)
  for repo in blamelist_repos_whitelist:
    validate.repo_url('blamelist_repos_whitelist', repo)
  if blamelist_repos_whitelist and not notify_blamelist:
    fail('blamelist_repos_whitelist requires notify_blamelist to be True')

  key = keys.notifier(name)
  graph.add_node(key, idempotent = True, props = {
      'name': name,
      'on_occurrence': on_occurrence,
      'on_new_status': on_new_status,

      'on_failure': on_failure,
      'on_new_failure': on_new_failure,
      'on_status_change': on_status_change,
      'on_success': on_success,

      'notify_emails': notify_emails,
      'notify_blamelist': notify_blamelist,
      'blamelist_repos_whitelist': blamelist_repos_whitelist,
  })
  graph.add_edge(keys.project(), key)

  for b in validate.list('notified_by', notified_by):
    graph.add_edge(
        parent = key,
        child = keys.builder_ref(b, attr='notified_by'),
        title = 'notified_by',
    )

  if template != None:
    graph.add_edge(key, keys.notifier_template(template))

  return graph.keyset(key)

def _buildbucket_status_validate(status_list):
  """Validates a list of buildbucket statuses and returns their corresponding
  identifiers.

  Args:
    status_list: a list of status strings; i.e. "SUCCESS" or "FAILURE"

  Returns:
    A list of the enum identifiers corresponding to the given statuses.
  """
  for status in status_list:
    if not hasattr(buildbucket_common_pb, status):
      fail('%s is not a valid status. Try one of the following %s.' %
        (status, str(dir(buildbucket_common_pb))))

  return [getattr(buildbucket_common_pb, status) for status in status_list]


notifier = lucicfg.rule(impl = _notifier)
