# Copyright 2020 The LUCI Authors.
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

"""Internal API for registering and checking experiments."""


def _register(experiment_id):
  """Register an experiment with the given ID.

  This ID becomes a valid target for lucicfg.enable_experiment(...) call.
  Returns an object that can be used to check whether the experiment was
  enabled. It is fine and even recommended to register the experiments during
  the module loading time:

  ```python
  load('@stdlib//internal/experiments.star', 'experiments')
  new_cool_feature_experiment = experiments.register('new_cool_feature')

  ...

  def something(...):
    new_cool_feature_experiment.require()
  ```

  Registering the same experiment ID multiple times is fine, it results in the
  exact same experiment.

  Args:
    experiment_id: an ID of the experiment to register.

  Returns:
    A struct with methods `require()` and `is_enabled()`:
      `is_enabled()` returns True if the experiment is enabled.
      `require()` fails the execution if the experiment is not enabled.
  """
  __native__.register_experiment(experiment_id)
  return struct(
      is_enabled = lambda: _is_enabled(experiment_id),
      require = lambda: _require(experiment_id),
  )


def _require(experiment_id):
  """Fails the execution if the given experiment ID is not enabled.

  Args:
    experiment_id: a string with the ID of the experiment to check.
  """
  if not _is_enabled(experiment_id):
    fail(
        'This feature requires enabling the experiment %r.\n' % experiment_id +
        'To enable it, add this line somewhere early in your script:\n' +
        '  lucicfg.enable_experiment(%r)\n' % experiment_id  +
        'Note that using experimental features comes with zero guarantees. ' +
        'See the doc for lucicfg.enable_experiment(...) for more information'
    )


def _is_enabled(experiment_id):
  """Returns True if the experiment was enabled."""
  return __native__.is_experiment_enabled(experiment_id)


experiments = struct(
    register = _register,
)
