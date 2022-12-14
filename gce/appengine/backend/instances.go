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

package backend

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"go.chromium.org/luci/appengine/tq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"

	"go.chromium.org/luci/gce/api/tasks/v1"
	"go.chromium.org/luci/gce/appengine/backend/internal/metrics"
	"go.chromium.org/luci/gce/appengine/model"
)

// setCreated sets the GCE instance as created in the datastore if it isn't already.
func setCreated(c context.Context, id string, inst *compute.Instance) error {
	t, err := time.Parse(time.RFC3339, inst.CreationTimestamp)
	if err != nil {
		return errors.Annotate(err, "failed to parse instance creation time").Err()
	}
	nics := make([]model.NetworkInterface, len(inst.NetworkInterfaces))
	for i, n := range inst.NetworkInterfaces {
		if len(n.AccessConfigs) > 0 {
			// GCE currently supports at most one access config per network interface.
			nics[i].ExternalIP = n.AccessConfigs[0].NatIP
			if len(n.AccessConfigs) > 1 {
				logging.Warningf(c, "network interface %q has more than one access config", n.Name)
			}
		}
		nics[i].InternalIP = n.NetworkIP
	}
	vm := &model.VM{
		ID: id,
	}
	put := false
	err = datastore.RunInTransaction(c, func(c context.Context) error {
		put = false
		switch err := datastore.Get(c, vm); {
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		case vm.Created > 0:
			return nil
		}
		vm.Created = t.Unix()
		vm.NetworkInterfaces = nics
		vm.URL = inst.SelfLink
		if err := datastore.Put(c, vm); err != nil {
			return errors.Annotate(err, "failed to store VM").Err()
		}
		put = true
		return nil
	}, nil)
	if put && err == nil {
		metrics.ReportCreationTime(c, float64(vm.Created-vm.Configured), vm.Prefix, vm.Attributes.GetProject(), vm.Attributes.GetZone())
	}
	return err
}

// logErrors logs the errors in the given *googleapi.Error.
func logErrors(c context.Context, name string, err *googleapi.Error) {
	var errMsgs []string
	for _, err := range err.Errors {
		errMsgs = append(errMsgs, err.Message)
	}
	logging.Errorf(c, "failure for %s, HTTP: %d, errors: %s", name, err.Code, strings.Join(errMsgs, ","))
}

// rateLimitExceeded returns whether the given *googleapi.Error contains a rate
// limit error.
func rateLimitExceeded(err *googleapi.Error) bool {
	for _, err := range err.Errors {
		switch {
		case strings.Contains(err.Message, "Queries per user per 100 seconds"):
			return true
		case strings.Contains(err.Message, "Rate Limit Exceeded"):
			return true
		case strings.Contains(err.Message, "rateLimitExceeded"):
			return true
		case strings.Contains(err.Reason, "rateLimitExceeded"):
			return true
		}
	}
	return false
}

// checkInstance fetches the GCE instance and either sets its creation details
// or deletes the VM if the instance doesn't exist.
func checkInstance(c context.Context, vm *model.VM) error {
	srv := getCompute(c).Instances
	call := srv.Get(vm.Attributes.GetProject(), vm.Attributes.GetZone(), vm.Hostname)
	inst, err := call.Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				metrics.UpdateFailures(c, gerr.Code, vm)
				if err := deleteVM(c, vm.ID, vm.Hostname); err != nil {
					return errors.Annotate(err, "instance not found").Err()
				}
				return errors.Annotate(err, "instance not found").Err()
			}
			logErrors(c, vm.Hostname, gerr)
		}
		return errors.Annotate(err, "failed to fetch instance").Err()
	}
	logging.Debugf(c, "created instance: %s", inst.SelfLink)
	return setCreated(c, vm.ID, inst)
}

// createInstanceQueue is the name of the create instance task handler queue.
const createInstanceQueue = "create-instance"

