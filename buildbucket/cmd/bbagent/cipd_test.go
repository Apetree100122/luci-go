// Copyright 2022 The LUCI Authors.
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/logging/memlogger"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
)

var successResult = &cipdOut{
	Result: map[string][]*cipdPkg{
		"path_a":        {{Package: "pkg_a", InstanceID: "instance_a"}},
		"path_b":        {{Package: "pkg_b", InstanceID: "instance_b"}},
		kitchenCheckout: {{Package: "package", InstanceID: "instance_k"}},
	},
}

var testCase string

// fakeExecCommand mocks exec Command. It will trigger TestHelperProcess to
// return the right mocked output.
func fakeExecCommand(_ context.Context, command string, args ...string) *exec.Cmd {
	os.Environ()
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	tc := "TEST_CASE=" + testCase
	fakeResultsFilePath := "RESULTS_FILE=" + resultsFilePath
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", tc, fakeResultsFilePath}
	return cmd
}

// TestHelperProcess produces fake outputs based on the "TEST_CASE" env var when
// executed with the env var "GO_WANT_HELPER_PROCESS" set to 1.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	// check if it's a `cipd ensure` command.
	if !(args[0] == "cipd" && args[1] == "ensure") {
		fmt.Fprintf(os.Stderr, "Not a cipd ensure command: %s\n", args)
		os.Exit(1)
	}
	switch os.Getenv("TEST_CASE") {
	case "success":
		// Mock the generated json file of `cipd ensure` command.
		jsonRs, _ := json.Marshal(successResult)
		if err := os.WriteFile(os.Getenv("RESULTS_FILE"), jsonRs, 0666); err != nil {
			fmt.Fprintf(os.Stderr, "Errors in preparing data for tests\n")
		}

	case "failure":
		os.Exit(1)
	}
}

func TestPrependPath(t *testing.T) {
	originalPathEnv := os.Getenv("PATH")
	Convey("prependPath", t, func() {
		defer func() {
			_ = os.Setenv("PATH", originalPathEnv)
		}()

		build := &bbpb.Build{
			Id: 123,
			Infra: &bbpb.BuildInfra{
				Buildbucket: &bbpb.BuildInfra_Buildbucket{
					Agent: &bbpb.BuildInfra_Buildbucket_Agent{
						Input: &bbpb.BuildInfra_Buildbucket_Agent_Input{
							Data: map[string]*bbpb.InputDataRef{
								"path_a": {
									DataType: &bbpb.InputDataRef_Cipd{
										Cipd: &bbpb.InputDataRef_CIPD{
											Specs: []*bbpb.InputDataRef_CIPD_PkgSpec{{Package: "pkg_a", Version: "latest"}},
										},
									},
									OnPath: []string{"path_a/bin", "path_a"},
								},
								"path_b": {
									DataType: &bbpb.InputDataRef_Cas{
										Cas: &bbpb.InputDataRef_CAS{
											CasInstance: "projects/project/instances/instance",
											Digest: &bbpb.InputDataRef_CAS_Digest{
												Hash:      "hash",
												SizeBytes: 1,
											},
										},
									},
									OnPath: []string{"path_b/bin", "path_b"},
								},
							},
						},
						Output: &bbpb.BuildInfra_Buildbucket_Agent_Output{},
					},
				},
			},
			Input: &bbpb.Build_Input{
				Experiments: []string{"luci.buildbucket.agent.cipd_installation"},
			},
		}

		cwd, err := os.Getwd()
		So(err, ShouldBeNil)
		So(prependPath(build, cwd), ShouldBeNil)
		pathEnv := os.Getenv("PATH")
		var expectedPath []string
		for _, p := range []string{"path_a", "path_a/bin", "path_b", "path_b/bin"} {
			expectedPath = append(expectedPath, filepath.Join(cwd, p))
		}
		So(strings.Contains(pathEnv, strings.Join(expectedPath, string(os.PathListSeparator))), ShouldBeTrue)
	})
}

