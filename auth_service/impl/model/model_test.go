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

package model

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/gae/filter/txndefer"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/service/protocol"
	"go.chromium.org/luci/server/tq"
	"go.chromium.org/luci/server/tq/tqtesting"

	"go.chromium.org/luci/auth_service/api/configspb"
	"go.chromium.org/luci/auth_service/api/taskspb"
	"go.chromium.org/luci/auth_service/impl/info"
	"go.chromium.org/luci/auth_service/internal/permissions"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

var (
	testCreatedTS  = time.Date(2020, time.August, 16, 15, 20, 0, 0, time.UTC)
	testModifiedTS = time.Date(2021, time.August, 16, 12, 20, 0, 0, time.UTC)
)

func testAuthVersionedEntityMixin() AuthVersionedEntityMixin {
	return AuthVersionedEntityMixin{
		ModifiedTS:    testModifiedTS,
		ModifiedBy:    "user:test-modifier@example.com",
		AuthDBRev:     1337,
		AuthDBPrevRev: 1336,
	}
}

func testSecurityConfig() *protocol.SecurityConfig {
	return &protocol.SecurityConfig{
		InternalServiceRegexp: []string{`.*\.example.com`},
	}
}

func testSecurityConfigBlob() []byte {
	blob, err := proto.Marshal(testSecurityConfig())
	if err != nil {
		panic(err)
	}
	return blob
}

func testAuthGlobalConfig(ctx context.Context) *AuthGlobalConfig {
	return &AuthGlobalConfig{
		Kind:                     "AuthGlobalConfig",
		ID:                       "root",
		AuthVersionedEntityMixin: testAuthVersionedEntityMixin(),
		OAuthClientID:            "test-client-id",
		OAuthAdditionalClientIDs: []string{
			"additional-client-id-0",
			"additional-client-id-1",
		},
		OAuthClientSecret: "test-client-secret",
		TokenServerURL:    "https://token-server.example.com",
		SecurityConfig:    testSecurityConfigBlob(),
	}
}

func testAuthReplicationState(ctx context.Context, rev int64) *AuthReplicationState {
	return &AuthReplicationState{
		Kind:       "AuthReplicationState",
		ID:         "self",
		Parent:     RootKey(ctx),
		AuthDBRev:  rev,
		ModifiedTS: testModifiedTS,
		PrimaryID:  "test-primary-id",
	}
}

func testAuthGroup(ctx context.Context, name string) *AuthGroup {
	members := []string{
		fmt.Sprintf("user:%s-m1@example.com", name),
		fmt.Sprintf("user:%s-m2@example.com", name),
	}
	return &AuthGroup{
		Kind:                     "AuthGroup",
		ID:                       name,
		Parent:                   RootKey(ctx),
		AuthVersionedEntityMixin: testAuthVersionedEntityMixin(),
		Members:                  members,
		Globs:                    []string{"user:*@example.com"},
		Nested:                   []string{"nested-" + name},
		Description:              fmt.Sprintf("This is a test auth group %q.", name),
		Owners:                   "owners-" + name,
		CreatedTS:                testCreatedTS,
		CreatedBy:                "user:test-creator@example.com",
	}
}

func testExternalAuthGroup(ctx context.Context, name string, members []string) *AuthGroup {
	return &AuthGroup{
		Kind:   "AuthGroup",
		ID:     name,
		Parent: RootKey(ctx),
		AuthVersionedEntityMixin: AuthVersionedEntityMixin{
			ModifiedTS:    testModifiedTS,
			ModifiedBy:    "user:test-modifier@example.com",
			AuthDBRev:     1,
			AuthDBPrevRev: 0,
		},
		Members:   members,
		Owners:    AdminGroup,
		CreatedTS: testCreatedTS,
		CreatedBy: "user:test-creator@example.com",
	}
}

// emptyAuthGroup creates a new AuthGroup, that owns itself, with no members.
func emptyAuthGroup(ctx context.Context, name string) *AuthGroup {
	return &AuthGroup{
		Kind:   "AuthGroup",
		ID:     name,
		Parent: RootKey(ctx),
		AuthVersionedEntityMixin: AuthVersionedEntityMixin{
			ModifiedTS:    testModifiedTS,
			ModifiedBy:    "user:test-modifier@example.com",
			AuthDBRev:     1,
			AuthDBPrevRev: 0,
		},
		Description: fmt.Sprintf("This is a test auth group %q.", name),
		Owners:      name,
		CreatedTS:   testCreatedTS,
		CreatedBy:   "user:test-creator@example.com",
	}
}

func testIPAllowlist(ctx context.Context, name string, subnets []string) *AuthIPAllowlist {
	if subnets == nil {
		subnets = []string{
			"127.0.0.1/10",
			"127.0.0.1/20",
		}
	}
	return &AuthIPAllowlist{
		Kind:                     "AuthIPWhitelist",
		ID:                       name,
		Parent:                   RootKey(ctx),
		AuthVersionedEntityMixin: testAuthVersionedEntityMixin(),
		Subnets:                  subnets,
		Description:              fmt.Sprintf("This is a test AuthIPAllowlist %q.", name),
		CreatedTS:                testCreatedTS,
		CreatedBy:                "user:test-creator@example.com",
	}
}

func testAuthDBSnapshot(ctx context.Context, rev int64) *AuthDBSnapshot {
	return &AuthDBSnapshot{
		Kind:           "AuthDBSnapshot",
		ID:             rev,
		AuthDBDeflated: []byte("test-db-deflated"),
		AuthDBSha256:   "test-sha-256",
		CreatedTS:      testCreatedTS,
	}
}

func testAuthRealmsGlobals(ctx context.Context) *AuthRealmsGlobals {
	return &AuthRealmsGlobals{
		AuthVersionedEntityMixin: testAuthVersionedEntityMixin(),
		Kind:                     "AuthRealmsGlobals",
		ID:                       "globals",
		Parent:                   RootKey(ctx),
	}
}

func testAuthProjectRealms(ctx context.Context, projectName string) *AuthProjectRealms {
	return &AuthProjectRealms{
		AuthVersionedEntityMixin: testAuthVersionedEntityMixin(),
		Kind:                     "AuthProjectRealms",
		ID:                       projectName,
		Parent:                   RootKey(ctx),
	}
}

func testAuthDBSnapshotSharded(ctx context.Context, rev int64, shardCount int) (*AuthDBSnapshot, []byte, error) {
	snapshot := &AuthDBSnapshot{
		Kind:         "AuthDBSnapshot",
		ID:           rev,
		ShardIDs:     make([]string, 0, shardCount),
		AuthDBSha256: "test-sha-256",
		CreatedTS:    testCreatedTS,
	}

	var expectedBlob []byte

	for i := 0; i < shardCount; i++ {
		blobShard := []byte(fmt.Sprintf("test-authdb-shard-%v", i))
		expectedBlob = append(expectedBlob, blobShard...)
		hash := sha256.Sum256(blobShard)
		shardID := fmt.Sprintf("%v:%s", rev, hash)
		shard := &AuthDBShard{
			Kind: "AuthDBShard",
			ID:   shardID,
			Blob: blobShard,
		}
		if err := datastore.Put(ctx, shard); err != nil {
			return nil, nil, err
		}
		snapshot.ShardIDs = append(snapshot.ShardIDs, shardID)
	}
	return snapshot, expectedBlob, nil
}

func getAllDatastoreEntities(ctx context.Context, entityKind string, parent *datastore.Key) ([]datastore.PropertyMap, error) {
	query := datastore.NewQuery(entityKind).Ancestor(parent)
	var entities []datastore.PropertyMap
	err := datastore.GetAll(ctx, query, &entities)
	return entities, err
}

