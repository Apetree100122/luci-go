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

package admin

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"

	adminpb "go.chromium.org/luci/cv/internal/admin/api"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/eventbox"
	"go.chromium.org/luci/cv/internal/gerrit/poller"
	"go.chromium.org/luci/cv/internal/gerrit/updater"
	"go.chromium.org/luci/cv/internal/prjmanager"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
	"go.chromium.org/luci/cv/internal/run"
	"go.chromium.org/luci/cv/internal/run/eventpb"
)

// allowGroup is a Chrome Infra Auth group, members of which are allowed to call
// admin API. See https://crbug.com/1183616.
const allowGroup = "service-luci-change-verifier-admins"

type AdminServer struct {
	GerritUpdater *updater.Updater
	PMNotifier    *prjmanager.Notifier
	RunNotifier   *run.Notifier

	adminpb.UnimplementedAdminServer
}

func (d *AdminServer) GetProject(ctx context.Context, req *adminpb.GetProjectRequest) (resp *adminpb.GetProjectResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "GetProject"); err != nil {
		return
	}
	if req.GetProject() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "project is required")
	}

	eg, ctx := errgroup.WithContext(ctx)

	var p *prjmanager.Project
	eg.Go(func() (err error) {
		p, err = prjmanager.Load(ctx, req.GetProject())
		return
	})

	resp = &adminpb.GetProjectResponse{}
	eg.Go(func() error {
		list, err := eventbox.List(ctx, datastore.MakeKey(ctx, prjmanager.ProjectKind, req.GetProject()))
		if err != nil {
			return err
		}
		events := make([]*prjpb.Event, len(list))
		for i, item := range list {
			events[i] = &prjpb.Event{}
			if err = proto.Unmarshal(item.Value, events[i]); err != nil {
				return errors.Annotate(err, "failed to unmarshal Event %q", item.ID).Err()
			}
		}
		resp.Events = events
		return nil
	})

	switch err = eg.Wait(); {
	case err != nil:
		return nil, err
	case p == nil:
		return nil, status.Errorf(codes.NotFound, "project not found")
	default:
		resp.State = p.State
		resp.State.LuciProject = req.GetProject()
		return resp, nil
	}
}

func (d *AdminServer) GetRun(ctx context.Context, req *adminpb.GetRunRequest) (resp *adminpb.GetRunResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "GetRun"); err != nil {
		return
	}
	if req.GetRun() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "run ID is required")
	}

	eg, ctx := errgroup.WithContext(ctx)

	r := &run.Run{ID: common.RunID(req.GetRun())}
	eg.Go(func() error {
		switch err := datastore.Get(ctx, r); {
		case err == datastore.ErrNoSuchEntity:
			return status.Errorf(codes.NotFound, "run not found")
		case err != nil:
			return status.Errorf(codes.Internal, "failed to fetch Run")
		default:
			return nil
		}
	})

	var events []*eventpb.Event
	eg.Go(func() error {
		list, err := eventbox.List(ctx, datastore.MakeKey(ctx, run.RunKind, req.GetRun()))
		if err != nil {
			return err
		}
		events = make([]*eventpb.Event, len(list))
		for i, item := range list {
			events[i] = &eventpb.Event{}
			if err = proto.Unmarshal(item.Value, events[i]); err != nil {
				return errors.Annotate(err, "failed to unmarshal Event %q", item.ID).Err()
			}
		}
		return nil
	})

	if err = eg.Wait(); err != nil {
		return nil, err
	}

	resp = &adminpb.GetRunResponse{
		Id:            req.GetRun(),
		Eversion:      int64(r.EVersion),
		Mode:          string(r.Mode),
		Status:        r.Status,
		CreateTime:    timestamppb.New(r.CreateTime),
		StartTime:     timestamppb.New(r.StartTime),
		UpdateTime:    timestamppb.New(r.UpdateTime),
		EndTime:       timestamppb.New(r.EndTime),
		Owner:         string(r.Owner),
		ConfigGroupId: string(r.ConfigGroupID),
		Cls:           common.CLIDsAsInt64s(r.CLs),
		Submission:    r.Submission,

		Events: events,
	}
	return resp, nil
}

