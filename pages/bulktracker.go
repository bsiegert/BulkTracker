/*-
 * Copyright (c) 2014-2022
 *      Benny Siegert <bsiegert@gmail.com>
 *
 * Provided that these terms and disclaimer and all copyright notices
 * are retained or reproduced in an accompanying document, permission
 * is granted to deal in this work without restriction, including un-
 * limited rights to use, publicly perform, distribute, sell, modify,
 * merge, give away, or sublicence.
 *
 * This work is provided "AS IS" and WITHOUT WARRANTY of any kind, to
 * the utmost extent permitted by applicable law, neither express nor
 * implied; without malicious intent or gross negligence. In no event
 * may a licensor, author or contributor be held liable for indirect,
 * direct, other damage, loss, or other issues arising in any way out
 * of dealing in the work, even if advised of the possibility of such
 * damage or existence of a defect, except proven that it results out
 * of said person's immediate fault when using the work as intended.
 */

// Package pages implements dynamic, server-side rendered pages for BulkTracker.
package pages

import (
	"strconv"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/dao"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/templates"

	"google.golang.org/appengine/datastore"

	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

type StartPage struct {
	DB *dao.DB
}

func (s *StartPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Try prepopulating the autocomplete cache early.
	// stateful.MaybePrefillCache(ctx)

	builds, err := s.DB.LatestBuilds(ctx, true)
	if err != nil {
		log.Errorf(ctx, "failed to read latest builds: %s", err)
	}

	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)
	io.WriteString(w, templates.StartPageLead)
	if len(builds) == 0 {
		templates.DatastoreError(w, err)
	}
	writeBuildListAll(ctx, w, builds)
	templates.DataTable(w, `"order": [0, "desc"]`)
}

func ShowBuilds(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)
	templates.Heading(w, "List of Builds")

	templates.TableBegin(w, "Date", "Branch", "Platform", "Stats", "User")
	io.WriteString(w, templates.TableEnd)
	io.WriteString(w, `<script src="/static/builds.js"></script>`)
}

func writeBuildListAll(ctx context.Context, w http.ResponseWriter, builds []bulk.Build) {
	templates.TableBegin(w, "Date", "Branch", "Platform", "Stats", "User")
	for i := range builds {
		templates.TableBuilds(w, &builds[i])
	}
	io.WriteString(w, templates.TableEnd)
}

// writePackageList writes a table of package results from the iterator it to w.
func writePackageList(ctx context.Context, w http.ResponseWriter, it *datastore.Iterator) {
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	p := &bulk.Pkg{}
	for {
		key, err := it.Next(p)
		if err == datastore.Done {
			break
		} else if err != nil {
			log.Errorf(ctx, "failed to read pkg: %s", err)
			w.WriteHeader(500)
			return
		}
		p.Key = key.Encode()
		templates.TablePkgs(w, p)
	}
	io.WriteString(w, templates.TableEnd)
}

type BuildDetails struct {
	DB *dao.DB
}

func (b *BuildDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/build/"), "/")
	if len(paths) == 0 {
		return
	}
	buildID, err := strconv.Atoi(paths[0])
	if err != nil {
		log.Warningf(ctx, "error decoding key: %s", err)
		return
	}

	build, err := b.DB.GetBuild(ctx, buildID)
	if err != nil {
		log.Warningf(ctx, "getting build record: %s", err)
		return
	}
	templates.BulkBuildInfo(w, build)
	switch r.URL.Query().Get("a") {
	case "reindex":
		// ingest.FetchReport(ctx, key, build.ReportURL)
		io.WriteString(w, templates.ReindexOK)
		return
	case "delete":
		// delete.DeleteBuildDetails.Call(ctx, key)
	}

	templates.DataTable(w, `"order": [3, "desc"]`)

	/*
		if len(paths) > 1 {
			category := paths[1] + "/"
			it := datastore.NewQuery("pkg").Ancestor(key).Filter("Category =", category).Order("Dir").Order("PkgName").Limit(10000).Run(ctx)
			templates.Heading(w, category)
			writePackageList(ctx, w, it)
			return
		}
	*/

	// NOTE: This used to be the list of categories for the current build.
	// Approximate by just showing all categories. The list of categories
	// does not change often.
	categories, _ := b.DB.GetCategories(ctx)
	if len(categories) == 0 {
		templates.NoDetails(w, r.URL.Path)
		return
	}
	templates.CategoryList(w, categories, r.URL.Path)

	/*
		templates.Heading(w, "Packages breaking most other packages")

		it := datastore.NewQuery("pkg").Ancestor(key).Filter("BuildStatus >", bulk.Prefailed).Order("BuildStatus").Order("-Breaks").Limit(100).Run(ctx)
		writePackageList(ctx, w, it)
	*/
}

func PkgDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)

	pkgKey, err := datastore.DecodeKey(path.Base(r.URL.Path))
	if err != nil {
		log.Warningf(ctx, "error decoding pkg key: %s", err)
		return
	}
	buildKey := pkgKey.Parent()

	p := &bulk.Pkg{}
	b := &bulk.Build{}
	if err = datastore.Get(ctx, pkgKey, p); err != nil {
		log.Warningf(ctx, "getting pkg record: %s", err)
		return
	}
	if buildKey != nil {
		if err = datastore.Get(ctx, buildKey, b); err != nil {
			log.Warningf(ctx, "getting build record: %s", err)
			return
		}
	}

	templates.PkgInfo(w, p, b)
	templates.DataTable(w, "")

	// Failed, breaking other packages.
	if p.Breaks > 0 {
		fmt.Fprintf(w, "<h2>This package breaks %d others:</h2>", p.Breaks)
		it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("FailedDeps =", p.PkgName).Order("Category").Order("Dir").Limit(1000).Run(ctx)
		writePackageList(ctx, w, it)
	}

	// Failed to build because of dependencies.
	if p.FailedDeps == nil {
		return
	}
	fmt.Fprintf(w, "<h2>This package has %d failed dependencies:</h2>", len(p.FailedDeps))
	// TODO(bsiegert) Unfortunately, we save a list of package names, not a
	// list of corresponding datastore keys. So we need to fetch them one by
	// one.
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	dp := &bulk.Pkg{}
	for _, dep := range p.FailedDeps {
		it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("PkgName =", dep).Limit(1).Run(ctx)
		key, err := it.Next(dp)
		if err != nil {
			continue
		}
		dp.Key = key.Encode()
		templates.TablePkgs(w, dp)
	}
	io.WriteString(w, templates.TableEnd)

}