// isPropIndexed returns true if any property with the given key is indexed.
func isPropIndexed(pm datastore.PropertyMap, key string) bool {
	ps := pm.Slice(key)
	for _, p := range ps {
		if p.IndexSetting() != datastore.NoIndex {
			return true
		}
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////

func TestGetReplicationState(t *testing.T) {
	t.Parallel()

	Convey("Testing GetReplicationState", t, func() {
		ctx := memory.Use(context.Background())

		state := testAuthReplicationState(ctx, 12345)

		_, err := GetReplicationState(ctx)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		So(datastore.Put(ctx, state), ShouldBeNil)

		actual, err := GetReplicationState(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, state)
	})
}

func TestGetAuthGroup(t *testing.T) {
	t.Parallel()

	Convey("Testing GetAuthGroup", t, func() {
		ctx := memory.Use(context.Background())

		authGroup := testAuthGroup(ctx, "test-auth-group-1")

		_, err := GetAuthGroup(ctx, "test-auth-group-1")
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		So(datastore.Put(ctx, authGroup), ShouldBeNil)

		actual, err := GetAuthGroup(ctx, "test-auth-group-1")
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, authGroup)
	})
}

func TestGetAllAuthGroups(t *testing.T) {
	t.Parallel()

	Convey("Testing GetAllAuthGroups", t, func() {
		ctx := memory.Use(context.Background())

		// Out of order alphabetically by ID.
		So(datastore.Put(ctx,
			testAuthGroup(ctx, "test-auth-group-3"),
			testAuthGroup(ctx, "test-auth-group-1"),
			testAuthGroup(ctx, "test-auth-group-2"),
		), ShouldBeNil)

		actualAuthGroups, err := GetAllAuthGroups(ctx)
		So(err, ShouldBeNil)

		// Returned in alphabetical order.
		So(actualAuthGroups, ShouldResemble, []*AuthGroup{
			testAuthGroup(ctx, "test-auth-group-1"),
			testAuthGroup(ctx, "test-auth-group-2"),
			testAuthGroup(ctx, "test-auth-group-3"),
		})
	})
}

func TestCreateAuthGroup(t *testing.T) {
	t.Parallel()

	Convey("CreateAuthGroup", t, func() {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		ctx, taskScheduler := tq.TestingContext(txndefer.FilterRDS(ctx), nil)

		Convey("empty group name", func() {
			group := testAuthGroup(ctx, "")

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldEqual, ErrInvalidName)
		})

		Convey("invalid group name", func() {
			group := testAuthGroup(ctx, "foo^")

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldEqual, ErrInvalidName)
		})

		Convey("external group name", func() {
			group := testAuthGroup(ctx, "mdb/foo")

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldEqual, ErrInvalidName)
		})

		Convey("group name that already exists", func() {
			So(datastore.Put(ctx,
				testAuthGroup(ctx, "foo"),
			), ShouldBeNil)

			group := testAuthGroup(ctx, "foo")

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldEqual, ErrAlreadyExists)
		})

		Convey("invalid member identities", func() {
			group := testAuthGroup(ctx, "foo")
			group.Members = []string{"no-prefix@google.com"}

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, "bad identity string \"no-prefix@google.com\"")
		})

		Convey("project member identities", func() {
			group := testAuthGroup(ctx, "foo")
			group.Members = []string{"project:abc"}

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, `"project:..." identities aren't allowed in groups`)
		})

		Convey("invalid identity globs", func() {
			group := testAuthGroup(ctx, "foo")
			group.Globs = []string{"*@no-prefix.com"}

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, "bad identity glob string \"*@no-prefix.com\"")
		})

		Convey("project identity globs", func() {
			group := testAuthGroup(ctx, "foo")
			group.Globs = []string{"project:*"}

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, `"project:..." globs aren't allowed in groups`)
		})

		Convey("all referenced groups must exist", func() {
			group := testAuthGroup(ctx, "foo")
			group.Owners = "bar"
			group.Nested = []string{"baz", "qux"}

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidReference)
			So(err, ShouldErrLike, "some referenced groups don't exist: baz, qux, bar")
		})

		Convey("owner must exist", func() {
			group := testAuthGroup(ctx, "foo")
			group.Owners = "bar"

			_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldErrLike, "bar")
		})

		Convey("with empty owners uses 'administrators' group", func() {
			So(datastore.Put(ctx,
				testAuthGroup(ctx, AdminGroup),
			), ShouldBeNil)

			group := emptyAuthGroup(ctx, "foo")
			group.Owners = ""

			createdGroup, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(createdGroup.Owners, ShouldEqual, AdminGroup)
		})

		Convey("group can own itself", func() {
			group := emptyAuthGroup(ctx, "foo")
			group.Owners = "foo"

			createdGroup, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(createdGroup.Owners, ShouldEqual, createdGroup.ID)
		})

		Convey("successfully writes to datastore", func() {
			group := emptyAuthGroup(ctx, "foo")

			createdGroup, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(createdGroup.ID, ShouldEqual, group.ID)
			So(createdGroup.Description, ShouldEqual, group.Description)
			So(createdGroup.Owners, ShouldEqual, group.Owners)
			So(createdGroup.Members, ShouldResemble, group.Members)
			So(createdGroup.Globs, ShouldResemble, group.Globs)
			So(createdGroup.Nested, ShouldResemble, group.Nested)
			So(createdGroup.CreatedBy, ShouldEqual, "user:someone@example.com")
			So(createdGroup.CreatedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(createdGroup.ModifiedBy, ShouldEqual, "user:someone@example.com")
			So(createdGroup.ModifiedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(createdGroup.AuthDBRev, ShouldEqual, 1)
			So(createdGroup.AuthDBPrevRev, ShouldEqual, 0)

			fetchedGroup, err := GetAuthGroup(ctx, "foo")
			So(err, ShouldBeNil)
			So(fetchedGroup.ID, ShouldEqual, group.ID)
			So(fetchedGroup.Description, ShouldEqual, group.Description)
			So(fetchedGroup.Owners, ShouldEqual, group.Owners)
			So(fetchedGroup.Members, ShouldResemble, group.Members)
			So(fetchedGroup.Globs, ShouldResemble, group.Globs)
			So(fetchedGroup.Nested, ShouldResemble, group.Nested)
			So(fetchedGroup.CreatedBy, ShouldEqual, "user:someone@example.com")
			So(fetchedGroup.CreatedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(fetchedGroup.ModifiedBy, ShouldEqual, "user:someone@example.com")
			So(fetchedGroup.ModifiedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(fetchedGroup.AuthDBRev, ShouldEqual, 1)
			So(fetchedGroup.AuthDBPrevRev, ShouldEqual, 0)
		})

		Convey("updates AuthDB revision only on successful write", func() {
			// Create a group.
			{
				group1 := emptyAuthGroup(ctx, "foo")

				createdGroup1, err := CreateAuthGroup(ctx, group1, false, "Go pRPC API")
				So(err, ShouldBeNil)
				So(createdGroup1.AuthDBRev, ShouldEqual, 1)
				So(createdGroup1.AuthDBPrevRev, ShouldEqual, 0)

				state1, err := GetReplicationState(ctx)
				So(err, ShouldBeNil)
				So(state1.AuthDBRev, ShouldEqual, 1)
				tasks := taskScheduler.Tasks()
				So(tasks, ShouldHaveLength, 2)
				processChangeTask := tasks[0]
				So(processChangeTask.Class, ShouldEqual, "process-change-task")
				So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 1})
				replicationTask := tasks[1]
				So(replicationTask.Class, ShouldEqual, "replication-task")
				So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 1})
			}

			// Create a second group.
			{
				group2 := emptyAuthGroup(ctx, "foo2")

				createdGroup2, err := CreateAuthGroup(ctx, group2, false, "Go pRPC API")
				So(err, ShouldBeNil)
				So(createdGroup2.AuthDBRev, ShouldEqual, 2)
				So(createdGroup2.AuthDBPrevRev, ShouldEqual, 0)

				state2, err := GetReplicationState(ctx)
				So(err, ShouldBeNil)
				So(state2.AuthDBRev, ShouldEqual, 2)
				tasks := taskScheduler.Tasks()
				So(tasks, ShouldHaveLength, 4)
			}

			// Try to create another group the same as the second, which should fail.
			{
				_, err := CreateAuthGroup(ctx, emptyAuthGroup(ctx, "foo2"), false, "Go pRPC API")
				So(err, ShouldBeError)

				state3, err := GetReplicationState(ctx)
				So(err, ShouldBeNil)
				So(state3.AuthDBRev, ShouldEqual, 2)
				tasks := taskScheduler.Tasks()
				So(tasks, ShouldHaveLength, 4)
			}
		})

		Convey("creates historical group entities", func() {
			// Create a group.
			{
				group := emptyAuthGroup(ctx, "foo")

				_, err := CreateAuthGroup(ctx, group, false, "test historical comment")
				So(err, ShouldBeNil)

				entities, err := getAllDatastoreEntities(ctx, "AuthGroupHistory", HistoricalRevisionKey(ctx, 1))
				So(err, ShouldBeNil)
				So(entities, ShouldHaveLength, 1)
				historicalEntity := entities[0]
				So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthGroupHistory,\"foo\"")
				So(getStringProp(historicalEntity, "description"), ShouldEqual, group.Description)
				So(getStringProp(historicalEntity, "owners"), ShouldEqual, group.Owners)
				So(getStringSliceProp(historicalEntity, "members"), ShouldResemble, group.Members)
				So(getStringSliceProp(historicalEntity, "globs"), ShouldResemble, group.Globs)
				So(getStringSliceProp(historicalEntity, "nested"), ShouldResemble, group.Nested)
				So(getStringProp(historicalEntity, "created_by"), ShouldEqual, "user:someone@example.com")
				So(getTimeProp(historicalEntity, "created_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
				So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
				So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
				So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
				So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldBeNil)
				So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeFalse)
				So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "test historical comment")
				So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

				// Check no properties are indexed.
				for k := range historicalEntity {
					So(isPropIndexed(historicalEntity, k), ShouldBeFalse)
				}
			}

			// Create a second group.
			{
				group := emptyAuthGroup(ctx, "foo2")

				_, err := CreateAuthGroup(ctx, group, false, "Go pRPC API")
				So(err, ShouldBeNil)

				entities, err := getAllDatastoreEntities(ctx, "AuthGroupHistory", HistoricalRevisionKey(ctx, 2))
				So(err, ShouldBeNil)
				So(entities, ShouldHaveLength, 1)
				historicalEntity := entities[0]
				So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,2/AuthGroupHistory,\"foo2\"")
				So(getStringProp(historicalEntity, "description"), ShouldEqual, group.Description)
				So(getStringProp(historicalEntity, "owners"), ShouldEqual, group.Owners)
				So(getStringSliceProp(historicalEntity, "members"), ShouldResemble, group.Members)
				So(getStringSliceProp(historicalEntity, "globs"), ShouldResemble, group.Globs)
				So(getStringSliceProp(historicalEntity, "nested"), ShouldResemble, group.Nested)
				So(getStringProp(historicalEntity, "created_by"), ShouldEqual, "user:someone@example.com")
				So(getTimeProp(historicalEntity, "created_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
				So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
				So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
				So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 2)
				So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldBeNil)
				So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeFalse)
				So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
				So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

				// Check no properties are indexed.
				for k := range historicalEntity {
					So(isPropIndexed(historicalEntity, k), ShouldBeFalse)
				}
			}
		})
	})
}

