// Copyright 2023 The LUCI Authors.
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

package clients

import (
	"context"

	"google.golang.org/grpc"

	"go.chromium.org/luci/common/errors"

	pb "go.chromium.org/luci/buildbucket/proto"
)

type contextKey string

var MockTaskBackendClientKey = contextKey("used in tests only for setting the mock SwarmingClient")

// BackendClient is the client to communicate with TaskBackend.
// It wraps a pb.TaskBackendClient.
type BackendClient struct {
	client TaskBackendClient
}

type TaskBackendClient interface {
	RunTask(ctx context.Context, taskReq *pb.RunTaskRequest, opts ...grpc.CallOption) (*pb.RunTaskResponse, error)
	FetchTasks(ctx context.Context, in *pb.FetchTasksRequest, opts ...grpc.CallOption) (*pb.FetchTasksResponse, error)
	ValidateConfigs(ctx context.Context, in *pb.ValidateConfigsRequest, opts ...grpc.CallOption) (*pb.ValidateConfigsResponse, error)
}

func newRawTaskBackendClient(ctx context.Context, host string, project string) (TaskBackendClient, error) {
	if mockClient, ok := ctx.Value(MockTaskBackendClientKey).(TaskBackendClient); ok {
		return mockClient, nil
	}
	prpcClient, err := CreateRawPrpcClient(ctx, host, project)
	if err != nil {
		return nil, err
	}
	return pb.NewTaskBackendPRPCClient(prpcClient), nil
}

func ComputeHostnameFromTarget(target string, globalCfg *pb.SettingsCfg) (hostname string, err error) {
	if globalCfg == nil {
		return "", errors.Reason("could not get global settings config").Err()
	}
	for _, config := range globalCfg.Backends {
		if config.Target == target {
			return config.Hostname, nil
		}
	}
	return "", errors.Reason("could not find target in global config settings").Err()
}

// NewBackendClient creates a client to communicate with Buildbucket.
func NewBackendClient(ctx context.Context, project, target string, globalCfg *pb.SettingsCfg) (*BackendClient, error) {
	hostname, err := ComputeHostnameFromTarget(target, globalCfg)
	if err != nil {
		return nil, err
	}
	client, err := newRawTaskBackendClient(ctx, hostname, project)
	if err != nil {
		return nil, err
	}
	return &BackendClient{
		client: client,
	}, nil
}

// RunTask returns for the requested task.
func (c *BackendClient) RunTask(ctx context.Context, taskReq *pb.RunTaskRequest, opts ...grpc.CallOption) (*pb.RunTaskResponse, error) {
	return c.client.RunTask(ctx, taskReq, opts...)
}

// FetchTasks returns the requested tasks.
func (c *BackendClient) FetchTasks(ctx context.Context, taskReq *pb.FetchTasksRequest, opts ...grpc.CallOption) (*pb.FetchTasksResponse, error) {
	return c.client.FetchTasks(ctx, taskReq, opts...)
}

// ValidateConfigs returns validation errors (if any).
func (c *BackendClient) ValidateConfigs(ctx context.Context, req *pb.ValidateConfigsRequest, opts ...grpc.CallOption) (*pb.ValidateConfigsResponse, error) {
	return c.client.ValidateConfigs(ctx, req, opts...)
}