func TestInstallCipdPackages(t *testing.T) {
	t.Parallel()
	resultsFilePath = filepath.Join(t.TempDir(), "cipd_ensure_results.json")
	Convey("installCipdPackages", t, func() {
		ctx := memory.Use(context.Background())
		ctx = memlogger.Use(ctx)
		execCommandContext = fakeExecCommand
		defer func() { execCommandContext = exec.CommandContext }()

		build := &bbpb.Build{
			Id: 123,
			Infra: &bbpb.BuildInfra{
				Buildbucket: &bbpb.BuildInfra_Buildbucket{
					Agent: &bbpb.BuildInfra_Buildbucket_Agent{
						Input: &bbpb.BuildInfra_Buildbucket_Agent_Input{
							Data: map[string]*bbpb.InputDataRef{
								"path_a": {
									DataType: &bbpb.InputDataRef_Cipd{
										Cipd: &bbpb.InputDataRef_CIPD{
											Specs: []*bbpb.InputDataRef_CIPD_PkgSpec{{Package: "pkg_a", Version: "latest"}},
										},
									},
									OnPath: []string{"path_a/bin", "path_a"},
								},
								"path_b": {
									DataType: &bbpb.InputDataRef_Cipd{
										Cipd: &bbpb.InputDataRef_CIPD{
											Specs: []*bbpb.InputDataRef_CIPD_PkgSpec{{Package: "pkg_b", Version: "latest"}},
										},
									},
									OnPath: []string{"path_b/bin", "path_b"},
								},
							},
						},
						Output: &bbpb.BuildInfra_Buildbucket_Agent_Output{},
					},
				},
			},
			Input: &bbpb.Build_Input{
				Experiments: []string{"luci.buildbucket.agent.cipd_installation"},
			},
		}

		Convey("success", func() {
			testCase = "success"
			cwd, err := os.Getwd()
			So(err, ShouldBeNil)
			So(installCipdPackages(ctx, build, cwd), ShouldBeNil)
			So(build.Infra.Buildbucket.Agent.Output.ResolvedData["path_a"], ShouldResembleProto, &bbpb.ResolvedDataRef{
				DataType: &bbpb.ResolvedDataRef_Cipd{
					Cipd: &bbpb.ResolvedDataRef_CIPD{
						Specs: []*bbpb.ResolvedDataRef_CIPD_PkgSpec{{Package: successResult.Result["path_a"][0].Package, Version: successResult.Result["path_a"][0].InstanceID}},
					},
				},
			})
			So(build.Infra.Buildbucket.Agent.Output.ResolvedData["path_b"], ShouldResembleProto, &bbpb.ResolvedDataRef{
				DataType: &bbpb.ResolvedDataRef_Cipd{
					Cipd: &bbpb.ResolvedDataRef_CIPD{
						Specs: []*bbpb.ResolvedDataRef_CIPD_PkgSpec{{Package: successResult.Result["path_b"][0].Package, Version: successResult.Result["path_b"][0].InstanceID}},
					},
				},
			})
		})

		Convey("failure", func() {
			testCase = "failure"
			err := installCipdPackages(ctx, build, ".")
			So(build.Infra.Buildbucket.Agent.Output.ResolvedData, ShouldBeNil)
			So(err, ShouldErrLike, "Failed to run cipd ensure command")
		})

		Convey("handle kitchenCheckout", func() {
			Convey("kitchenCheckout not in agent input", func() {
				testCase = "success"
				build.Exe = &bbpb.Executable{
					CipdPackage: "package",
					CipdVersion: "version",
				}
				cwd, err := os.Getwd()
				So(err, ShouldBeNil)
				So(installCipdPackages(ctx, build, cwd), ShouldBeNil)
				So(build.Infra.Buildbucket.Agent.Purposes[kitchenCheckout], ShouldEqual, bbpb.BuildInfra_Buildbucket_Agent_PURPOSE_EXE_PAYLOAD)
				So(build.Infra.Buildbucket.Agent.Output.ResolvedData[kitchenCheckout], ShouldResembleProto, &bbpb.ResolvedDataRef{
					DataType: &bbpb.ResolvedDataRef_Cipd{
						Cipd: &bbpb.ResolvedDataRef_CIPD{
							Specs: []*bbpb.ResolvedDataRef_CIPD_PkgSpec{{Package: successResult.Result[kitchenCheckout][0].Package, Version: successResult.Result[kitchenCheckout][0].InstanceID}},
						},
					},
				})
			})
			Convey("kitchenCheckout in agent input", func() {
				testCase = "success"
				build.Exe = &bbpb.Executable{
					CipdPackage: "package",
					CipdVersion: "version",
				}
				build.Infra.Buildbucket.Agent.Input.Data[kitchenCheckout] = &bbpb.InputDataRef{
					DataType: &bbpb.InputDataRef_Cipd{
						Cipd: &bbpb.InputDataRef_CIPD{
							Specs: []*bbpb.InputDataRef_CIPD_PkgSpec{{Package: "package", Version: "version"}},
						},
					},
				}
				build.Infra.Buildbucket.Agent.Purposes = map[string]bbpb.BuildInfra_Buildbucket_Agent_Purpose{
					kitchenCheckout: bbpb.BuildInfra_Buildbucket_Agent_PURPOSE_EXE_PAYLOAD,
				}
				cwd, err := os.Getwd()
				So(err, ShouldBeNil)
				So(installCipdPackages(ctx, build, cwd), ShouldBeNil)
				So(build.Infra.Buildbucket.Agent.Purposes[kitchenCheckout], ShouldEqual, bbpb.BuildInfra_Buildbucket_Agent_PURPOSE_EXE_PAYLOAD)
				So(build.Infra.Buildbucket.Agent.Output.ResolvedData[kitchenCheckout], ShouldResembleProto, &bbpb.ResolvedDataRef{
					DataType: &bbpb.ResolvedDataRef_Cipd{
						Cipd: &bbpb.ResolvedDataRef_CIPD{
							Specs: []*bbpb.ResolvedDataRef_CIPD_PkgSpec{{Package: successResult.Result[kitchenCheckout][0].Package, Version: successResult.Result[kitchenCheckout][0].InstanceID}},
						},
					},
				})
			})
		})
	})
}