func TestUpdateAuthGroup(t *testing.T) {
	t.Parallel()

	Convey("UpdateAuthGroup", t, func() {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{"owners-foo"},
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		ctx, taskScheduler := tq.TestingContext(txndefer.FilterRDS(ctx), nil)

		// A test group to be put in Datastore for updating.
		group := emptyAuthGroup(ctx, "foo")
		group.Owners = "owners-foo"

		// Etag to use for the group, derived from the last-modified time.
		etag := `W/"MjAyMS0wOC0xNlQxMjoyMDowMFo="`

		// Set current auth DB revision to 10.
		So(datastore.Put(ctx, testAuthReplicationState(ctx, 10)), ShouldBeNil)

		Convey("can't update external group", func() {
			group.ID = "mdb/foo"
			So(datastore.Put(ctx, group), ShouldBeNil)
			_, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldErrLike, "cannot update external group")
		})

		Convey("can't update if not an owner", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:someone@example.com",
			})
			So(datastore.Put(ctx, group), ShouldBeNil)
			_, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldEqual, ErrPermissionDenied)
		})

		Convey("can update if admin", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity:       "user:someone@example.com",
				IdentityGroups: []string{AdminGroup},
			})
			So(datastore.Put(ctx, group), ShouldBeNil)
			_, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)
		})

		Convey("can't delete if etag doesn't match", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			_, err := UpdateAuthGroup(ctx, group, nil, "bad-etag", false, "Go pRPC API")
			So(err, ShouldErrLike, ErrConcurrentModification)
		})

		Convey("group name that doesn't exist", func() {
			group.ID = "non-existent-group"
			_, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldEqual, datastore.ErrNoSuchEntity)
		})

		Convey("invalid member identities", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Members = []string{"no-prefix@google.com"}

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"members"}}, etag, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, "bad identity string \"no-prefix@google.com\"")
		})

		Convey("project member identities", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Members = []string{"project:abc"}

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"members"}}, etag, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, `"project:..." identities aren't allowed in groups`)
		})

		Convey("invalid identity globs", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Globs = []string{"*@no-prefix.com"}

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"globs"}}, etag, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, "bad identity glob string \"*@no-prefix.com\"")
		})

		Convey("project identity globs", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Globs = []string{"project:*"}

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"globs"}}, etag, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidIdentity)
			So(err, ShouldErrLike, `"project:..." globs aren't allowed in groups`)
		})

		Convey("all nested groups must exist", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Nested = []string{"baz", "qux"}

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, etag, false, "Go pRPC API")
			So(err, ShouldUnwrapTo, ErrInvalidReference)
			So(err, ShouldErrLike, "some referenced groups don't exist")
		})

		Convey("owner must exist", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Owners = "bar"

			_, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"owners"}}, etag, false, "Go pRPC API")
			So(err, ShouldErrLike, "bar")
		})

		Convey("with empty owners uses 'administrators' group", func() {
			So(datastore.Put(ctx, testAuthGroup(ctx, AdminGroup)), ShouldBeNil)
			So(datastore.Put(ctx, group), ShouldBeNil)

			group.Owners = ""

			updatedGroup, err := UpdateAuthGroup(ctx, group, &fieldmaskpb.FieldMask{Paths: []string{"owners"}}, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(updatedGroup.Owners, ShouldEqual, AdminGroup)
		})

		Convey("successfully writes to datastore", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-owner-group")), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-nested-group")), ShouldBeNil)

			group.Description = "updated description"
			group.Owners = "new-owner-group"
			group.Members = []string{"user:updated@example.com"}
			group.Globs = []string{"user:*@updated.com"}
			group.Nested = []string{"new-nested-group"}

			updatedGroup, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(updatedGroup.ID, ShouldEqual, group.ID)
			So(updatedGroup.Description, ShouldEqual, group.Description)
			So(updatedGroup.Owners, ShouldEqual, group.Owners)
			So(updatedGroup.Members, ShouldResemble, group.Members)
			So(updatedGroup.Globs, ShouldResemble, group.Globs)
			So(updatedGroup.Nested, ShouldResemble, group.Nested)
			So(updatedGroup.CreatedBy, ShouldEqual, "user:test-creator@example.com")
			So(updatedGroup.CreatedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(updatedGroup.ModifiedBy, ShouldEqual, "user:someone@example.com")
			So(updatedGroup.ModifiedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(updatedGroup.AuthDBRev, ShouldEqual, 11)
			So(updatedGroup.AuthDBPrevRev, ShouldEqual, 1)

			fetchedGroup, err := GetAuthGroup(ctx, "foo")
			So(err, ShouldBeNil)
			So(fetchedGroup.ID, ShouldEqual, group.ID)
			So(fetchedGroup.Description, ShouldEqual, group.Description)
			So(fetchedGroup.Owners, ShouldEqual, group.Owners)
			So(fetchedGroup.Members, ShouldResemble, group.Members)
			So(fetchedGroup.Globs, ShouldResemble, group.Globs)
			So(fetchedGroup.Nested, ShouldResemble, group.Nested)
			So(fetchedGroup.CreatedBy, ShouldEqual, "user:test-creator@example.com")
			So(fetchedGroup.CreatedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(fetchedGroup.ModifiedBy, ShouldEqual, "user:someone@example.com")
			So(fetchedGroup.ModifiedTS.Unix(), ShouldEqual, testCreatedTS.Unix())
			So(fetchedGroup.AuthDBRev, ShouldEqual, 11)
			So(fetchedGroup.AuthDBPrevRev, ShouldEqual, 1)
		})

		Convey("updates AuthDB revision only on successful write", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-owner-group")), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-nested-group")), ShouldBeNil)

			// Update a group, should succeed and bump AuthDB revision.
			group.Description = "updated description"
			group.Owners = "new-owner-group"
			group.Members = []string{"user:updated@example.com"}
			group.Globs = []string{"user:*@updated.com"}
			group.Nested = []string{"new-nested-group"}

			updatedGroup, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(updatedGroup.AuthDBRev, ShouldEqual, 11)
			So(updatedGroup.AuthDBPrevRev, ShouldEqual, 1)

			state, err := GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state.AuthDBRev, ShouldEqual, 11)
			tasks := taskScheduler.Tasks()
			So(tasks, ShouldHaveLength, 2)
			processChangeTask := tasks[0]
			So(processChangeTask.Class, ShouldEqual, "process-change-task")
			So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 11})
			replicationTask := tasks[1]
			So(replicationTask.Class, ShouldEqual, "replication-task")
			So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 11})

			// Update a group, should fail (due to bad etag) and *not* bump AuthDB revision.
			_, err = UpdateAuthGroup(ctx, group, nil, "bad-etag", false, "Go pRPC API")
			So(err, ShouldBeError)

			state, err = GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state.AuthDBRev, ShouldEqual, 11)
			So(taskScheduler.Tasks(), ShouldHaveLength, 2)
		})

		Convey("creates historical group entities", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-owner-group")), ShouldBeNil)
			So(datastore.Put(ctx, emptyAuthGroup(ctx, "new-nested-group")), ShouldBeNil)

			// Update a group, should succeed and bump AuthDB revision.
			group.Description = "updated description"
			group.Owners = "new-owner-group"
			group.Members = []string{"user:updated@example.com"}
			group.Globs = []string{"user:*@updated.com"}
			group.Nested = []string{"new-nested-group"}

			_, err := UpdateAuthGroup(ctx, group, nil, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)

			entities, err := getAllDatastoreEntities(ctx, "AuthGroupHistory", HistoricalRevisionKey(ctx, 11))
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 1)
			historicalEntity := entities[0]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,11/AuthGroupHistory,\"foo\"")
			So(getStringProp(historicalEntity, "description"), ShouldEqual, group.Description)
			So(getStringProp(historicalEntity, "owners"), ShouldEqual, group.Owners)
			So(getStringSliceProp(historicalEntity, "members"), ShouldResemble, group.Members)
			So(getStringSliceProp(historicalEntity, "globs"), ShouldResemble, group.Globs)
			So(getStringSliceProp(historicalEntity, "nested"), ShouldResemble, group.Nested)
			So(getStringProp(historicalEntity, "created_by"), ShouldEqual, "user:test-creator@example.com")
			So(getTimeProp(historicalEntity, "created_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 11)
			So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldEqual, 1)
			So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeFalse)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

			// Check no properties are indexed.
			for k := range historicalEntity {
				So(isPropIndexed(historicalEntity, k), ShouldBeFalse)
			}
		})

		Convey("cyclic dependencies", func() {
			// Use admin creds for simplicity.
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity:       "user:someone@example.com",
				IdentityGroups: []string{AdminGroup},
			})

			// Initial state is a tree with 4 groups like this:
			//
			//      A
			//     / \
			//    B1 B2
			//   /
			//  C
			//
			a := emptyAuthGroup(ctx, "A")
			a.Nested = []string{"B1", "B2"}
			b1 := emptyAuthGroup(ctx, "B1")
			b1.Nested = []string{"C"}
			b2 := emptyAuthGroup(ctx, "B2")
			c := emptyAuthGroup(ctx, "C")
			So(datastore.Put(ctx, []*AuthGroup{a, b1, b2, c}), ShouldBeNil)

			Convey("self-reference", func() {
				//   A
				//  /
				// A
				a.Nested = []string{"A"}

				_, err := UpdateAuthGroup(ctx, a, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, "", false, "Go pRPC API")
				So(err, ShouldErrLike, "groups can not have cyclic dependencies: A -> A.")
			})

			Convey("cycle of length 2", func() {
				//   A
				//  /
				// B2
				//  \
				//   A
				b2.Nested = []string{"A"}

				_, err := UpdateAuthGroup(ctx, b2, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, "", false, "Go pRPC API")
				So(err, ShouldErrLike, "groups can not have cyclic dependencies: B2 -> A -> B2.")
			})

			Convey("cycle of length 3", func() {
				//   A
				//  /
				// B1
				//  \
				//   C
				//  /
				// A
				c.Nested = []string{"A"}

				_, err := UpdateAuthGroup(ctx, c, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, "", false, "Go pRPC API")
				So(err, ShouldErrLike, "groups can not have cyclic dependencies: C -> A -> B1 -> C.")
			})

			Convey("cycle not at root", func() {
				//   B1
				//  /
				// C
				//  \
				//   B1
				c.Nested = []string{"B1"}

				_, err := UpdateAuthGroup(ctx, c, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, "", false, "Go pRPC API")
				So(err, ShouldErrLike, "groups can not have cyclic dependencies: C -> B1 -> C.")
			})

			Convey("diamond shape", func() {
				//      A
				//     / \
				//    B1 B2
				//     \ /
				//      C
				b2.Nested = []string{"C"}

				_, err := UpdateAuthGroup(ctx, b2, &fieldmaskpb.FieldMask{Paths: []string{"nested"}}, "", false, "Go pRPC API")
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestDeleteAuthGroup(t *testing.T) {
	t.Parallel()

	Convey("DeleteAuthGroup", t, func() {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{AdminGroup},
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		ctx, taskScheduler := tq.TestingContext(txndefer.FilterRDS(ctx), nil)

		// A test group to be put in Datastore for deletion.
		group := testAuthGroup(ctx, "foo")
		group.Owners = "foo"
		group.AuthDBRev = 0
		group.AuthDBPrevRev = 0

		// Etag to use for the group, derived from the last-modified time.
		etag := `W/"MjAyMS0wOC0xNlQxMjoyMDowMFo="`

		Convey("can't delete the admin group", func() {
			err := DeleteAuthGroup(ctx, AdminGroup, "", false, "Go pRPC API")
			So(err, ShouldEqual, ErrPermissionDenied)
		})

		Convey("can't delete external group", func() {
			group.ID = "mdb/foo"
			So(datastore.Put(ctx, group), ShouldBeNil)
			err := DeleteAuthGroup(ctx, group.ID, "", false, "Go pRPC API")
			So(err, ShouldErrLike, "cannot delete external group")
		})

		Convey("can't delete if not an owner or admin", func() {
			ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
				Identity: "user:someone@example.com",
			})
			So(datastore.Put(ctx, group), ShouldBeNil)
			err := DeleteAuthGroup(ctx, group.ID, "", false, "Go pRPC API")
			So(err, ShouldEqual, ErrPermissionDenied)
		})

		Convey("can't delete if etag doesn't match", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			err := DeleteAuthGroup(ctx, group.ID, "bad-etag", false, "Go pRPC API")
			So(err, ShouldErrLike, ErrConcurrentModification)
		})

		Convey("group name that doesn't exist", func() {
			err := DeleteAuthGroup(ctx, "non-existent-group", "", false, "Go pRPC API")
			So(err, ShouldEqual, datastore.ErrNoSuchEntity)
		})

		Convey("can't delete if group owns another group", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			ownedGroup := testAuthGroup(ctx, "owned")
			ownedGroup.Owners = group.ID
			So(datastore.Put(ctx, ownedGroup), ShouldBeNil)

			err := DeleteAuthGroup(ctx, group.ID, "", false, "Go pRPC API")
			So(err, ShouldErrLike, ErrReferencedEntity)
			So(err, ShouldErrLike, "this group is referenced by other groups: [owned]")
		})

		Convey("can't delete if group is nested by group", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)

			nestingGroup := testAuthGroup(ctx, "nester")
			nestingGroup.Nested = []string{group.ID}
			So(datastore.Put(ctx, nestingGroup), ShouldBeNil)

			err := DeleteAuthGroup(ctx, group.ID, "", false, "Go pRPC API")
			So(err, ShouldErrLike, ErrReferencedEntity)
			So(err, ShouldErrLike, "this group is referenced by other groups: [nester]")
		})

		Convey("successfully deletes from datastore and updates AuthDB", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			err := DeleteAuthGroup(ctx, group.ID, etag, false, "Go pRPC API")
			So(err, ShouldBeNil)

			state1, err := GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state1.AuthDBRev, ShouldEqual, 1)
			tasks := taskScheduler.Tasks()
			So(tasks, ShouldHaveLength, 2)
			processChangeTask := tasks[0]
			So(processChangeTask.Class, ShouldEqual, "process-change-task")
			So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 1})
			replicationTask := tasks[1]
			So(replicationTask.Class, ShouldEqual, "replication-task")
			So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 1})
		})

		Convey("creates historical group entities", func() {
			So(datastore.Put(ctx, group), ShouldBeNil)
			err := DeleteAuthGroup(ctx, group.ID, "", false, "Go pRPC API")
			So(err, ShouldBeNil)

			entities, err := getAllDatastoreEntities(ctx, "AuthGroupHistory", HistoricalRevisionKey(ctx, 1))
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 1)
			historicalEntity := entities[0]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthGroupHistory,\"foo\"")
			So(getStringProp(historicalEntity, "description"), ShouldEqual, group.Description)
			So(getStringProp(historicalEntity, "owners"), ShouldEqual, group.Owners)
			So(getStringSliceProp(historicalEntity, "members"), ShouldResemble, group.Members)
			So(getStringSliceProp(historicalEntity, "globs"), ShouldResemble, group.Globs)
			So(getStringSliceProp(historicalEntity, "nested"), ShouldResemble, group.Nested)
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldBeNil)
			So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeTrue)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

			// Check no properties are indexed.
			for k := range historicalEntity {
				So(isPropIndexed(historicalEntity, k), ShouldBeFalse)
			}
		})
	})
}