// createInstance creates a GCE instance.
func createInstance(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.CreateInstance)
	switch {
	case !ok:
		return errors.Reason("unexpected payload %q", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	}
	vm := &model.VM{
		ID: task.Id,
	}
	switch err := datastore.Get(c, vm); {
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	case vm.URL != "":
		logging.Debugf(c, "instance exists: %s", vm.URL)
		return nil
	}
	logging.Debugf(c, "creating instance %q", vm.Hostname)
	// Generate a request ID based on the hostname.
	// Ensures duplicate operations aren't created in GCE.
	// Request IDs are valid for 24 hours.
	rID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("create-%s", vm.Hostname)))
	srv := getCompute(c).Instances
	call := srv.Insert(vm.Attributes.GetProject(), vm.Attributes.GetZone(), vm.GetInstance())
	op, err := call.RequestId(rID.String()).Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			logErrors(c, vm.Hostname, gerr)
			metrics.UpdateFailures(c, gerr.Code, vm)
			// TODO(b/130826296): Remove this once rate limit returns a transient HTTP error code.
			if rateLimitExceeded(gerr) {
				return errors.Annotate(err, "rate limit exceeded creating instance %s", vm.Hostname).Err()
			}
			if gerr.Code == http.StatusTooManyRequests || gerr.Code >= 500 {
				return errors.Annotate(err, "transiently failed to create instance %s", vm.Hostname).Err()
			}
			if err := deleteVM(c, task.Id, vm.Hostname); err != nil {
				return errors.Annotate(err, "failed to create instance %s", vm.Hostname).Err()
			}
		}
		return errors.Annotate(err, "failed to create instance %s", vm.Hostname).Err()
	}
	if op.Error != nil && len(op.Error.Errors) > 0 {
		for _, err := range op.Error.Errors {
			logging.Errorf(c, "for vm %s, %s: %s", vm.Hostname, err.Code, err.Message)
		}
		metrics.UpdateFailures(c, 200, vm)
		if err := deleteVM(c, task.Id, vm.Hostname); err != nil {
			return errors.Annotate(err, "failed to create instance %s", vm.Hostname).Err()
		}
		return errors.Reason("failed to create instance %s", vm.Hostname).Err()
	}
	if op.Status == "DONE" {
		return checkInstance(c, vm)
	}
	// Instance creation is pending.
	return nil
}

// destroyInstanceAsync schedules a task queue task to destroy a GCE instance.
func destroyInstanceAsync(c context.Context, id, url string) error {
	t := &tq.Task{
		Payload: &tasks.DestroyInstance{
			Id:  id,
			Url: url,
		},
	}
	if err := getDispatcher(c).AddTask(c, t); err != nil {
		return errors.Annotate(err, "failed to schedule destroy task").Err()
	}
	return nil
}

// destroyInstanceQueue is the name of the destroy instance task handler queue.
const destroyInstanceQueue = "destroy-instance"

// destroyInstance destroys a GCE instance.
func destroyInstance(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.DestroyInstance)
	switch {
	case !ok:
		return errors.Reason("unexpected payload type %T", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	case task.GetUrl() == "":
		return errors.Reason("URL is required").Err()
	}
	vm := &model.VM{
		ID: task.Id,
	}
	switch err := datastore.Get(c, vm); {
	case err == datastore.ErrNoSuchEntity:
		return nil
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM %s", vm.ID).Err()
	case vm.URL != task.Url:
		// Instance is already destroyed and replaced. Don't destroy the new one.
		logging.Debugf(c, "instance %s does not exist: %s", vm.Hostname, task.Url)
		return nil
	}
	logging.Debugf(c, "destroying instance %q", vm.Hostname)
	rID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("destroy-%s", vm.Hostname)))
	srv := getCompute(c).Instances
	call := srv.Delete(vm.Attributes.GetProject(), vm.Attributes.GetZone(), vm.Hostname)
	op, err := call.RequestId(rID.String()).Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				// Instance is already destroyed.
				logging.Debugf(c, "instance does not exist: %s", vm.URL)
				return deleteBotAsync(c, task.Id, vm.Hostname)
			}
			logErrors(c, vm.Hostname, gerr)
		}
		return errors.Annotate(err, "failed to destroy instance %s", vm.Hostname).Err()
	}
	if op.Error != nil && len(op.Error.Errors) > 0 {
		for _, err := range op.Error.Errors {
			logging.Errorf(c, "%s: %s", err.Code, err.Message)
		}
		return errors.Reason("failed to destroy instance %s", vm.Hostname).Err()
	}
	if op.Status == "DONE" {
		logging.Debugf(c, "destroyed instance: %s", op.TargetLink)
		return deleteBotAsync(c, task.Id, vm.Hostname)
	}
	// Instance destruction is pending.
	return nil
}
