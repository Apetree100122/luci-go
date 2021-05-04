// Copyright 2015 The LUCI Authors.
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

package flex

import (
	"context"
	"time"

	gcst "cloud.google.com/go/storage"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"

	"go.chromium.org/luci/logdog/appengine/coordinator"
	"go.chromium.org/luci/logdog/common/storage"
	"go.chromium.org/luci/logdog/common/storage/archive"
	"go.chromium.org/luci/logdog/common/storage/bigtable"
)

const (
	// maxSignedURLLifetime is the maximum allowed signed URL lifetime.
	maxSignedURLLifetime = 1 * time.Hour
)

// Services is a set of support services used by Coordinator endpoints.
//
// Each instance is valid for a single request, but can be re-used throughout
// that request. This is advised, as the Services instance may optionally cache
// values.
//
// Services methods are goroutine-safe.
type Services interface {
	// Storage returns a Storage instance for the supplied log stream.
	//
	// The caller must close the returned instance if successful.
	StorageForStream(ctx context.Context, state *coordinator.LogStreamState, project string) (coordinator.SigningStorage, error)
}

// GlobalServices is an application singleton that stores cross-request service
// structures.
//
// It lives in the root context.
type GlobalServices struct {
	btStorage       *bigtable.Storage
	gsClientFactory func(ctx context.Context, project string) (gs.Client, error)
	storageCache    *StorageCache
}

// NewGlobalServices instantiates a new GlobalServices instance.
//
// Receives the location of the BigTable with intermediate logs.
//
// The Context passed to GlobalServices should be a global server Context not a
// request-specific Context.
func NewGlobalServices(c context.Context, bt *bigtable.Flags) (*GlobalServices, error) {
	// LRU in-memory cache in front of BigTable.
	storageCache := &StorageCache{}

	// Construct the storage, inject the caching implementation into it.
	storage, err := bigtable.StorageFromFlags(c, bt)
	if err != nil {
		return nil, errors.Annotate(err, "failed to connect to BigTable").Err()
	}
	storage.Cache = storageCache

	return &GlobalServices{
		btStorage:    storage,
		storageCache: storageCache,
		gsClientFactory: func(c context.Context, project string) (client gs.Client, e error) {
			// TODO(vadimsh): Switch to AsProject + WithProject(project) once
			// we are ready to roll out project scoped service accounts in Logdog.
			transport, err := auth.GetRPCTransport(c, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
			if err != nil {
				return nil, errors.Annotate(err, "failed to create Google Storage RPC transport").Err()
			}
			prodClient, err := gs.NewProdClient(c, transport)
			if err != nil {
				return nil, errors.Annotate(err, "Failed to create GS client.").Err()
			}
			return prodClient, nil
		},
	}, nil
}

// Storage returns a Storage instance for the supplied log stream.
//
// The caller must close the returned instance if successful.
func (gsvc *GlobalServices) StorageForStream(c context.Context, lst *coordinator.LogStreamState, project string) (
	coordinator.SigningStorage, error) {

	if !lst.ArchivalState().Archived() {
		logging.Debugf(c, "Log is not archived. Fetching from intermediate storage.")
		return noSignedURLStorage{gsvc.btStorage}, nil
	}

	// Some very old logs have malformed data where they claim to be archived but
	// have no archive or index URLs.
	if lst.ArchiveStreamURL == "" {
		logging.Warningf(c, "Log has no archive URL")
		return nil, errors.New("log has no archive URL", grpcutil.NotFoundTag)
	}
	if lst.ArchiveIndexURL == "" {
		logging.Warningf(c, "Log has no index URL")
		return nil, errors.New("log has no index URL", grpcutil.NotFoundTag)
	}

	gsClient, err := gsvc.gsClientFactory(c, project)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to create Google Storage client.")
		return nil, err
	}

	logging.Fields{
		"indexURL":    lst.ArchiveIndexURL,
		"streamURL":   lst.ArchiveStreamURL,
		"archiveTime": lst.ArchivedTime,
	}.Debugf(c, "Log is archived. Fetching from archive storage.")

	st, err := archive.New(archive.Options{
		Index:  gs.Path(lst.ArchiveIndexURL),
		Stream: gs.Path(lst.ArchiveStreamURL),
		Cache:  gsvc.storageCache,
		Client: gsClient,
	})
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to create Google Storage storage instance.")
		return nil, err
	}

	rv := &googleStorage{
		Storage: st,
		svc:     gsvc,
		gs:      gsClient,
		stream:  gs.Path(lst.ArchiveStreamURL),
		index:   gs.Path(lst.ArchiveIndexURL),
	}
	return rv, nil
}

// noSignedURLStorage is a thin wrapper around a Storage instance that cannot
// sign URLs.
type noSignedURLStorage struct {
	storage.Storage
}

func (noSignedURLStorage) GetSignedURLs(context.Context, *coordinator.URLSigningRequest) (
	*coordinator.URLSigningResponse, error) {

	return nil, nil
}

type googleStorage struct {
	// Storage is the base storage.Storage instance.
	storage.Storage
	// svc is the services instance that created this.
	svc *GlobalServices

	// ctx is the Context that was bound at the time of of creation.
	ctx context.Context
	// gs is the backing Google Storage client.
	gs gs.Client

	// stream is the stream's Google Storage URL.
	stream gs.Path
	// index is the index's Google Storage URL.
	index gs.Path

	gsSigningOpts func(context.Context) (*gcst.SignedURLOptions, error)
}

func (si *googleStorage) Close() {
	si.Storage.Close()
	si.gs.Close()
}

func (si *googleStorage) GetSignedURLs(c context.Context, req *coordinator.URLSigningRequest) (*coordinator.URLSigningResponse, error) {
	signer := auth.GetSigner(c)
	info, err := signer.ServiceInfo(c)
	if err != nil {
		return nil, errors.Annotate(err, "").InternalReason("failed to get service info").Err()
	}

	lifetime := req.Lifetime
	switch {
	case lifetime < 0:
		return nil, errors.Reason("invalid signed URL lifetime: %s", lifetime).Err()

	case lifetime > maxSignedURLLifetime:
		lifetime = maxSignedURLLifetime
	}

	// Get our signing options.
	resp := coordinator.URLSigningResponse{
		Expiration: clock.Now(c).Add(lifetime),
	}
	opts := gcst.SignedURLOptions{
		GoogleAccessID: info.ServiceAccountName,
		SignBytes: func(b []byte) ([]byte, error) {
			_, signedBytes, err := signer.SignBytes(c, b)
			return signedBytes, err
		},
		Method:  "GET",
		Expires: resp.Expiration,
	}

	doSign := func(path gs.Path) (string, error) {
		url, err := gcst.SignedURL(path.Bucket(), path.Filename(), &opts)
		if err != nil {
			return "", errors.Annotate(err, "").InternalReason(
				"failed to sign URL: bucket(%s)/filename(%s)", path.Bucket(), path.Filename()).Err()
		}
		return url, nil
	}

	// Sign stream URL.
	if req.Stream {
		if resp.Stream, err = doSign(si.stream); err != nil {
			return nil, errors.Annotate(err, "").InternalReason("failed to sign stream URL").Err()
		}
	}

	// Sign index URL.
	if req.Index {
		if resp.Index, err = doSign(si.index); err != nil {
			return nil, errors.Annotate(err, "").InternalReason("failed to sign index URL").Err()
		}
	}

	return &resp, nil
}