func TestGetAuthIPAllowlist(t *testing.T) {
	t.Parallel()

	Convey("Testing GetAuthIPAllowlist", t, func() {
		ctx := memory.Use(context.Background())

		authIPAllowlist := testIPAllowlist(ctx, "test-auth-ip-allowlist-1", []string{
			"123.456.789.101/24",
			"123.456.789.112/24",
		})

		_, err := GetAuthIPAllowlist(ctx, "test-auth-ip-allowlist-1")
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		So(datastore.Put(ctx, authIPAllowlist), ShouldBeNil)

		actual, err := GetAuthIPAllowlist(ctx, "test-auth-ip-allowlist-1")
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, authIPAllowlist)
	})
}

func TestGetAllAuthIPAllowlists(t *testing.T) {
	t.Parallel()

	Convey("Testing GetAllAuthIPAllowlists", t, func() {
		ctx := memory.Use(context.Background())

		// Out of order alphabetically by ID.
		So(datastore.Put(ctx,
			testIPAllowlist(ctx, "test-allowlist-3", nil),
			testIPAllowlist(ctx, "test-allowlist-1", nil),
			testIPAllowlist(ctx, "test-allowlist-2", nil),
		), ShouldBeNil)

		actualAuthIPAllowlists, err := GetAllAuthIPAllowlists(ctx)
		So(err, ShouldBeNil)

		// Returned in alphabetical order.
		So(actualAuthIPAllowlists, ShouldResemble, []*AuthIPAllowlist{
			testIPAllowlist(ctx, "test-allowlist-1", nil),
			testIPAllowlist(ctx, "test-allowlist-2", nil),
			testIPAllowlist(ctx, "test-allowlist-3", nil),
		})
	})
}

