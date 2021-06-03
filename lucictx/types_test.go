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

package lucictx

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestPredefinedTypes(t *testing.T) {
	t.Parallel()

	Convey("Test predefined types", t, func() {
		c := context.Background()
		Convey("local_auth", func() {
			So(GetLocalAuth(c), ShouldBeNil)

			localAuth := &LocalAuth{
				RpcPort: 100,
				Secret:  []byte("foo"),
				Accounts: []*LocalAuthAccount{
					{Id: "test", Email: "some@example.com"},
				},
				DefaultAccountId: "test",
			}

			c = SetLocalAuth(c, localAuth)
			data := getCurrent(c).sections["local_auth"]
			var v interface{}
			So(json.Unmarshal(*data, &v), ShouldBeNil)
			So(v, ShouldResemble, map[string]interface{}{
				"accounts": []interface{}{map[string]interface{}{
					"email": "some@example.com",
					"id":    "test",
				}},
				"default_account_id": "test",
				"secret":             "Zm9v",
				"rpc_port":           100.0,
			})

			So(GetLocalAuth(c), ShouldResembleProto, localAuth)
		})

		Convey("swarming", func() {
			So(GetSwarming(c), ShouldBeNil)

			c = SetSwarming(c, &Swarming{SecretBytes: []byte("foo")})
			data, _ := getCurrent(c).sections["swarming"]
			So(string(*data), ShouldEqual, `{"secret_bytes":"Zm9v"}`)

			So(GetSwarming(c), ShouldResembleProto, &Swarming{SecretBytes: []byte("foo")})
		})

		Convey("resultdb", func() {
			So(GetResultDB(c), ShouldBeNil)

			resultdb := &ResultDB{
				Hostname: "test.results.cr.dev",
				CurrentInvocation: &ResultDBInvocation{
					Name:        "invocations/build:1",
					UpdateToken: "foobarbazsecretoken",
				}}
			c = SetResultDB(c, resultdb)
			data := getCurrent(c).sections["resultdb"]
			var v interface{}
			So(json.Unmarshal(*data, &v), ShouldBeNil)
			So(v, ShouldResemble, map[string]interface{}{
				"current_invocation": map[string]interface{}{
					"name":         "invocations/build:1",
					"update_token": "foobarbazsecretoken",
				},
				"hostname": "test.results.cr.dev",
			})

			So(GetResultDB(c), ShouldResembleProto, resultdb)
		})

		Convey("realm", func() {
			So(GetRealm(c), ShouldBeNil)

			r := &Realm{
				Name: "test:realm",
			}
			c = SetRealm(c, r)
			data, _ := getCurrent(c).sections["realm"]
			So(string(*data), ShouldEqual, `{"name":"test:realm"}`)
			So(GetRealm(c), ShouldResembleProto, r)
			proj, realm := CurrentRealm(c)
			So(proj, ShouldEqual, "test")
			So(realm, ShouldEqual, "realm")
		})
	})
}
