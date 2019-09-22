// Copyright 2016 The LUCI Authors.
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

package authdb

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/server/auth/internal"
	"go.chromium.org/luci/server/auth/service/protocol"
	"go.chromium.org/luci/server/auth/signing"
	"go.chromium.org/luci/server/auth/signing/signingtest"
	"go.chromium.org/luci/server/caching"

	"go.chromium.org/luci/server/auth/authdb/internal/oauthid"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSnapshotDB(t *testing.T) {
	c := context.Background()

	securityConfig, _ := proto.Marshal(&protocol.SecurityConfig{
		InternalServiceRegexp: []string{
			`(.*-dot-)?i1\.example\.com`,
			`(.*-dot-)?i2\.example\.com`,
		},
	})

	db, err := NewSnapshotDB(&protocol.AuthDB{
		OauthClientId: "primary-client-id",
		OauthAdditionalClientIds: []string{
			"additional-client-id-1",
			"additional-client-id-2",
		},
		TokenServerUrl: "http://token-server",
		Groups: []*protocol.AuthGroup{
			{
				Name:    "direct",
				Members: []string{"user:abc@example.com"},
			},
			{
				Name:  "via glob",
				Globs: []string{"user:*@example.com"},
			},
			{
				Name:   "via nested",
				Nested: []string{"direct"},
			},
			{
				Name:   "cycle",
				Nested: []string{"cycle"},
			},
			{
				Name:   "unknown nested",
				Nested: []string{"unknown"},
			},
		},
		IpWhitelistAssignments: []*protocol.AuthIPWhitelistAssignment{
			{
				Identity:    "user:abc@example.com",
				IpWhitelist: "whitelist",
			},
		},
		IpWhitelists: []*protocol.AuthIPWhitelist{
			{
				Name: "whitelist",
				Subnets: []string{
					"1.2.3.4/32",
					"10.0.0.0/8",
				},
			},
			{
				Name: "empty",
			},
		},
		SecurityConfig: securityConfig,
	}, "http://auth-service", 1234, false)
	if err != nil {
		panic(err)
	}

	Convey("IsAllowedOAuthClientID works", t, func() {
		call := func(email, clientID string) bool {
			res, err := db.IsAllowedOAuthClientID(c, email, clientID)
			So(err, ShouldBeNil)
			return res
		}

		So(call("abc@appspot.gserviceaccount.com", "anonymous"), ShouldBeTrue)
		So(call("dude@example.com", ""), ShouldBeFalse)
		So(call("dude@example.com", oauthid.GoogleAPIExplorerClientID), ShouldBeTrue)
		So(call("dude@example.com", "primary-client-id"), ShouldBeTrue)
		So(call("dude@example.com", "additional-client-id-2"), ShouldBeTrue)
		So(call("dude@example.com", "unknown-client-id"), ShouldBeFalse)
	})

	Convey("IsInternalService works", t, func() {
		call := func(hostname string) bool {
			res, err := db.IsInternalService(c, hostname)
			So(err, ShouldBeNil)
			return res
		}

		So(call("i1.example.com"), ShouldBeTrue)
		So(call("i2.example.com"), ShouldBeTrue)
		So(call("abc-dot-i1.example.com"), ShouldBeTrue)
		So(call("external.example.com"), ShouldBeFalse)
		So(call("something-i1.example.com"), ShouldBeFalse)
		So(call("i1.example.com-something"), ShouldBeFalse)
	})

	Convey("IsMember works", t, func() {
		call := func(ident string, groups ...string) bool {
			res, err := db.IsMember(c, identity.Identity(ident), groups)
			So(err, ShouldBeNil)
			return res
		}

		So(call("user:abc@example.com", "direct"), ShouldBeTrue)
		So(call("user:another@example.com", "direct"), ShouldBeFalse)

		So(call("user:abc@example.com", "via glob"), ShouldBeTrue)
		So(call("user:abc@another.com", "via glob"), ShouldBeFalse)

		So(call("user:abc@example.com", "via nested"), ShouldBeTrue)
		So(call("user:another@example.com", "via nested"), ShouldBeFalse)

		So(call("user:abc@example.com", "cycle"), ShouldBeFalse)
		So(call("user:abc@example.com", "unknown"), ShouldBeFalse)
		So(call("user:abc@example.com", "unknown nested"), ShouldBeFalse)

		So(call("user:abc@example.com"), ShouldBeFalse)
		So(call("user:abc@example.com", "unknown", "direct"), ShouldBeTrue)
		So(call("user:abc@example.com", "via glob", "direct"), ShouldBeTrue)
	})

	Convey("CheckMembership works", t, func() {
		call := func(ident string, groups ...string) []string {
			res, err := db.CheckMembership(c, identity.Identity(ident), groups)
			So(err, ShouldBeNil)
			return res
		}

		So(call("user:abc@example.com", "direct"), ShouldResemble, []string{"direct"})
		So(call("user:another@example.com", "direct"), ShouldBeNil)

		So(call("user:abc@example.com", "via glob"), ShouldResemble, []string{"via glob"})
		So(call("user:abc@another.com", "via glob"), ShouldBeNil)

		So(call("user:abc@example.com", "via nested"), ShouldResemble, []string{"via nested"})
		So(call("user:another@example.com", "via nested"), ShouldBeNil)

		So(call("user:abc@example.com", "cycle"), ShouldBeNil)
		So(call("user:abc@example.com", "unknown"), ShouldBeNil)
		So(call("user:abc@example.com", "unknown nested"), ShouldBeNil)

		So(call("user:abc@example.com"), ShouldBeNil)
		So(call("user:abc@example.com", "unknown", "direct"), ShouldResemble, []string{"direct"})
		So(call("user:abc@example.com", "via glob", "direct"), ShouldResemble, []string{"via glob", "direct"})
	})

	Convey("GetCertificates works", t, func(c C) {
		tokenService := signingtest.NewSigner(&signing.ServiceInfo{
			AppID:              "token-server",
			ServiceAccountName: "token-server-account@example.com",
		})

		calls := 0

		ctx := context.Background()
		ctx = caching.WithEmptyProcessCache(ctx)

		ctx = internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
			calls++
			if r.URL.String() != "http://token-server/auth/api/v1/server/certificates" {
				return 404, "Wrong URL"
			}
			certs, err := tokenService.Certificates(ctx)
			if err != nil {
				panic(err)
			}
			blob, err := json.Marshal(certs)
			if err != nil {
				panic(err)
			}
			return 200, string(blob)
		})

		certs, err := db.GetCertificates(ctx, "user:token-server-account@example.com")
		So(err, ShouldBeNil)
		So(certs, ShouldNotBeNil)

		// Fetched one bundle.
		So(calls, ShouldEqual, 1)

		// For unknown signer returns (nil, nil).
		certs, err = db.GetCertificates(ctx, "user:unknown@example.com")
		So(err, ShouldBeNil)
		So(certs, ShouldBeNil)
	})

	Convey("IsInWhitelist works", t, func() {
		wl, err := db.GetWhitelistForIdentity(c, "user:abc@example.com")
		So(err, ShouldBeNil)
		So(wl, ShouldEqual, "whitelist")

		wl, err = db.GetWhitelistForIdentity(c, "user:unknown@example.com")
		So(err, ShouldBeNil)
		So(wl, ShouldEqual, "")

		call := func(ip, whitelist string) bool {
			ipaddr := net.ParseIP(ip)
			So(ipaddr, ShouldNotBeNil)
			res, err := db.IsInWhitelist(c, ipaddr, whitelist)
			So(err, ShouldBeNil)
			return res
		}

		So(call("1.2.3.4", "whitelist"), ShouldBeTrue)
		So(call("10.255.255.255", "whitelist"), ShouldBeTrue)
		So(call("9.255.255.255", "whitelist"), ShouldBeFalse)
		So(call("1.2.3.4", "empty"), ShouldBeFalse)
	})

	Convey("Revision works", t, func() {
		So(Revision(&SnapshotDB{Rev: 123}), ShouldEqual, 123)
		So(Revision(ErroringDB{}), ShouldEqual, 0)
		So(Revision(nil), ShouldEqual, 0)
	})

	Convey("SnapshotDBFromTextProto works", t, func() {
		db, err := SnapshotDBFromTextProto(strings.NewReader(`
			groups {
				name: "group"
				members: "user:a@example.com"
			}
		`))
		So(err, ShouldBeNil)
		yes, err := db.IsMember(c, "user:a@example.com", []string{"group"})
		So(err, ShouldBeNil)
		So(yes, ShouldBeTrue)
	})

	Convey("SnapshotDBFromTextProto bad proto", t, func() {
		_, err := SnapshotDBFromTextProto(strings.NewReader(`
			groupz {}
		`))
		So(err, ShouldErrLike, "not a valid AuthDB text proto file")
	})

	Convey("SnapshotDBFromTextProto bad structure", t, func() {
		_, err := SnapshotDBFromTextProto(strings.NewReader(`
			groups {
				name: "group 1"
				nested: "group 2"
			}
			groups {
				name: "group 2"
				nested: "group 1"
			}
		`))
		So(err, ShouldErrLike, "dependency cycle found")
	})
}

func BenchmarkIsMember(b *testing.B) {
	c := context.Background()
	db, _ := NewSnapshotDB(&protocol.AuthDB{
		Groups: []*protocol.AuthGroup{
			{
				Name:   "outer",
				Nested: []string{"A", "B"},
			},
			{
				Name:   "A",
				Nested: []string{"A_A", "A_B"},
			},
			{
				Name:   "B",
				Nested: []string{"B_A", "B_B"},
			},
			{
				Name:   "A_A",
				Nested: []string{"A_A_A"},
			},
			{
				Name:   "A_A_A",
				Nested: []string{"A_A_A_A"},
			},
			{
				Name: "A_A_A_A",
			},
			{
				Name: "A_B",
			},
			{
				Name: "B_A",
			},
			{
				Name: "B_B",
			},
		},
	}, "http://auth-service", 1234, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.IsMember(c, "user:somedude@example.com", []string{"outer"})
	}
}