func TestUpdateAllowlistEntities(t *testing.T) {
	t.Parallel()

	Convey("Testing UpdateAllowlistEntities", t, func() {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{AdminGroup},
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		ctx, taskScheduler := tq.TestingContext(txndefer.FilterRDS(ctx), nil)

		So(datastore.Put(ctx,
			testIPAllowlist(ctx, "test-allowlist-1", nil),
			testIPAllowlist(ctx, "test-allowlist-2", []string{"0.0.0.0/24", "127.0.0.1/20"}),
		), ShouldBeNil)

		baseSubnetMap := map[string][]string{
			"test-allowlist-1": {"127.0.0.1/10", "127.0.0.1/20"},
			"test-allowlist-2": {"0.0.0.0/24", "127.0.0.1/20"},
		}
		baseAllowlistSlice := []*AuthIPAllowlist{
			{
				AuthVersionedEntityMixin: AuthVersionedEntityMixin{
					ModifiedTS:    testCreatedTS,
					ModifiedBy:    "user:someone@example.com",
					AuthDBRev:     1,
					AuthDBPrevRev: 1337,
				},
				Kind:        "AuthIPWhitelist",
				ID:          "test-allowlist-1",
				Parent:      RootKey(ctx),
				Subnets:     []string{"127.0.0.1/10", "127.0.0.1/20"},
				Description: fmt.Sprintf("This is a test AuthIPAllowlist %q.", "test-allowlist-1"),
				CreatedTS:   testCreatedTS,
				CreatedBy:   "user:test-creator@example.com",
			},
			{
				AuthVersionedEntityMixin: AuthVersionedEntityMixin{
					ModifiedTS:    testCreatedTS,
					ModifiedBy:    "user:someone@example.com",
					AuthDBRev:     1,
					AuthDBPrevRev: 1337,
				},
				Kind:        "AuthIPWhitelist",
				ID:          "test-allowlist-2",
				Parent:      RootKey(ctx),
				Subnets:     []string{"0.0.0.0/24", "127.0.0.1/20"},
				Description: fmt.Sprintf("This is a test AuthIPAllowlist %q.", "test-allowlist-2"),
				CreatedTS:   testCreatedTS,
				CreatedBy:   "user:test-creator@example.com",
			},
		}

		allowlistToCreate := &AuthIPAllowlist{
			AuthVersionedEntityMixin: AuthVersionedEntityMixin{
				ModifiedTS:    testCreatedTS,
				ModifiedBy:    "user:someone@example.com",
				AuthDBRev:     1,
				AuthDBPrevRev: 0,
			},
			Kind:        "AuthIPWhitelist",
			ID:          "test-allowlist-3",
			Parent:      RootKey(ctx),
			Subnets:     []string{"123.4.5.6"},
			Description: "Imported from ip_allowlist.cfg",
			CreatedTS:   testCreatedTS,
			CreatedBy:   "user:someone@example.com",
		}

		Convey("Create allowlist entity", func() {
			baseSubnetMap["test-allowlist-3"] = []string{"123.4.5.6"}
			So(UpdateAllowlistEntities(ctx, baseSubnetMap, false, "Go pRPC API"), ShouldBeNil)
			allowlists, err := GetAllAuthIPAllowlists(ctx)
			So(err, ShouldBeNil)
			expectedSlice := append(baseAllowlistSlice, allowlistToCreate)
			So(allowlists, ShouldResemble, expectedSlice)
		})

		Convey("Update allowlist entity", func() {
			baseSubnetMap["test-allowlist-1"] = []string{"122.22.44.66"}
			So(UpdateAllowlistEntities(ctx, baseSubnetMap, false, "Go pRPC API"), ShouldBeNil)
			allowlists, err := GetAllAuthIPAllowlists(ctx)
			baseAllowlistSlice[0].Subnets = []string{"122.22.44.66"}
			So(err, ShouldBeNil)
			So(allowlists, ShouldResemble, baseAllowlistSlice)
		})

		Convey("Delete allowlist entity", func() {
			delete(baseSubnetMap, "test-allowlist-1")
			So(UpdateAllowlistEntities(ctx, baseSubnetMap, false, "Go pRPC API"), ShouldBeNil)
			allowlists, err := GetAllAuthIPAllowlists(ctx)
			expectedSlice := baseAllowlistSlice[1:]
			So(err, ShouldBeNil)
			So(allowlists, ShouldResemble, expectedSlice)
		})

		Convey("Multiple allowlist entity changes", func() {
			baseSubnetMap["test-allowlist-3"] = []string{"123.4.5.6"}
			baseSubnetMap["test-allowlist-1"] = []string{"122.22.44.66"}
			delete(baseSubnetMap, "test-allowlist-2")
			So(UpdateAllowlistEntities(ctx, baseSubnetMap, false, "Go pRPC API"), ShouldBeNil)
			allowlists, err := GetAllAuthIPAllowlists(ctx)
			allowlist0Copy := *baseAllowlistSlice[0]
			allowlist0Copy.Subnets = baseSubnetMap["test-allowlist-1"]
			expectedAllowlists := []*AuthIPAllowlist{&allowlist0Copy, allowlistToCreate}
			So(err, ShouldBeNil)
			So(allowlists, ShouldResemble, expectedAllowlists)

			state1, err := GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state1.AuthDBRev, ShouldEqual, 1)
			tasks := taskScheduler.Tasks()
			So(tasks, ShouldHaveLength, 2)
			processChangeTask := tasks[0]
			So(processChangeTask.Class, ShouldEqual, "process-change-task")
			So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 1})
			replicationTask := tasks[1]
			So(replicationTask.Class, ShouldEqual, "replication-task")
			So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 1})

			entities, err := getAllDatastoreEntities(ctx, "AuthIPWhitelistHistory", HistoricalRevisionKey(ctx, 1))
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 3)
			historicalEntity := entities[0]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthIPWhitelistHistory,\"test-allowlist-1\"")
			So(getStringProp(historicalEntity, "description"), ShouldEqual, baseAllowlistSlice[0].Description)
			So(getStringSliceProp(historicalEntity, "subnets"), ShouldResemble, allowlist0Copy.Subnets)
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getInt64Prop(historicalEntity, "auth_db_prev_rev"), ShouldEqual, 1337)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

			historicalEntity = entities[1]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthIPWhitelistHistory,\"test-allowlist-2\"")
			So(getStringProp(historicalEntity, "description"), ShouldEqual, baseAllowlistSlice[1].Description)
			So(getStringSliceProp(historicalEntity, "subnets"), ShouldResemble, baseAllowlistSlice[1].Subnets)
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getInt64Prop(historicalEntity, "auth_db_prev_rev"), ShouldEqual, 1337)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")
			So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeTrue)

			historicalEntity = entities[2]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthIPWhitelistHistory,\"test-allowlist-3\"")
			So(getStringProp(historicalEntity, "description"), ShouldEqual, allowlistToCreate.Description)
			So(getStringSliceProp(historicalEntity, "subnets"), ShouldResemble, allowlistToCreate.Subnets)
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldBeNil)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")

			// Check no properties are indexed.
			for k := range historicalEntity {
				So(isPropIndexed(historicalEntity, k), ShouldBeFalse)
			}
		})
	})
}

