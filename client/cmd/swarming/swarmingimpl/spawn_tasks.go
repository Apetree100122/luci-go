// Copyright 2018 The LUCI Authors.
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

package swarmingimpl

import (
	"context"
	"flag"
	"io"
	"os"
	"sync"

	"github.com/maruel/subcommands"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/sync/parallel"

	"go.chromium.org/luci/client/cmd/swarming/swarmingimpl/base"
	"go.chromium.org/luci/client/cmd/swarming/swarmingimpl/clipb"
	"go.chromium.org/luci/swarming/client/swarming"
	swarmingv2 "go.chromium.org/luci/swarming/proto/api_v2"
)

// CmdSpawnTasks returns an object for the `spawn-tasks` subcommand.
func CmdSpawnTasks(authFlags base.AuthFlags) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "spawn-tasks -S <server> -json-input <path>",
		ShortDesc: "spawns a set of Swarming tasks defined in a JSON file",
		LongDesc:  "Spawns a set of Swarming tasks defined in a JSON file.",
		CommandRun: func() subcommands.CommandRun {
			return base.NewCommandRun(authFlags, &spawnTasksImpl{}, base.Features{
				MinArgs:         0,
				MaxArgs:         0,
				MeasureDuration: true,
				OutputJSON: base.OutputJSON{
					Enabled:         true,
					DefaultToStdout: true,
				},
			})
		},
	}
}

type spawnTasksImpl struct {
	jsonInput        string
	cancelExtraTasks bool

	requests []*swarmingv2.NewTaskRequest
}

func (cmd *spawnTasksImpl) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&cmd.jsonInput, "json-input", "", "(required) Read Swarming task requests from this file.")
	// TODO(https://crbug.com/997221): Remove this option.
	fs.BoolVar(&cmd.cancelExtraTasks, "cancel-extra-tasks", false, "Legacy option that does absolutely nothing.")
}

func (cmd *spawnTasksImpl) ParseInputs(args []string, env subcommands.Env) error {
	if cmd.jsonInput == "" {
		return errors.Reason("input JSON file is required, pass it via -json-input").Err()
	}

	tasksFile, err := os.Open(cmd.jsonInput)
	if err != nil {
		return errors.Annotate(err, "failed to open -json-input tasks file").Err()
	}
	defer tasksFile.Close()

	cmd.requests, err = processTasksStream(tasksFile, env)
	return err
}

func (cmd *spawnTasksImpl) Execute(ctx context.Context, svc swarming.Client, extra base.Extra) (any, error) {
	results, merr := createNewTasks(ctx, svc, cmd.requests)
	return &clipb.SpawnTasksOutput{Tasks: results}, merr
}

func processTasksStream(tasks io.Reader, env subcommands.Env) ([]*swarmingv2.NewTaskRequest, error) {
	blob, err := io.ReadAll(tasks)
	if err != nil {
		return nil, errors.Annotate(err, "reading tasks file").Err()
	}
	var requests clipb.SpawnTasksInput
	if err := protojson.Unmarshal(blob, &requests); err != nil {
		return nil, errors.Annotate(err, "decoding tasks file").Err()
	}
	// Populate the tasks with information about the current environment
	// if they're not already set.
	for _, ntr := range requests.Requests {
		if ntr.User == "" {
			ntr.User = env[swarming.UserEnvVar].Value
		}
		if ntr.ParentTaskId == "" {
			ntr.ParentTaskId = env[swarming.TaskIDEnvVar].Value
		}
	}
	return requests.Requests, nil
}

func createNewTasks(ctx context.Context, service swarming.Client, requests []*swarmingv2.NewTaskRequest) ([]*swarmingv2.TaskRequestMetadataResponse, error) {
	var mu sync.Mutex
	results := make([]*swarmingv2.TaskRequestMetadataResponse, 0, len(requests))
	err := parallel.WorkPool(8, func(gen chan<- func() error) {
		for _, request := range requests {
			request := request
			gen <- func() error {
				result, err := service.NewTask(ctx, request)
				if err != nil {
					return err
				}
				mu.Lock()
				defer mu.Unlock()
				results = append(results, result)
				return nil
			}
		}
	})
	return results, err
}