func (d *AdminServer) GetCL(ctx context.Context, req *adminpb.GetCLRequest) (resp *adminpb.GetCLResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "GetCL"); err != nil {
		return
	}

	var cl *changelist.CL
	var eid changelist.ExternalID
	switch {
	case req.GetId() != 0:
		cl = &changelist.CL{ID: common.CLID(req.GetId())}
		err = datastore.Get(ctx, cl)
	case req.GetExternalId() != "":
		eid = changelist.ExternalID(req.GetExternalId())
		cl, err = eid.Get(ctx)
	case req.GetGerritUrl() != "":
		eid, err = parseGerritURL(req.GetExternalId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid Gerrit URL %q: %s", req.GetGerritUrl(), err)
		}
		cl, err = eid.Get(ctx)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "id or external_id or gerrit_url is required")
	}

	switch {
	case err == datastore.ErrNoSuchEntity:
		if req.GetId() == 0 {
			return nil, status.Errorf(codes.NotFound, "CL %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.NotFound, "CL %s not found", eid)
	case err != nil:
		return nil, err
	}
	runs := make([]string, len(cl.IncompleteRuns))
	for i, id := range cl.IncompleteRuns {
		runs[i] = string(id)
	}
	resp = &adminpb.GetCLResponse{
		Id:               int64(cl.ID),
		Eversion:         int64(cl.EVersion),
		ExternalId:       string(cl.ExternalID),
		Snapshot:         cl.Snapshot,
		ApplicableConfig: cl.ApplicableConfig,
		DependentMeta:    cl.DependentMeta,
		IncompleteRuns:   runs,
	}
	return resp, nil
}

func (d *AdminServer) GetPoller(ctx context.Context, req *adminpb.GetPollerRequest) (resp *adminpb.GetPollerResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "GetPoller"); err != nil {
		return
	}
	if req.GetProject() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "project is required")
	}

	s := poller.State{LuciProject: req.GetProject()}
	switch err := datastore.Get(ctx, &s); {
	case err == datastore.ErrNoSuchEntity:
		return nil, status.Errorf(codes.NotFound, "poller not found")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to fetch poller state")
	}
	resp = &adminpb.GetPollerResponse{
		Project:    s.LuciProject,
		Eversion:   s.EVersion,
		ConfigHash: s.ConfigHash,
		UpdateTime: timestamppb.New(s.UpdateTime),
		Subpollers: s.SubPollers,
	}
	return resp, nil
}

// Copy from dsset.
type itemEntity struct {
	_kind string `gae:"$kind,dsset.Item"`

	ID     string         `gae:"$id"`
	Parent *datastore.Key `gae:"$parent"`
	Value  []byte         `gae:",noindex"`
}

func (d *AdminServer) DeleteProjectEvents(ctx context.Context, req *adminpb.DeleteProjectEventsRequest) (resp *adminpb.DeleteProjectEventsResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "DeleteProjectEvents"); err != nil {
		return
	}

	switch {
	case req.GetProject() == "":
		return nil, status.Errorf(codes.InvalidArgument, "project is required")
	case req.GetLimit() <= 0:
		return nil, status.Errorf(codes.InvalidArgument, "limit must be >0")
	}

	parent := datastore.MakeKey(ctx, prjmanager.ProjectKind, req.GetProject())
	q := datastore.NewQuery("dsset.Item").Ancestor(parent).Limit(req.GetLimit())
	var entities []*itemEntity
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, errors.Annotate(err, "failed to fetch up to %d events", req.GetLimit()).Tag(transient.Tag).Err()
	}

	stats := make(map[string]int64, 10)
	for _, e := range entities {
		pb := &prjpb.Event{}
		if err := proto.Unmarshal(e.Value, pb); err != nil {
			stats["<unknown>"]++
		} else {
			stats[fmt.Sprintf("%T", pb.GetEvent())]++
		}
	}
	if err := datastore.Delete(ctx, entities); err != nil {
		return nil, errors.Annotate(err, "failed to delete %d events", len(entities)).Tag(transient.Tag).Err()
	}
	return &adminpb.DeleteProjectEventsResponse{Events: stats}, nil
}

func (d *AdminServer) RefreshProjectCLs(ctx context.Context, req *adminpb.RefreshProjectCLsRequest) (resp *adminpb.RefreshProjectCLsResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "RefreshProjectCLs"); err != nil {
		return
	}
	if req.GetProject() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "project is required")
	}

	p, err := prjmanager.Load(ctx, req.GetProject())
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch Project %q", req.GetProject()).Tag(transient.Tag).Err()
	}

	cls := make([]*changelist.CL, len(p.State.GetPcls()))
	errs := parallel.WorkPool(20, func(work chan<- func() error) {
		for i, pcl := range p.State.GetPcls() {
			i := i
			id := pcl.GetClid()
			work <- func() error {
				// Load individual CL to avoid OOMs.
				cl := changelist.CL{ID: common.CLID(id)}
				if err := datastore.Get(ctx, &cl); err != nil {
					return errors.Annotate(err, "failed to fetch CL %d", id).Tag(transient.Tag).Err()
				}
				cls[i] = &changelist.CL{ID: cl.ID, EVersion: cl.EVersion}

				host, change, err := cl.ExternalID.ParseGobID()
				if err != nil {
					return err
				}
				payload := &updater.RefreshGerritCL{
					LuciProject: req.GetProject(),
					Host:        host,
					Change:      change,
					ClidHint:    id,
				}
				return d.GerritUpdater.Schedule(ctx, payload)
			}
		}
	})
	if err := common.MostSevereError(errs); err != nil {
		return nil, err
	}

	if err := d.PMNotifier.NotifyCLsUpdated(ctx, req.GetProject(), cls); err != nil {
		return nil, err
	}

	clvs := make(map[int64]int64, len(p.State.GetPcls()))
	for _, cl := range cls {
		clvs[int64(cl.ID)] = int64(cl.EVersion)
	}
	return &adminpb.RefreshProjectCLsResponse{ClVersions: clvs}, nil
}

