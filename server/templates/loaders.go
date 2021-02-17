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

package templates

import (
	"context"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// AssetsLoader returns Loader that loads templates from the given assets map.
//
// The directory with templates is expected to have special structure:
//  * 'pages/' contain all top-level templates that will be loaded
//  * 'includes/' contain templates that will be associated with every top-level template from 'pages/'.
//  * 'widgets/' contain all standalone templates that will be loaded without templates from 'includes/'
// Only templates from 'pages/' and 'widgets/' are included in the output map.
func AssetsLoader(assets map[string]string) Loader {
	return func(c context.Context, funcMap template.FuncMap) (map[string]*template.Template, error) {
		// Pick all includes.
		includes := []string(nil)
		for name, body := range assets {
			if strings.HasPrefix(name, "includes/") {
				includes = append(includes, body)
			}
		}

		// Parse all top level templates and associate them with all includes.
		toplevel := map[string]*template.Template{}
		for name, body := range assets {
			if strings.HasPrefix(name, "pages/") {
				t := template.New(name).Funcs(funcMap)
				// TODO(vadimsh): There's probably a way to avoid reparsing includes
				// all the time.
				for _, includeSrc := range includes {
					if _, err := t.Parse(includeSrc); err != nil {
						return nil, err
					}
				}

				if _, err := t.Parse(body); err != nil {
					return nil, err
				}
				toplevel[name] = t
			}
		}

		for name, body := range assets {
			if strings.HasPrefix(name, "widgets/") {
				t := template.New(name).Funcs(funcMap)
				if _, err := t.Parse(body); err != nil {
					return nil, err
				}
				toplevel[name] = t
			}
		}

		return toplevel, nil
	}
}

// FileSystemLoader returns Loader that loads templates from file system.
//
// The directory with templates is expected to have special structure:
//  * 'pages/' contain all top-level templates that will be loaded
//  * 'includes/' contain templates that will be associated with every top-level template from 'pages/'.
//  * 'widgets/' contain all standalone templates that will be loaded without templates from 'includes/'
// Only templates from 'pages/' and 'widgets/' are included in the output map.
func FileSystemLoader(path string) Loader {
	return func(c context.Context, funcMap template.FuncMap) (map[string]*template.Template, error) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		// Read all relevant files into memory. It's ok, they are small.
		files := map[string]string{}
		if err = readFilesInDir(filepath.Join(abs, "includes"), files); err != nil {
			return nil, err
		}
		if err = readFilesInDir(filepath.Join(abs, "pages"), files); err != nil {
			return nil, err
		}
		if err = readFilesInDir(filepath.Join(abs, "widgets"), files); err != nil {
			return nil, err
		}

		// Convert to assets map as consumed by AssetsLoader (relative slash
		// separated paths) and reuse AssetsLoader.
		assets := map[string]string{}
		for k, v := range files {
			rel, err := filepath.Rel(abs, k)
			if err != nil {
				return nil, err
			}
			assets[filepath.ToSlash(rel)] = v
		}
		return AssetsLoader(assets)(c, funcMap)
	}
}

// readFilesInDir loads content of files into a map "file path" => content.
//
// Used only for small HTML templates, and thus it's fine to load them
// in memory.
//
// Does nothing is 'dir' is missing.
func readFilesInDir(dir string, out map[string]string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err == nil {
			out[path] = string(body)
		}
		return err
	})
}