func TestAuthGlobalConfig(t *testing.T) {
	t.Parallel()

	Convey("Testing GetAuthGlobalConfig", t, func() {
		ctx := memory.Use(context.Background())

		_, err := GetAuthGlobalConfig(ctx)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		cfg := testAuthGlobalConfig(ctx)
		err = datastore.Put(ctx, cfg)
		So(err, ShouldBeNil)

		actual, err := GetAuthGlobalConfig(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, cfg)
	})

	Convey("Testing UpdateAuthGlobalConfig", t, func() {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{AdminGroup},
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		ctx, taskScheduler := tq.TestingContext(txndefer.FilterRDS(ctx), nil)
		oauthcfgpb := &configspb.OAuthConfig{
			PrimaryClientId:     "new-test-client-id",
			PrimaryClientSecret: "new-test-client-secret",
			ClientIds: []string{
				"new-test-client-id-1",
				"new-test-client-id-2",
			},
			TokenServerUrl: "https://new-token-server-url.example.com",
		}
		seccfgpb := testSecurityConfig()

		Convey("Creating new AuthGlobalConfig", func() {
			So(UpdateAuthGlobalConfig(ctx, oauthcfgpb, seccfgpb, false, "Go pRPC API"), ShouldBeNil)
			updatedCfg, err := GetAuthGlobalConfig(ctx)
			So(err, ShouldBeNil)
			So(updatedCfg, ShouldResemble, &AuthGlobalConfig{
				AuthVersionedEntityMixin: AuthVersionedEntityMixin{
					ModifiedTS:    testCreatedTS,
					ModifiedBy:    "user:someone@example.com",
					AuthDBRev:     1,
					AuthDBPrevRev: 0,
				},
				Kind:                     "AuthGlobalConfig",
				ID:                       "root",
				OAuthClientID:            "new-test-client-id",
				OAuthAdditionalClientIDs: []string{"new-test-client-id-1", "new-test-client-id-2"},
				OAuthClientSecret:        "new-test-client-secret",
				TokenServerURL:           "https://new-token-server-url.example.com",
				SecurityConfig:           testSecurityConfigBlob(),
			})
			state1, err := GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state1.AuthDBRev, ShouldEqual, 1)
			tasks := taskScheduler.Tasks()
			So(tasks, ShouldHaveLength, 2)
			processChangeTask := tasks[0]
			So(processChangeTask.Class, ShouldEqual, "process-change-task")
			So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 1})
			replicationTask := tasks[1]
			So(replicationTask.Class, ShouldEqual, "replication-task")
			So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 1})

			entities, err := getAllDatastoreEntities(ctx, "AuthGlobalConfigHistory", HistoricalRevisionKey(ctx, 1))
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 1)
			historicalEntity := entities[0]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthGlobalConfigHistory,\"root\"")
			So(getStringProp(historicalEntity, "oauth_client_id"), ShouldEqual, "new-test-client-id")
			So(getStringProp(historicalEntity, "oauth_client_secret"), ShouldEqual, "new-test-client-secret")
			So(getStringProp(historicalEntity, "token_server_url"), ShouldEqual, "https://new-token-server-url.example.com")
			So(getStringSliceProp(historicalEntity, "oauth_additional_client_ids"), ShouldResemble, []string{"new-test-client-id-1", "new-test-client-id-2"})
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getProp(historicalEntity, "auth_db_prev_rev"), ShouldBeNil)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")
			So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeFalse)
		})

		Convey("Updating AuthGlobalConfig", func() {
			So(datastore.Put(ctx, testAuthGlobalConfig(ctx)), ShouldBeNil)
			So(UpdateAuthGlobalConfig(ctx, oauthcfgpb, seccfgpb, false, "Go pRPC API"), ShouldBeNil)
			updatedCfg, err := GetAuthGlobalConfig(ctx)
			So(err, ShouldBeNil)
			So(updatedCfg, ShouldResemble, &AuthGlobalConfig{
				AuthVersionedEntityMixin: AuthVersionedEntityMixin{
					ModifiedTS:    testCreatedTS,
					ModifiedBy:    "user:someone@example.com",
					AuthDBRev:     1,
					AuthDBPrevRev: 1337,
				},
				Kind:                     "AuthGlobalConfig",
				ID:                       "root",
				OAuthClientID:            "new-test-client-id",
				OAuthAdditionalClientIDs: []string{"new-test-client-id-1", "new-test-client-id-2"},
				OAuthClientSecret:        "new-test-client-secret",
				TokenServerURL:           "https://new-token-server-url.example.com",
				SecurityConfig:           testSecurityConfigBlob(),
			})
			state1, err := GetReplicationState(ctx)
			So(err, ShouldBeNil)
			So(state1.AuthDBRev, ShouldEqual, 1)
			tasks := taskScheduler.Tasks()
			So(tasks, ShouldHaveLength, 2)
			processChangeTask := tasks[0]
			So(processChangeTask.Class, ShouldEqual, "process-change-task")
			So(processChangeTask.Payload, ShouldResembleProto, &taskspb.ProcessChangeTask{AuthDbRev: 1})
			replicationTask := tasks[1]
			So(replicationTask.Class, ShouldEqual, "replication-task")
			So(replicationTask.Payload, ShouldResembleProto, &taskspb.ReplicationTask{AuthDbRev: 1})

			entities, err := getAllDatastoreEntities(ctx, "AuthGlobalConfigHistory", HistoricalRevisionKey(ctx, 1))
			So(err, ShouldBeNil)
			So(entities, ShouldHaveLength, 1)
			historicalEntity := entities[0]
			So(getDatastoreKey(historicalEntity).String(), ShouldEqual, "dev~app::/AuthGlobalConfig,\"root\"/Rev,1/AuthGlobalConfigHistory,\"root\"")
			So(getStringProp(historicalEntity, "oauth_client_id"), ShouldEqual, "new-test-client-id")
			So(getStringProp(historicalEntity, "oauth_client_secret"), ShouldEqual, "new-test-client-secret")
			So(getStringProp(historicalEntity, "token_server_url"), ShouldEqual, "https://new-token-server-url.example.com")
			So(getByteSliceProp(historicalEntity, "security_config"), ShouldResemble, testSecurityConfigBlob())
			So(getStringSliceProp(historicalEntity, "oauth_additional_client_ids"), ShouldResemble, []string{"new-test-client-id-1", "new-test-client-id-2"})
			So(getStringProp(historicalEntity, "modified_by"), ShouldEqual, "user:someone@example.com")
			So(getTimeProp(historicalEntity, "modified_ts").Unix(), ShouldEqual, testCreatedTS.Unix())
			So(getInt64Prop(historicalEntity, "auth_db_rev"), ShouldEqual, 1)
			So(getInt64Prop(historicalEntity, "auth_db_prev_rev"), ShouldEqual, 1337)
			So(getStringProp(historicalEntity, "auth_db_change_comment"), ShouldEqual, "Go pRPC API")
			So(getStringProp(historicalEntity, "auth_db_app_version"), ShouldEqual, "test-version")
			So(getBoolProp(historicalEntity, "auth_db_deleted"), ShouldBeFalse)
		})
	})
}

