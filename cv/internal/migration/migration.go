// Copyright 2020 The LUCI Authors.
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

package migration

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"

	cvbqpb "go.chromium.org/luci/cv/api/bigquery/v1"
	migrationpb "go.chromium.org/luci/cv/api/migration"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/gerrit"
	"go.chromium.org/luci/cv/internal/run"
)

// AllowGroup is a Chrome Infra Auth group, members of which are allowed to call
// migration API. It's hardcoded here because this code is temporary.
const AllowGroup = "luci-cv-migration-crbug-1141880"

// RunNotifier abstracts out dependency of MigrationServer on run.Notifier.
type RunNotifier interface {
	NotifyCQDVerificationCompleted(ctx context.Context, runID common.RunID) error
}

// MigrationServer implements CQDaemon -> CV migration API.
type MigrationServer struct {
	RunNotifier RunNotifier

	migrationpb.UnimplementedMigrationServer
}

// ReportRuns notifies CV of the Runs CQDaemon is currently working with.
//
// Used to determine whether CV's view of the world matches that of CQDaemon.
// Initially, this is just FYI for CV.
func (m *MigrationServer) ReportRuns(ctx context.Context, req *migrationpb.ReportRunsRequest) (resp *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}

	project := "<UNKNOWN>"
	if i := auth.CurrentIdentity(ctx); i.Kind() == identity.Project {
		project = i.Value()
	}

	cls := 0
	for _, r := range req.Runs {
		cls += len(r.Attempt.GerritChanges)
	}
	logging.Infof(ctx, "CQD[%s] is working on %d attempts %d CLs right now", project, len(req.Runs), cls)
	return &emptypb.Empty{}, nil
}

