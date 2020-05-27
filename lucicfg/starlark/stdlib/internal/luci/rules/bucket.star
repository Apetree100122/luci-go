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
load('@stdlib//internal/luci/lib/realms.star', 'realms')

load('@stdlib//internal/luci/rules/realm.star', 'realm')


def _bucket(ctx, *, name=None, acls=None):
  """Defines a bucket: a container for LUCI resources that share the same ACL.

  Args:
    name: name of the bucket, e.g. `ci` or `try`. Required.
    acls: list of acl.entry(...) objects.
  """
  name = validate.string('name', name)
  if name.startswith('luci.'):
    fail('long bucket names aren\'t allowed, please use e.g. "try" instead of "luci.project.try"')

  key = keys.bucket(name)
  graph.add_node(key, props = {
      'name': name,
      'acls': aclimpl.validate_acls(acls, project_level=False),
  })
  graph.add_edge(keys.project(), key)

  # Each bucket also has an associated realm with the matching name.
  if realms.experiment.is_enabled():
    realm(name = name)

  return graph.keyset(key)


bucket = lucicfg.rule(impl = _bucket)
