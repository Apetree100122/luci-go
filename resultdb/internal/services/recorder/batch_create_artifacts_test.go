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

package recorder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	repb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"go.chromium.org/luci/resultdb/internal/artifacts"
	"go.chromium.org/luci/resultdb/internal/invocations"
	"go.chromium.org/luci/resultdb/internal/spanutil"
	"go.chromium.org/luci/resultdb/internal/testutil"
	"go.chromium.org/luci/resultdb/internal/testutil/insert"
	pb "go.chromium.org/luci/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

// fakeRBEClient mocks BatchUpdateBlobs.
type fakeRBEClient struct {
	repb.ContentAddressableStorageClient
	req  *repb.BatchUpdateBlobsRequest
	resp *repb.BatchUpdateBlobsResponse
	err  error
}

func (c *fakeRBEClient) BatchUpdateBlobs(ctx context.Context, in *repb.BatchUpdateBlobsRequest, opts ...grpc.CallOption) (*repb.BatchUpdateBlobsResponse, error) {
	c.req = in
	return c.resp, c.err
}

func (c *fakeRBEClient) mockResp(err error, cds ...codes.Code) {
	c.err = err
	c.resp = &repb.BatchUpdateBlobsResponse{}
	for _, cd := range cds {
		c.resp.Responses = append(c.resp.Responses, &repb.BatchUpdateBlobsResponse_Response{
			Status: &spb.Status{Code: int32(cd)},
		})
	}
}

func TestNewArtifactCreationRequestsFromProto(t *testing.T) {
	newArtReq := func(parent, artID, contentType string) *pb.CreateArtifactRequest {
		return &pb.CreateArtifactRequest{
			Parent:   parent,
			Artifact: &pb.Artifact{ArtifactId: artID, ContentType: contentType},
		}
	}

	Convey("newArtifactCreationRequestsFromProto", t, func() {
		bReq := &pb.BatchCreateArtifactsRequest{}
		invArt := newArtReq("invocations/inv1", "art1", "text/html")
		trArt := newArtReq("invocations/inv1/tests/t1/results/r1", "art2", "image/png")

		Convey("successes", func() {
			bReq.Requests = append(bReq.Requests, invArt)
			bReq.Requests = append(bReq.Requests, trArt)
			invID, arts, err := parseBatchCreateArtifactsRequest(bReq)
			So(err, ShouldBeNil)
			So(invID, ShouldEqual, invocations.ID("inv1"))
			So(len(arts), ShouldEqual, len(bReq.Requests))

			// invocation-level artifact
			So(arts[0].artifactID, ShouldEqual, "art1")
			So(arts[0].parentID(), ShouldEqual, artifacts.ParentID("", ""))
			So(arts[0].contentType, ShouldEqual, "text/html")

			// test-result-level artifact
			So(arts[1].artifactID, ShouldEqual, "art2")
			So(arts[1].parentID(), ShouldEqual, artifacts.ParentID("t1", "r1"))
			So(arts[1].contentType, ShouldEqual, "image/png")
		})

		Convey("ignores size_bytes", func() {
			bReq.Requests = append(bReq.Requests, trArt)
			trArt.Artifact.SizeBytes = 123
			trArt.Artifact.Contents = make([]byte, 10249)
			_, arts, err := parseBatchCreateArtifactsRequest(bReq)
			So(err, ShouldBeNil)
			So(arts[0].size, ShouldEqual, 10249)
		})

		Convey("sum() of artifact.Contents is too big", func() {
			for i := 0; i < 11; i++ {
				req := newArtReq("invocations/inv1", fmt.Sprintf("art%d", i), "text/html")
				req.Artifact.Contents = make([]byte, 1024*1024)
				bReq.Requests = append(bReq.Requests, req)
			}
			_, _, err := parseBatchCreateArtifactsRequest(bReq)
			So(err, ShouldErrLike, "the total size of artifact contents exceeded")
		})

		Convey("if more than one invocations", func() {
			bReq.Requests = append(bReq.Requests, newArtReq("invocations/inv1", "art1", "text/html"))
			bReq.Requests = append(bReq.Requests, newArtReq("invocations/inv2", "art1", "text/html"))
			_, _, err := parseBatchCreateArtifactsRequest(bReq)
			So(err, ShouldErrLike, `only one invocation is allowed: "inv1", "inv2"`)
		})
	})
}

