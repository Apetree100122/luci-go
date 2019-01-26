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
load('@stdlib//internal/validate.star', 'validate')

load('@stdlib//internal/luci/common.star', 'keys')


def logdog(gs_bucket=None):
  """Defines configuration of the LogDog service for this project.

  Usually required for any non-trivial project.

  Args:
    gs_bucket: base Google Storage archival path, archive logs will be written
        to this bucket/path.
  """
  graph.add_node(keys.logdog(), props = {
      'gs_bucket': validate.string('gs_bucket', gs_bucket, required=False),
  })
  graph.add_edge(keys.project(), keys.logdog())
