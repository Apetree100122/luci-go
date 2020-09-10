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

package casviewer

import (
	"context"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/appengine"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/realms"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"
)

// Path to the templates dir from the executable.
const templatePath = "templates"

var permMintToken = realms.RegisterPermission("luci.serviceAccounts.mintToken")

// InstallHandlers install CAS Viewer handlers to the router.
func InstallHandlers(r *router.Router, cc *ClientCache) {
	baseMW := router.NewMiddlewareChain(
		templates.WithTemplates(getTemplateBundle()),
	)
	blobMW := baseMW.Extend(
		checkPermission,
		withClientCacheMW(cc),
	)

	r.GET("/", baseMW, rootHandler)
	r.GET("/projects/:project/instances/:instance/blobs/:hash/:size/tree", blobMW, treeHandler)
	r.GET("/projects/:project/instances/:instance/blobs/:hash/:size", blobMW, getHandler)
}

// getTemplateBundles returns template Bundle with base args.
func getTemplateBundle() *templates.Bundle {
	return &templates.Bundle{
		Loader:          templates.FileSystemLoader(templatePath),
		DefaultTemplate: "base",
		DefaultArgs: func(c context.Context, e *templates.Extra) (templates.Args, error) {
			return templates.Args{
				"AppVersion": strings.Split(appengine.VersionID(c), ".")[0],
				"User":       auth.CurrentUser(c),
			}, nil
		},
		FuncMap: template.FuncMap{
			"treeURL": treeURL,
			"getURL":  getURL,
		},
	}
}

// checkPermission checks if the user has permission to read the blob.
func checkPermission(c *router.Context, next router.Handler) {
	switch ok, err := auth.HasPermission(c.Context, permMintToken, readOnlyRealm(c.Params)); {
	case err != nil:
		renderErrorPage(c.Context, c.Writer, err)
	case !ok:
		err = errors.New("permission denied", grpcutil.PermissionDeniedTag)
		renderErrorPage(c.Context, c.Writer, err)
	default:
		next(c)
	}
}

// rootHandler renders top page.
func rootHandler(c *router.Context) {
	templates.MustRender(c.Context, c.Writer, "pages/index.html", nil)
}

func treeHandler(c *router.Context) {
	inst := fullInstName(c.Params)
	cl, err := GetClient(c.Context, inst)
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
		return
	}
	bd, err := blobDigest(c.Params)
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
		return
	}
	err = renderTree(c.Context, c.Writer, cl, bd, inst)
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
	}
}

func getHandler(c *router.Context) {
	cl, err := GetClient(c.Context, fullInstName(c.Params))
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
		return
	}
	bd, err := blobDigest(c.Params)
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
		return
	}
	err = returnBlob(c.Context, c.Writer, cl, bd)
	if err != nil {
		renderErrorPage(c.Context, c.Writer, err)
	}
}

func readOnlyRealm(p httprouter.Params) string {
	return fmt.Sprintf("@internal:%s/cas-read-only", p.ByName("project"))
}

// fullInstName constructs full instance name from the URL parameters.
func fullInstName(p httprouter.Params) string {
	return fmt.Sprintf(
		"projects/%s/instances/%s", p.ByName("project"), p.ByName("instance"))
}

// blobDigest constructs a Digest from the URL parameters.
func blobDigest(p httprouter.Params) (*digest.Digest, error) {
	size, err := strconv.ParseInt(p.ByName("size"), 10, 64)
	if err != nil {
		err = errors.Annotate(err, "Digest size must be number").Tag(grpcutil.InvalidArgumentTag).Err()
		return nil, err
	}

	return &digest.Digest{
		Hash: p.ByName("hash"),
		Size: size,
	}, nil
}