// ReportVerifiedRun notifies CV of the Run CQDaemon has just finished
// verifying.
//
// Only called iff run was given to CQDaemon by CV via FetchActiveRuns.
func (m *MigrationServer) ReportVerifiedRun(ctx context.Context, req *migrationpb.ReportVerifiedRunRequest) (resp *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}

	k := req.GetRun().GetAttempt().GetKey()
	if k == "" {
		return nil, appstatus.Error(codes.InvalidArgument, "attempt key is required")
	}

	optionalID := common.RunID(req.GetRun().GetId())
	logging.Debugf(ctx, "ReportVerifiedRun(Run %q | Attempt %q)", optionalID, k)
	r, err := fetchRun(ctx, optionalID, k)
	switch {
	case err != nil:
		return nil, err
	case r == nil:
		logging.Errorf(ctx, "BUG: ReportVerifiedRun(Run %q | Attempt %q) for a Run not known to CV", optionalID, k)
		return nil, appstatus.Errorf(codes.NotFound, "Run %q does not exist", optionalID)
	case optionalID == "":
		// Set the missing Run ID.
		req.GetRun().Id = string(r.ID)
	}

	err = saveVerifiedCQDRun(ctx, req, func(ctx context.Context) error {
		return m.RunNotifier.NotifyCQDVerificationCompleted(ctx, r.ID)
	})
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ReportFinishedRun is used by CQD to report runs it handled to completion.
//
// It'll removed upon hitting Milestone 1.
func (m *MigrationServer) ReportFinishedRun(ctx context.Context, req *migrationpb.ReportFinishedRunRequest) (resp *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}
	k := req.GetRun().GetAttempt().GetKey()
	if k == "" {
		return nil, appstatus.Error(codes.InvalidArgument, "attempt key is required")
	}

	optionalID := common.RunID(req.GetRun().GetId())
	logging.Debugf(ctx, "ReportFinishedRun(Run %q | Attempt %q)", optionalID, k)
	r, err := fetchRun(ctx, optionalID, k)
	switch {
	case err != nil:
		return nil, err
	case r == nil && optionalID != "":
		return nil, appstatus.Errorf(codes.NotFound, "Run %q does not exist", optionalID)
	case r == nil:
		logging.Warningf(ctx, "No matching Run, saving FinishedCQDRun(attempt key %q) anyway", k)
	case optionalID == "":
		// Set the missing Run ID.
		req.GetRun().Id = string(r.ID)
	}

	if err = saveFinishedCQDRun(ctx, req.GetRun()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (m *MigrationServer) ReportUsedNetrc(ctx context.Context, req *migrationpb.ReportUsedNetrcRequest) (resp *emptypb.Empty, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}
	if req.AccessToken == "" || req.GerritHost == "" {
		return nil, appstatus.Error(codes.InvalidArgument, "access_token and gerrit_host required")
	}

	project := "<UNKNOWN>"
	if i := auth.CurrentIdentity(ctx); i.Kind() == identity.Project {
		project = i.Value()
	}
	logging.Infof(ctx, "CQD[%s] uses netrc access token for %s", project, req.GerritHost)
	if err = gerrit.SaveLegacyNetrcToken(ctx, req.GerritHost, req.AccessToken); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (m *MigrationServer) PostGerritMessage(ctx context.Context, req *migrationpb.PostGerritMessageRequest) (resp *migrationpb.PostGerritMessageResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}
	logging.Debugf(ctx, "%s calls PostGerritMessage\n%s", auth.CurrentIdentity(ctx), req)

	clExternalID, err := changelist.GobID(req.GetHost(), req.GetChange())
	switch {
	case req.GetHost() == "" || req.GetChange() <= 0 || req.GetRevision() == "":
		return nil, appstatus.Error(codes.InvalidArgument, "host, change and revision are required")
	case err != nil:
		return nil, appstatus.Errorf(codes.InvalidArgument, "host/change are invalid: %s", err)

	case req.GetProject() == "":
		return nil, appstatus.Error(codes.InvalidArgument, "project is required")
	case req.GetAttemptKey() == "":
		return nil, appstatus.Error(codes.InvalidArgument, "attempt_key is required")
	case req.GetComment() == "":
		return nil, appstatus.Error(codes.InvalidArgument, "comment is required")
	}

	// Load Run & CL in parallel.
	// The only downside is that if both fail, there is a race between which one
	// will be returned the user, but for an internal API this is fine.
	var r *run.Run
	var cl *changelist.CL
	eg, eCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		r, err = fetchRun(eCtx, common.RunID(req.GetRunId()), req.GetAttemptKey())
		return err
	})
	eg.Go(func() error {
		var err error
		cl, err = clExternalID.Get(ctx)
		if err == datastore.ErrNoSuchEntity {
			cl = nil
			err = nil
		}
		return err
	})
	if err = eg.Wait(); err != nil {
		return nil, err
	}

	switch {
	// The first two checks should trigger only rarely: when CV is behind CQD from
	// perspective of Gerrit's data (e.g. CV uses a stale Gerrit replica).
	// These should to resolve themselves as CV catches up.
	case r == nil:
		return nil, appstatus.Errorf(codes.Unavailable, "Run %q | Attempt %q doesn't exist", req.GetRunId(), req.GetAttemptKey())
	case cl == nil:
		return nil, appstatus.Error(codes.Unavailable, clExternalID.MustURL()+" is not yet known to CV")

	// These checks are just early detection iff CQD and CV diverge.
	case r.ID.LUCIProject() != req.GetProject():
		return nil, appstatus.Errorf(codes.FailedPrecondition, "Run %q doesn't match expected LUCI project %q", r.ID, req.GetProject())
	case run.IsEnded(r.Status):
		return nil, appstatus.Errorf(codes.FailedPrecondition, "Run %q is already finished (%s)", r.ID, r.Status)
	}

	ci := cl.Snapshot.GetGerrit().GetInfo()
	msg := strings.TrimSpace(req.GetComment())
	truncatedMsg := gerrit.TruncateMessage(msg)

	for _, m := range ci.GetMessages() {
		switch {
		case m.GetDate().AsTime().Before(r.CreateTime):
			// Message posted before this Run.
			continue
		case strings.Contains(m.GetMessage(), msg) || strings.Contains(m.GetMessage(), truncatedMsg):
			// Message has already been posted in the context of this Run.
			logging.Infof(ctx, "message was already posted at %s", m.GetDate().AsTime())
			return &migrationpb.PostGerritMessageResponse{}, nil
		}
	}
	gc, err := gerrit.CurrentClient(ctx, req.GetHost(), req.GetProject())
	if err != nil {
		return nil, appstatus.Errorf(codes.Internal, "failed to obtain Gerrit Client: %s", err)
	}

	_, err = gc.SetReview(ctx, makeGerritSetReviewRequest(r, ci, truncatedMsg, req.GetRevision(), req.GetSendEmail()))
	switch code := grpcutil.Code(err); code {
	case codes.OK:
		return &migrationpb.PostGerritMessageResponse{}, nil
	case codes.PermissionDenied, codes.NotFound, codes.FailedPrecondition:
		// Propagate the same gRPC error code.
		return nil, errors.Annotate(err, "failed to SetReview").Err()
	default:
		// Propagate the same gRPC error code, but also record this unexpected
		// response.
		return nil, gerrit.UnhandledError(ctx, err, "failed to SetReview")
	}
}

