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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/templates"
)

var errNoArg = errors.New("no argument")

type StartPage struct {
	DB       *ddao.DB
	BasePath string
}

func (s *StartPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Try prepopulating the autocomplete cache early.
	// stateful.MaybePrefillCache(ctx)

	templates.PageHeader(w)
	defer templates.PageFooter(w)
	templates.StartPageLead(w)

	categories, _ := s.DB.GetCategories(ctx)
	if len(categories) > 0 {
		templates.CategoryList(w, categories, strings.TrimRight(s.BasePath, "/"))
	}

	templates.StartPageLead2(w)

	builds, err := s.DB.LatestBuilds(ctx, true)
	if err != nil {
		log.Errorf(ctx, "failed to read latest builds: %s", err)
	}

	if len(builds) == 0 {
		templates.DatastoreError(w, err)
	}
	writeBuildListAll(ctx, w, builds)
	templates.DataTable(w, nil, `"order": [0, "desc"]`)
}

func ShowBuilds(w http.ResponseWriter, r *http.Request) {
	templates.PageHeader(w)
	defer templates.PageFooter(w)
	templates.Heading(w, "List of Builds")

	templates.TableBegin(w, "Date", "Branch", "Platform", "Stats", "User")
	templates.TableEnd(w)
	templates.LoadScript(w, "builds.js")
}

func writeBuildListAll(ctx context.Context, w http.ResponseWriter, builds []ddao.Build) {
	templates.TableBegin(w, "Date", "Branch", "Platform", "Stats", "User")
	for i := range builds {
		templates.TableBuilds(w, &builds[i])
	}
	templates.TableEnd(w)
}

// writePackageList writes a table of package results from the list of rows to w.
func writePackageList(ctx context.Context, w http.ResponseWriter, rows []ddao.GetResultsInCategoryRow) {
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	templates.TablePkgs(w, rows)
	templates.TableEnd(w)
}

type BuildDetails struct {
	DB *ddao.DB
}

func (b *BuildDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templates.PageHeader(w)
	defer templates.PageFooter(w)

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
		templates.ReindexOK(w)
		return
	case "delete":
		// delete.DeleteBuildDetails.Call(ctx, key)
	}

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
		templates.DataTable(w, nil, `"order": [0, "asc"]`)
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
	templates.Heading(w, "Results by Category")
	templates.CategoryList(w, categories, path.Join(templates.BasePath, r.URL.Path))

	templates.Heading(w, "Packages breaking most other packages")
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")
	templates.TableEnd(w)

	templates.LoadScript(w, "builddetails.js")
	templates.BuildDetailsInit(w, ".table", "pkgsbreakingmostothers", buildID)
}

type PkgDetails struct {
	DB *ddao.DB
}

func (PkgDetails) arg(r *http.Request) (int64, error) {
	_, arg, ok := strings.Cut(r.URL.Path, "/pkg/")
	if !ok || arg == "" {
		return 0, errNoArg
	}

	paths := strings.Split(arg, "/")
	if len(paths) == 0 {
		return 0, errNoArg
	}
	return strconv.ParseInt(paths[0], 10, 64)
}

func (p *PkgDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templates.PageHeader(w)
	defer templates.PageFooter(w)

	resultID, err := p.arg(r)
	if err != nil {
		if err != errNoArg {
			log.Warningf(ctx, "failed to decode result ID: %v", err)
		}
		return
	}

	db, cancel, err := p.DB.BeginReadOnlyTransaction(ctx)
	if err != nil {
		log.Errorf(ctx, "starting r/o transaction: %v", err)
		return
	}
	defer cancel()

	res, err := db.GetSingleResult(ctx, resultID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Errorf(ctx, "GetSingleResult: %v", err)
		}
		return
	}

	templates.PkgInfo(w, res)
	// templates.DataTable(w, "")

	// Failed, breaking other packages.
	if res.Breaks > 0 {
		id := templates.ID("breaking")
		fmt.Fprintf(w, "<h2>This package breaks %d others</h2>", res.Breaks)
		templates.TableBeginID(w, id, "Location", "Package Name", "Status", "Breaks")
		templates.TableEnd(w)

		templates.LoadScript(w, "builddetails.js")
		templates.BuildDetailsInit(w, "#breaking", "pkgsbrokenby", resultID)
	}

	// Failed to build because of dependencies.
	if res.FailedDeps == "" {
		return
	}
	failedDeps := strings.Split(res.FailedDeps, ",")
	fmt.Fprintf(w, "<h2>This package has %d failed dependencies</h2>", len(failedDeps))
	templates.TableBegin(w, "Location", "Package Name", "Status", "Breaks")

	sqlBuildID := sql.NullInt64{
		Valid: true,
		Int64: res.BuildID,
	}
	rows := make([]ddao.GetSingleResultByPkgNameRow, 0, len(failedDeps))
	for _, dep := range failedDeps {
		row, err := db.GetSingleResultByPkgName(ctx, ddao.GetSingleResultByPkgNameParams{
			PkgName: dep,
			BuildID: sqlBuildID,
		})
		if err != nil {
			// Swallow and ignore error.
			continue
		}
		rows = append(rows, row)
	}
	templates.TablePkgs(w, rows)
	templates.TableEnd(w)
}

// Dirs is a handler for a subpage showing all the package directories for a given category.
type Dirs struct {
	DB         *ddao.DB
	PkgResults http.Handler
	BasePath   string
}

func (d *Dirs) args(r *http.Request) []string {
	return strings.Split(strings.Trim(r.URL.Path, "/"), "/")
}

func (d *Dirs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paths := d.args(r)
	if len(paths) == 0 {
		return
	}
	if len(paths) > 1 {
		d.PkgResults.ServeHTTP(w, r)
		return
	}
	templates.PageHeader(w)
	defer templates.PageFooter(w)

	category := paths[0] + "/"

	templates.Heading(w, category)

	dirs, err := d.DB.GetPkgsInCategory(ctx, category)
	if err != nil {
		log.Errorf(ctx, "Dirs: GetPkgsInCategory: %v", err)
	}

	templates.CategoryList(w, dirs, path.Join(d.BasePath, category))
}

// PkgResults is the package results page.
type PkgResults struct{}

func (PkgResults) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	templates.PageHeader(w)
	templates.PkgResults(w)
	templates.PageFooter(w)
}
