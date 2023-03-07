/*-
 * Copyright (c) 2014-2023
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
	"database/sql"
	"errors"
	"strconv"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/templates"

	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type StartPage struct {
	DB *ddao.DB
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

func writeBuildListAll(ctx context.Context, w http.ResponseWriter, builds []ddao.Build) {
	templates.TableBegin(w, "Date", "Branch", "Platform", "Stats", "User")
	for i := range builds {
		templates.TableBuilds(w, &builds[i])
	}
	io.WriteString(w, templates.TableEnd)
}

// writePackageList writes a table of package results from the iterator it to w.
func writePackageList(ctx context.Context, w http.ResponseWriter, rows []ddao.GetResultsInCategoryRow) {
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	templates.TablePkgs(w, rows)
	io.WriteString(w, templates.TableEnd)
}

type BuildDetails struct {
	DB *ddao.DB
}

func (b *BuildDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/build/"), "/")
	if len(paths) == 0 {
		return
	}
	buildID, err := strconv.ParseInt(paths[0], 10, 64)
	if err != nil {
		log.Warningf(ctx, "error decoding key: %s", err)
		return
	}

	build, err := b.DB.GetBuild(ctx, buildID)
	if err != nil {
		log.Warningf(ctx, "getting build record: %s", err)
		return
	}
	templates.BulkBuildInfo(w, &build)
	switch r.URL.Query().Get("a") {
	case "reindex":
		// ingest.FetchReport(ctx, key, build.ReportURL)
		io.WriteString(w, templates.ReindexOK)
		return
	case "delete":
		// delete.DeleteBuildDetails.Call(ctx, key)
	}

	templates.DataTable(w, `"order": [3, "desc"]`)

	if len(paths) > 1 {
		category := paths[1] + "/"
		results, err := b.DB.GetResultsInCategory(ctx, ddao.GetResultsInCategoryParams{
			BuildID: sql.NullInt64{
				Int64: buildID,
				Valid: true,
			},
			Category: category,
		})
		if err != nil {
			log.Errorf(ctx, "GetResultsInCategory: %v", err)
		}
		templates.Heading(w, category)
		writePackageList(ctx, w, results)
		return
	}

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

type PkgDetails struct {
	DB *ddao.DB
}

func (p *PkgDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/pkg/"), "/")
	if len(paths) == 0 {
		return
	}
	resultID, err := strconv.ParseInt(paths[0], 10, 64)
	if err != nil {
		log.Warningf(ctx, "error decoding key: %s", err)
		return
	}

	res, err := p.DB.GetSingleResult(ctx, resultID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Errorf(ctx, "GetSingleResult: %v", err)
		}
		return
	}

	templates.PkgInfo(w, res)
	templates.DataTable(w, "")

	// Failed, breaking other packages.
	if res.Breaks > 0 {
		fmt.Fprintf(w, "<h2>This package breaks %d others:</h2>", res.Breaks)
		//it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("FailedDeps =", p.PkgName).Order("Category").Order("Dir").Limit(1000).Run(ctx)
		//writePackageList(ctx, w, it)
	}

	// Failed to build because of dependencies.
	if res.FailedDeps == "" {
		return
	}
	failedDeps := strings.Split(res.FailedDeps, ",")
	fmt.Fprintf(w, "<h2>This package has %d failed dependencies:</h2>", len(failedDeps))
	// // TODO(bsiegert) Unfortunately, we save a list of package names, not a
	// // list of corresponding datastore keys. So we need to fetch them one by
	// // one.
	// templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	// dp := &bulk.Pkg{}
	// for _, dep := range p.FailedDeps {
	// 	it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("PkgName =", dep).Limit(1).Run(ctx)
	// 	key, err := it.Next(dp)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	dp.Key = key.Encode()
	// 	templates.TablePkgs(w, dp)
	// }
	// io.WriteString(w, templates.TableEnd)

}