// FetchActiveRuns returns all RUNNING runs for the given LUCI Project.
func (m *MigrationServer) FetchActiveRuns(ctx context.Context, req *migrationpb.FetchActiveRunsRequest) (resp *migrationpb.FetchActiveRunsResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}
	if req.GetLuciProject() == "" {
		return nil, appstatus.Error(codes.InvalidArgument, "luci_project is required")
	}

	resp = &migrationpb.FetchActiveRunsResponse{}
	if resp.ActiveRuns, err = fetchActiveRuns(ctx, req.GetLuciProject()); err != nil {
		return nil, err
	}
	return resp, nil
}

// FetchExcludedCLs returns all CLs referenced by VerifiedCQDRun entities
// corresponding to not yet finished Runs.
func (m *MigrationServer) FetchExcludedCLs(ctx context.Context, req *migrationpb.FetchExcludedCLsRequest) (resp *migrationpb.FetchExcludedCLsResponse, err error) {
	defer func() { err = grpcutil.GRPCifyAndLogErr(ctx, err) }()
	if err = m.checkAllowed(ctx); err != nil {
		return nil, err
	}
	if req.GetLuciProject() == "" {
		return nil, appstatus.Error(codes.InvalidArgument, "luci_project is required")
	}

	resp = &migrationpb.FetchExcludedCLsResponse{}
	resp.Cls, err = fetchExcludedCLs(ctx, req.GetLuciProject())
	return resp, err
}

func (m *MigrationServer) checkAllowed(ctx context.Context) error {
	i := auth.CurrentIdentity(ctx)
	if i.Kind() == identity.Project {
		// Only small list of LUCI services is allowed,
		// we can assume no malicious access, hence this is CQDaemon.
		return nil
	}
	logging.Warningf(ctx, "Unusual caller %s", i)

	switch yes, err := auth.IsMember(ctx, AllowGroup); {
	case err != nil:
		return status.Errorf(codes.Internal, "failed to check ACL")
	case !yes:
		return status.Errorf(codes.PermissionDenied, "not a member of %s", AllowGroup)
	default:
		return nil
	}
}

// clsOf emits CL of the Attempt (aka Run) preserving the order but avoiding
// duplicating hostnames.
func clsOf(a *cvbqpb.Attempt) string {
	if len(a.GerritChanges) == 0 {
		return "NO CLS"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d CLs:", len(a.GerritChanges))
	priorIdx := 0
	emit := func(excluding int) {
		fmt.Fprintf(&buf, " [%s", a.GerritChanges[priorIdx].Host)
		for i := priorIdx; i < excluding; i++ {
			cl := a.GerritChanges[i]
			fmt.Fprintf(&buf, " %d/%d", cl.Change, cl.Patchset)
		}
		buf.WriteString("]")
		priorIdx = excluding
	}
	for j, cl := range a.GerritChanges {
		if a.GerritChanges[priorIdx].Host != cl.Host {
			emit(j)
		}
	}
	emit(len(a.GerritChanges))
	return buf.String()
}