func TestAuthRealmsConfig(t *testing.T) {
	t.Parallel()

	getCtx := func() (context.Context, *tqtesting.Scheduler) {
		ctx := auth.WithState(memory.Use(context.Background()), &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = clock.Set(ctx, testclock.New(testCreatedTS))
		ctx = info.SetImageVersion(ctx, "test-version")
		return tq.TestingContext(txndefer.FilterRDS(ctx), nil)
	}

	Convey("Testing GetAuthRealmsGlobals", t, func() {
		ctx := memory.Use(context.Background())

		_, err := GetAuthRealmsGlobals(ctx)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		realmGlobals := testAuthRealmsGlobals(ctx)
		err = datastore.Put(ctx, realmGlobals)
		So(err, ShouldBeNil)

		actual, err := GetAuthRealmsGlobals(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, realmGlobals)
	})

	Convey("Testing GetAuthProjectRealms", t, func() {
		ctx := memory.Use(context.Background())

		testProject := "testproject"
		_, err := GetAuthProjectRealms(ctx, testProject)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		projectRealms := testAuthProjectRealms(ctx, testProject)
		err = datastore.Put(ctx, projectRealms)
		So(err, ShouldBeNil)

		actual, err := GetAuthProjectRealms(ctx, testProject)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, projectRealms)
	})

	Convey("Testing GetAllAuthProjectRealms", t, func() {
		ctx := memory.Use(context.Background())

		// Querying when there are no project realms should succeed.
		actual, err := GetAllAuthProjectRealms(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldBeEmpty)

		// Put 2 project realms in datastore.
		projectRealmsA := testAuthProjectRealms(ctx, "testproject-a")
		projectRealmsB := testAuthProjectRealms(ctx, "testproject-b")
		So(datastore.Put(ctx, projectRealmsA, projectRealmsB), ShouldBeNil)

		actual, err = GetAllAuthProjectRealms(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldHaveLength, 2)
		// No guarantees on order, so sort the output before comparing.
		sort.Slice(actual, func(i, j int) bool {
			return actual[i].ID < actual[j].ID
		})
		So(actual, ShouldResembleProto, []*AuthProjectRealms{
			projectRealmsA,
			projectRealmsB,
		})
	})

	Convey("Testing DeleteAuthProjectRealms", t, func() {
		ctx, ts := getCtx()

		testProject := "testproject"
		projectRealms := testAuthProjectRealms(ctx, testProject)
		err := datastore.Put(ctx, projectRealms)
		So(err, ShouldBeNil)

		_, err = GetAuthProjectRealms(ctx, testProject)
		So(err, ShouldBeNil)

		err = DeleteAuthProjectRealms(ctx, testProject, false, "Go pRPC API")
		So(err, ShouldBeNil)
		So(ts.Tasks(), ShouldHaveLength, 2)

		err = DeleteAuthProjectRealms(ctx, testProject, false, "Go pRPC API")
		So(err, ShouldErrLike, datastore.ErrNoSuchEntity)
	})

	Convey("Testing GetAuthProjectRealmsMeta", t, func() {
		ctx := memory.Use(context.Background())

		testProject := "testproject"
		_, err := GetAuthProjectRealmsMeta(ctx, testProject)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		projectRealms := testAuthProjectRealms(ctx, testProject)
		projectRealmsMeta := makeAuthProjectRealmsMeta(ctx, testProject)

		So(datastore.Put(ctx, projectRealms), ShouldBeNil)
		So(datastore.Put(ctx, projectRealmsMeta), ShouldBeNil)

		actual, err := GetAuthProjectRealmsMeta(ctx, testProject)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, projectRealmsMeta)
		actualID, err := actual.ProjectID()
		So(err, ShouldBeNil)
		So(actualID, ShouldEqual, testProject)
	})

	Convey("Testing GetAllAuthProjectRealmsMeta", t, func() {
		ctx := memory.Use(context.Background())

		testProjects := []string{"testproject-1", "testproject-2", "testproject-3"}
		projectRealms := make([]*AuthProjectRealms, len(testProjects))
		projectRealmsMeta := make([]*AuthProjectRealmsMeta, len(testProjects))
		for i, project := range testProjects {
			projectRealms[i] = testAuthProjectRealms(ctx, project)
			projectRealmsMeta[i] = makeAuthProjectRealmsMeta(ctx, project)
		}

		So(datastore.Put(ctx, projectRealms, projectRealmsMeta), ShouldBeNil)

		actual, err := GetAllAuthProjectRealmsMeta(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, projectRealmsMeta)
		for idx, proj := range testProjects {
			id, err := actual[idx].ProjectID()
			So(err, ShouldBeNil)
			So(id, ShouldEqual, proj)
		}
	})

	Convey("Testing UpdateAuthRealmsGlobals", t, func() {
		Convey("no previous AuthRealmsGlobals entity present", func() {
			ctx, ts := getCtx()

			permCfg := &configspb.PermissionsConfig{
				Role: []*configspb.PermissionsConfig_Role{
					{
						Name: "role/test.role",
						Permissions: []*protocol.Permission{
							{
								Name: "test.perm.create",
							},
						},
					},
				},
			}

			err := UpdateAuthRealmsGlobals(ctx, permCfg, false, "Go pRPC API")
			So(err, ShouldBeNil)
			So(ts.Tasks(), ShouldHaveLength, 2)

			fetched, err := GetAuthRealmsGlobals(ctx)
			So(err, ShouldBeNil)
			So(fetched.PermissionsList.GetPermissions(), ShouldResembleProto,
				[]*protocol.Permission{
					{
						Name:     "test.perm.create",
						Internal: false,
					},
				})
		})

		Convey("updating permissions entity already present", func() {
			ctx, ts := getCtx()

			permCreate, _ := proto.Marshal(&protocol.Permission{
				Name: "test.perm.create",
			})
			permTwoCreate, _ := proto.Marshal(&protocol.Permission{
				Name: "testtwo.perm.delete",
			})
			permIntSched, _ := proto.Marshal(&protocol.Permission{
				Name:     "testint.perm.schedule",
				Internal: true,
			})

			authRealmGlobals := testAuthRealmsGlobals(ctx)
			// This is a little inconsistent with how exactly it'll be in datastore but
			// since we can't get the exact format that the Python version got, this is
			// the closest we can get which is what it looks like after it's been unmarshalled
			// from Python.
			authRealmGlobals.Permissions = []string{
				string(permCreate),
				string(permTwoCreate),
				string(permIntSched),
			}

			So(datastore.Put(ctx, authRealmGlobals), ShouldBeNil)

			permsCfg := &configspb.PermissionsConfig{
				Role: []*configspb.PermissionsConfig_Role{
					{
						Name: "role/test.role",
						Permissions: []*protocol.Permission{
							{
								Name: "test.perm.create",
							},
						},
						Includes: []string{},
					},
					{
						Name: "role/test.role.two",
						Permissions: []*protocol.Permission{
							{
								Name: "testtwo.perm.delete",
							},
						},
					},
					{
						Name: "role/luci.internal.testint.role",
						Permissions: []*protocol.Permission{
							{
								Name:     "testint.perm.schedule",
								Internal: true,
							},
						},
					},
				},
			}

			So(UpdateAuthRealmsGlobals(ctx, permsCfg, false, "Go pRPC API"), ShouldBeNil)
			So(ts.Tasks(), ShouldHaveLength, 2)

			fetched, err := GetAuthRealmsGlobals(ctx)
			So(err, ShouldBeNil)
			So(fetched.PermissionsList.GetPermissions(), ShouldResembleProto,
				[]*protocol.Permission{
					{
						Name:     "test.perm.create",
						Internal: false,
					},
					{
						Name:     "testint.perm.schedule",
						Internal: true,
					},
					{
						Name:     "testtwo.perm.delete",
						Internal: false,
					},
				})
		})

		Convey("skip update if permissions unchanged", func() {
			ctx, ts := getCtx()

			permCreate, _ := proto.Marshal(&protocol.Permission{
				Name: "test.perm.create",
			})
			permTwoCreate, _ := proto.Marshal(&protocol.Permission{
				Name: "testtwo.perm.delete",
			})
			permIntSched, _ := proto.Marshal(&protocol.Permission{
				Name:     "testint.perm.schedule",
				Internal: true,
			})

			authRealmGlobals := testAuthRealmsGlobals(ctx)
			authRealmGlobals.PermissionsList = &permissions.PermissionsList{
				Permissions: []*protocol.Permission{
					{
						Name:     "test.perm.create",
						Internal: false,
					},
					{
						Name:     "testint.perm.schedule",
						Internal: true,
					},
					{
						Name:     "testtwo.perm.delete",
						Internal: false,
					},
				},
			}
			// This is a little inconsistent with how exactly it'll be in datastore but
			// since we can't get the exact format that the Python version got, this is
			// the closest we can get which is what it looks like after it's been unmarshalled
			// from Python.
			authRealmGlobals.Permissions = []string{
				string(permCreate),
				string(permTwoCreate),
				string(permIntSched),
			}

			So(datastore.Put(ctx, authRealmGlobals), ShouldBeNil)

			permsCfg := &configspb.PermissionsConfig{
				Role: []*configspb.PermissionsConfig_Role{
					{
						Name: "role/test.role",
						Permissions: []*protocol.Permission{
							{
								Name: "test.perm.create",
							},
						},
						Includes: []string{},
					},
					{
						Name: "role/test.role.two",
						Permissions: []*protocol.Permission{
							{
								Name: "testtwo.perm.delete",
							},
						},
					},
					{
						Name: "role/luci.internal.testint.role",
						Permissions: []*protocol.Permission{
							{
								Name:     "testint.perm.schedule",
								Internal: true,
							},
						},
					},
				},
			}

			So(UpdateAuthRealmsGlobals(ctx, permsCfg, false, "Go pRPC API"), ShouldBeNil)
			So(ts.Tasks(), ShouldHaveLength, 0)

			fetched, err := GetAuthRealmsGlobals(ctx)
			So(err, ShouldBeNil)
			So(fetched.PermissionsList.GetPermissions(), ShouldResembleProto,
				[]*protocol.Permission{
					{
						Name:     "test.perm.create",
						Internal: false,
					},
					{
						Name:     "testint.perm.schedule",
						Internal: true,
					},
					{
						Name:     "testtwo.perm.delete",
						Internal: false,
					},
				})
		})
	})
}

