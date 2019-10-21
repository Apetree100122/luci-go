# Copyright 2018 The LUCI Authors.
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
load('@stdlib//internal/luci/lib/acl.star', 'aclimpl')
load('@stdlib//internal/luci/lib/service.star', 'service')


def _project(
      ctx,
      *,
      name=None,
      config_dir=None,

      buildbucket=None,
      logdog=None,
      milo=None,
      notify=None,
      scheduler=None,
      swarming=None,

      acls=None
  ):
  """Defines a LUCI project.

  There should be exactly one such definition in the top-level config file.

  Args:
    name: full name of the project. Required.
    config_dir: a subdirectory of the config output directory (see `config_dir`
        in lucicfg.config(...)) to place generated LUCI configs under. Default
        is `.`. A custom value is useful when using `lucicfg` to generate LUCI
        and non-LUCI configs at the same time.
    buildbucket: appspot hostname of a Buildbucket service to use (if any).
    logdog: appspot hostname of a LogDog service to use (if any).
    milo: appspot hostname of a Milo service to use (if any).
    notify: appspot hostname of a LUCI Notify service to use (if any).
    scheduler: appspot hostname of a LUCI Scheduler service to use (if any).
    swarming: appspot hostname of a Swarming service to use (if any).
    acls: list of acl.entry(...) objects, will be inherited by all buckets.
  """
  key = keys.project()
  graph.add_node(key, props = {
      'name': validate.string('name', name),
      'config_dir': validate.relative_path('config_dir', config_dir, required=False, default='.'),
      'buildbucket': service.from_host('buildbucket', buildbucket),
      'logdog': service.from_host('logdog', logdog),
      'milo': service.from_host('milo', milo),
      'notify': service.from_host('notify', notify),
      'scheduler': service.from_host('scheduler', scheduler),
      'swarming': service.from_host('swarming', swarming),
      'acls': aclimpl.validate_acls(acls, project_level=True),
  })
  return graph.keyset(key)


project = lucicfg.rule(impl = _project)
