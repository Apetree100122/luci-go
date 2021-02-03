// Copyright 2021 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is a client to a Swarming server.
//
// The reference server python implementation documentation can be found at
// https://github.com/luci/luci-py/tree/master/appengine/swarming/doc
package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/maruel/subcommands"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/client/cmd/swarming/lib"
	"go.chromium.org/luci/common/api/swarming/swarming/v1"
)

// IntegrationTestEnvVar is the name of the environment variable which controls
// whether integaration tests are executed.
// The value must be "1" for integration tests to run.
const IntegrationTestEnvVar = "INTEGRATION_TESTS"

// runIntegrationTests true if integration tests should run.
func runIntegrationTests() bool {
	return os.Getenv(IntegrationTestEnvVar) == "1"
}

// runCmd runs swarming commands appending common flags.
// It skips if integration should not run.
func runCmd(t *testing.T, cmd string, args ...string) int {
	if !runIntegrationTests() {
		t.Skipf("Skip integration tests")
	}
	args = append([]string{cmd, "-server", "chromium-swarm-dev.appspot.com", "-quiet"}, args...)
	return subcommands.Run(getApplication(), args)
}

func TestBotsCommand(t *testing.T) {
	t.Parallel()
	Convey(`ok`, t, func() {
		dir := t.TempDir()
		jsonPath := filepath.Join(dir, "out.json")

		So(runCmd(t, "bots", "-json", jsonPath), ShouldEqual, 0)
	})
}

func TestTasksCommand(t *testing.T) {
	t.Parallel()
	Convey(`ok`, t, func() {
		dir := t.TempDir()
		jsonPath := filepath.Join(dir, "out.json")

		So(runCmd(t, "tasks", "-limit", "1", "-json", jsonPath), ShouldEqual, 0)
	})
}

// triggerTaskWithIsolate triggers a task that uploads ouput to Isolate, and returns triggered TaskRequest.
func triggerTaskWithIsolate(t *testing.T) *swarming.SwarmingRpcsTaskRequestMetadata {
	return triggerTask(t, []string{"-isolate-server", "isolateserver-dev.appspot.com"})
}

// triggerTaskWithCAS triggers a task that uploads ouput to CAS, and returns triggered TaskRequest.
func triggerTaskWithCAS(t *testing.T) *swarming.SwarmingRpcsTaskRequestMetadata {
	// TODO(jwata): ensure the digest is uploaded on CAS.
	// https://cas-viewer-dev.appspot.com/projects/chromium-swarm-dev/instances/default_instance/blobs/1febd720bb5e438578194b08ace1e6da072a9741068923798fd5b41856190710/77/tree
	return triggerTask(t, []string{"-digest", "1febd720bb5e438578194b08ace1e6da072a9741068923798fd5b41856190710/77"})
}

// triggerTask triggers a task and returns the triggered TaskRequest.
func triggerTask(t *testing.T, args []string) *swarming.SwarmingRpcsTaskRequestMetadata {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "out.json")
	args = append(args, []string{
		"-d", "pool=chromium.tests",
		"-d", "os=Linux",
		"-dump-json", jsonPath,
		"-idempotent",
		"--", "/bin/bash", "-c", "echo hi > ${ISOLATED_OUTDIR}/out",
	}...)
	So(runCmd(t, "trigger", args...), ShouldEqual, 0)

	results := readTriggerResults(jsonPath)
	So(results.Tasks, ShouldHaveLength, 1)
	return results.Tasks[0]
}

// readTriggerResults reads TriggerResults from output json file.
func readTriggerResults(jsonPath string) *lib.TriggerResults {
	resultsJSON, err := ioutil.ReadFile(jsonPath)
	So(err, ShouldBeNil)

	results := &lib.TriggerResults{}
	err = json.Unmarshal(resultsJSON, results)
	So(err, ShouldBeNil)

	return results
}