func TestGetAuthDBSnapshot(t *testing.T) {
	t.Parallel()
	ctx := memory.Use(context.Background())

	Convey("Testing GetAuthDBSnapshot", t, func() {
		_, err := GetAuthDBSnapshot(ctx, 42, false)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		snapshot := testAuthDBSnapshot(ctx, 42)

		err = datastore.Put(ctx, snapshot)
		So(err, ShouldBeNil)

		actual, err := GetAuthDBSnapshot(ctx, 42, false)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, snapshot)

		err = datastore.Put(ctx, snapshot)
		So(err, ShouldBeNil)

		actual, err = GetAuthDBSnapshot(ctx, 42, false)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, snapshot)
	})

	Convey("Testing GetAuthDBSnapshotLatest", t, func() {
		_, err := GetAuthDBSnapshotLatest(ctx)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		snapshot := testAuthDBSnapshot(ctx, 42)

		authDBSnapshotLatest := &AuthDBSnapshotLatest{
			Kind:         "AuthDBSnapshotLatest",
			ID:           "latest",
			AuthDBRev:    snapshot.ID,
			AuthDBSha256: snapshot.AuthDBSha256,
			ModifiedTS:   testModifiedTS,
		}
		err = datastore.Put(ctx, authDBSnapshotLatest)

		So(err, ShouldBeNil)

		actual, err := GetAuthDBSnapshotLatest(ctx)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, authDBSnapshotLatest)
	})

	Convey("Testing unshardAuthDB", t, func() {
		authDBShard1 := &AuthDBShard{
			Kind: "AuthDBShard",
			ID:   "42:7F404D83A3F440591C25A09B0A471EC4BB7D4EA3B50C081BCE37AA879E15EB69",
			Blob: []byte("half-1"),
		}
		authDBShard2 := &AuthDBShard{
			Kind: "AuthDBShard",
			ID:   "42:55915DB56DAD50F22BD882DACEE545FEFCA583CB8B3DACC4E5D9CAC9A4A2460C",
			Blob: []byte("half-2"),
		}
		shardIDs := []string{authDBShard1.ID, authDBShard2.ID}

		expectedBlob := []byte("half-1half-2")
		So(datastore.Put(ctx, authDBShard1), ShouldBeNil)
		So(datastore.Put(ctx, authDBShard2), ShouldBeNil)

		actualBlob, err := unshardAuthDB(ctx, shardIDs)
		So(err, ShouldBeNil)
		So(actualBlob, ShouldResemble, expectedBlob)
	})

	Convey("Testing GetAuthDBSnapshot with sharded DB", t, func() {
		snapshot, expectedAuthDB, err := testAuthDBSnapshotSharded(ctx, 42, 3)
		So(err, ShouldBeNil)
		So(datastore.Put(ctx, snapshot), ShouldBeNil)

		actualSnapshot, err := GetAuthDBSnapshot(ctx, 42, false)
		So(err, ShouldBeNil)
		So(actualSnapshot.AuthDBDeflated, ShouldResemble, expectedAuthDB)

		actualSnapshot, err = GetAuthDBSnapshot(ctx, 42, true)
		So(err, ShouldBeNil)
		So(actualSnapshot.AuthDBDeflated, ShouldBeNil)
	})
}

func TestProtoConversion(t *testing.T) {
	t.Parallel()

	Convey("AuthGroup FromProto and ToProto round trip equivalence", t, func() {
		ctx := memory.Use(context.Background())

		empty := &AuthGroup{
			Kind:   "AuthGroup",
			Parent: RootKey(ctx),
		}

		So(AuthGroupFromProto(ctx, empty.ToProto(true)), ShouldResemble, empty)

		g := testAuthGroup(ctx, "foo-group")
		// Ignore the versioned entity mixin since this doesn't survive the proto conversion round trip.
		g.AuthVersionedEntityMixin = AuthVersionedEntityMixin{}

		So(AuthGroupFromProto(ctx, g.ToProto(true)), ShouldResemble, g)
	})
}