func TestBatchCreateArtifacts(t *testing.T) {
	Convey("TestBatchCreateArtifacts", t, func() {
		ctx := testutil.SpannerTestContext(t)
		token, err := generateInvocationToken(ctx, "inv")
		So(err, ShouldBeNil)
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(UpdateTokenMetadataKey, token))

		casClient := &fakeRBEClient{}
		recorder := newTestRecorderServer()
		recorder.casClient = casClient
		bReq := &pb.BatchCreateArtifactsRequest{}

		appendArtReq := func(aID, content, cType string) {
			bReq.Requests = append(bReq.Requests, &pb.CreateArtifactRequest{
				Parent: "invocations/inv",
				Artifact: &pb.Artifact{
					ArtifactId: aID, Contents: []byte(content), ContentType: cType,
				},
			})
		}
		fetchState := func(aID string) (size int64, hash string, contentType string) {
			testutil.MustReadRow(
				ctx, "Artifacts", invocations.ID("inv").Key("", aID),
				map[string]interface{}{
					"Size":        &size,
					"RBECASHash":  &hash,
					"ContentType": &contentType,
				},
			)
			return
		}
		compHash := func(content string) string {
			h := sha256.Sum256([]byte(content))
			return fmt.Sprintf("sha256:%s", hex.EncodeToString(h[:]))
		}

		Convey("works", func() {
			testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))
			appendArtReq("art1", "c0ntent", "text/plain")
			appendArtReq("art2", "c1ntent", "text/richtext")
			casClient.mockResp(nil, codes.OK, codes.OK)

			resp, err := recorder.BatchCreateArtifacts(ctx, bReq)
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &pb.BatchCreateArtifactsResponse{
				Artifacts: []*pb.Artifact{
					{
						Name:        "invocations/inv/artifacts/art1",
						ArtifactId:  "art1",
						ContentType: "text/plain",
						SizeBytes:   7,
					},
					{
						Name:        "invocations/inv/artifacts/art2",
						ArtifactId:  "art2",
						ContentType: "text/richtext",
						SizeBytes:   7,
					},
				},
			})
			// verify the RBECAS reqs
			So(casClient.req, ShouldResemble, &repb.BatchUpdateBlobsRequest{
				InstanceName: "",
				Requests: []*repb.BatchUpdateBlobsRequest_Request{
					{
						Digest: &repb.Digest{
							Hash:      compHash("c0ntent"),
							SizeBytes: int64(len("c0ntent")),
						},
						Data: []byte("c0ntent"),
					},
					{
						Digest: &repb.Digest{
							Hash:      compHash("c1ntent"),
							SizeBytes: int64(len("c1ntent")),
						},
						Data: []byte("c1ntent"),
					},
				},
			})
			// verify the Spanner states
			size, hash, cType := fetchState("art1")
			So(size, ShouldEqual, int64(len("c0ntent")))
			So(hash, ShouldEqual, compHash("c0ntent"))
			So(cType, ShouldEqual, "text/plain")

			size, hash, cType = fetchState("art2")
			So(size, ShouldEqual, int64(len("c1ntent")))
			So(hash, ShouldEqual, compHash("c1ntent"))
			So(cType, ShouldEqual, "text/richtext")
		})

		Convey("BatchUpdateBlobs fails", func() {
			testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))
			appendArtReq("art1", "c0ntent", "text/plain")
			appendArtReq("art2", "c1ntent", "text/richtext")

			Convey("Partly", func() {
				casClient.mockResp(nil, codes.OK, codes.InvalidArgument)
				_, err := recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldErrLike, `artifact "invocations/inv/artifacts/art2": cas.BatchUpdateBlobs failed`)
			})

			Convey("Entirely", func() {
				// exceeded the maximum size limit is the only possible error that
				// can cause the entire request failed.
				casClient.mockResp(errors.New("err"), codes.OK, codes.OK)
				_, err := recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldErrLike, "cas.BatchUpdateBlobs failed")
			})
		})

		Convey("Token", func() {
			appendArtReq("art1", "", "text/plain")
			testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))

			Convey("Missing", func() {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs())
				_, err = recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldHaveAppStatus, codes.Unauthenticated, `missing update-token`)
			})
			Convey("Wrong", func() {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(UpdateTokenMetadataKey, "rong"))
				_, err = recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldHaveAppStatus, codes.PermissionDenied, `invalid update token`)
			})
		})

		Convey("Verify state", func() {
			appendArtReq("art1", "c0ntent", "text/plain")
			casClient.mockResp(nil, codes.OK, codes.OK)

			Convey("Finalized invocation", func() {
				testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_FINALIZED, nil))
				_, err = recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldHaveAppStatus, codes.FailedPrecondition, `invocations/inv is not active`)
			})

			art := map[string]interface{}{
				"InvocationId": invocations.ID("inv"),
				"ParentId":     "",
				"ArtifactId":   "art1",
				"RBECASHash":   compHash("c0ntent"),
				"Size":         len("c0ntent"),
				"ContentType":  "text/plain",
			}

			Convey("Same artifact exists", func() {
				testutil.MustApply(ctx,
					insert.Invocation("inv", pb.Invocation_ACTIVE, nil),
					spanutil.InsertMap("Artifacts", art),
				)
				resp, err := recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldBeNil)
				So(resp, ShouldResemble, (*pb.BatchCreateArtifactsResponse)(nil))
			})

			Convey("Different artifact exists", func() {
				testutil.MustApply(ctx,
					insert.Invocation("inv", pb.Invocation_ACTIVE, nil),
					spanutil.InsertMap("Artifacts", art),
				)

				bReq.Requests[0].Artifact.Contents = []byte("loooong content")
				_, err := recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldHaveAppStatus, codes.AlreadyExists, "exists w/ different size")

				bReq.Requests[0].Artifact.Contents = []byte("c1ntent")
				_, err = recorder.BatchCreateArtifacts(ctx, bReq)
				So(err, ShouldHaveAppStatus, codes.AlreadyExists, "exists w/ different hash")
			})
		})
	})
}