// unsetParentTaskID unset SWARMING_TASK_ID environment variable, otherwise task trigger may fail for parent task association.
func unsetParentTaskID(t *testing.T) {
	parentTaskID := os.Getenv(lib.TaskIDEnvVar)
	os.Unsetenv(lib.TaskIDEnvVar)
	t.Cleanup(func() {
		os.Setenv(lib.TaskIDEnvVar, parentTaskID)
	})
}

func TestTriggerCommand(t *testing.T) {
	unsetParentTaskID(t)
	Convey(`ok with Isolate server`, t, func() {
		triggerTaskWithIsolate(t)
	})
	Convey(`ok with CAS`, t, func() {
		triggerTaskWithCAS(t)
	})
}

func testCollectCommand(t *testing.T, taskID string) {
	dir := t.TempDir()
	So(runCmd(t, "collect", "-output-dir", dir, taskID), ShouldEqual, 0)
	out, err := ioutil.ReadFile(filepath.Join(dir, taskID, "out"))
	So(err, ShouldBeNil)
	So(string(out), ShouldResemble, "hi\n")
}

func TestCollectCommand(t *testing.T) {
	unsetParentTaskID(t)

	Convey(`ok with Isolate server`, t, func() {
		triggeredTask := triggerTaskWithIsolate(t)
		testCollectCommand(t, triggeredTask.TaskId)
	})

	Convey(`ok with CAS`, t, func() {
		triggeredTask := triggerTaskWithCAS(t)
		testCollectCommand(t, triggeredTask.TaskId)
	})
}

func TestRequestShowCommand(t *testing.T) {
	unsetParentTaskID(t)

	Convey(`ok with Isolate`, t, func() {
		triggeredTask := triggerTaskWithIsolate(t)
		So(runCmd(t, "request_show", triggeredTask.TaskId), ShouldEqual, 0)
	})

	Convey(`ok with CAS`, t, func() {
		triggeredTask := triggerTaskWithCAS(t)
		So(runCmd(t, "request_show", triggeredTask.TaskId), ShouldEqual, 0)
	})
}

const spawnTaskInputJSON = `
{
	"requests": [
		{
			"name": "spawn-task test with Isolate server",
			"priority": "200",
			"task_slices": [
				{
					"expiration_secs": "21600",
					"properties": {
						"dimensions": [
							{"key": "pool", "value": "chromium.tests"},
							{"key": "os", "value": "Linux"}
						],
						"command": ["/bin/bash", "-c", "echo hi > ${ISOLATED_OUTDIR}/out"],
						"execution_timeout_secs": "3600",
						"idempotent": true
					}
				}
			]
		},
		{
			"name": "spawn-task test with CAS",
			"priority": "200",
			"task_slices": [
				{
					"expiration_secs": "21600",
					"properties": {
						"dimensions": [
							{"key": "pool", "value": "chromium.tests"},
							{"key": "os", "value": "Linux"}
						],
						"cas_input_root": {
							"cas_instance": "projects/chromium-swarm-dev/instances/default_instance",
							"digest": {
								"hash": "70a9a5e030074dc7eb69d167e91a47fadd3f14c14a52be85fd10f57cfb72dd0a",
								"size_bytes": "77"
							}
						},
						"command": ["/bin/bash", "-c", "echo hi > ${ISOLATED_OUTDIR}/out"],
						"execution_timeout_secs": "3600",
						"idempotent": true
					}
				}
			]
		}
	]
}
`

func TestSpawnTasksCommand(t *testing.T) {
	unsetParentTaskID(t)
	Convey(`ok`, t, func() {
		// prepare input file.
		dir := t.TempDir()
		inputPath := filepath.Join(dir, "input.json")
		err := ioutil.WriteFile(inputPath, []byte(spawnTaskInputJSON), 0600)
		So(err, ShouldBeNil)

		outputPath := filepath.Join(dir, "output.json")
		So(runCmd(t, "spawn_task", "-json-input", inputPath, "-json-output", outputPath), ShouldEqual, 0)

		results := readTriggerResults(outputPath)
		So(results.Tasks, ShouldHaveLength, 2)
	})
}