func (d *AdminServer) SendProjectEvent(ctx context.Context, req *adminpb.SendProjectEventRequest) (_ *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "SendProjectEvent"); err != nil {
		return
	}
	switch {
	case req.GetProject() == "":
		return nil, status.Errorf(codes.InvalidArgument, "project is required")
	case req.GetEvent().GetEvent() == nil:
		return nil, status.Errorf(codes.InvalidArgument, "event with a specific inner event is required")
	}

	switch p, err := prjmanager.Load(ctx, req.GetProject()); {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to fetch Project")
	case p == nil:
		return nil, status.Errorf(codes.NotFound, "project not found")
	}

	if err := d.PMNotifier.TaskRefs.SendNow(ctx, req.GetProject(), req.GetEvent()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send event: %s", err)
	}
	return &emptypb.Empty{}, nil
}

func (d *AdminServer) SendRunEvent(ctx context.Context, req *adminpb.SendRunEventRequest) (_ *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = checkAllowed(ctx, "SendRunEvent"); err != nil {
		return
	}
	switch {
	case req.GetRun() == "":
		return nil, status.Errorf(codes.InvalidArgument, "Run is required")
	case req.GetEvent().GetEvent() == nil:
		return nil, status.Errorf(codes.InvalidArgument, "event with a specific inner event is required")
	}

	switch err := datastore.Get(ctx, &run.Run{ID: common.RunID(req.GetRun())}); {
	case err == datastore.ErrNoSuchEntity:
		return nil, status.Errorf(codes.NotFound, "Run not found")
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to fetch Run")
	}

	if err := d.RunNotifier.TaskRefs.SendNow(ctx, common.RunID(req.GetRun()), req.GetEvent()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send event: %s", err)
	}
	return &emptypb.Empty{}, nil
}

func checkAllowed(ctx context.Context, name string) error {
	switch yes, err := auth.IsMember(ctx, allowGroup); {
	case err != nil:
		return status.Errorf(codes.Internal, "failed to check ACL")
	case !yes:
		return status.Errorf(codes.PermissionDenied, "not a member of %s", allowGroup)
	default:
		logging.Warningf(ctx, "%s is calling admin.%s", auth.CurrentIdentity(ctx), name)
		return nil
	}
}

var regexCrRevPath = regexp.MustCompile(`/([ci])/(\d+)(/(\d+))?`)
var regexGoB = regexp.MustCompile(`((\w+-)+review\.googlesource\.com)/(#/)?(c/)?(([^\+]+)/\+/)?(\d+)(/(\d+)?)?`)

func parseGerritURL(s string) (changelist.ExternalID, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	var host string
	var change int64
	if u.Host == "crrev.com" {
		m := regexCrRevPath.FindStringSubmatch(u.Path)
		if m == nil {
			return "", errors.New("invalid crrev.com URL")
		}
		switch m[1] {
		case "c":
			host = "chromium-review.googlesource.com"
		case "i":
			host = "chrome-internal-review.googlesource.com"
		default:
			panic("impossible")
		}
		if change, err = strconv.ParseInt(m[2], 10, 64); err != nil {
			return "", errors.Reason("invalid crrev.com URL change number /%s/", m[2]).Err()
		}
	} else {
		m := regexGoB.FindStringSubmatch(s)
		if m == nil {
			return "", errors.Reason("Gerrit URL didn't match regexp %q", regexGoB.String()).Err()
		}
		if host = m[1]; host == "" {
			return "", errors.New("invalid Gerrit host")
		}
		if change, err = strconv.ParseInt(m[7], 10, 64); err != nil {
			return "", errors.Reason("invalid Gerrit URL change number /%s/", m[7]).Err()
		}
	}
	return changelist.GobID(host, change)
}
