/*-
 * Copyright (c) 2014-2019
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

package main

import (
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/data"
	"github.com/bsiegert/BulkTracker/ingest"
	"github.com/bsiegert/BulkTracker/json"
	"github.com/bsiegert/BulkTracker/templates"

	"context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

func main() {
	http.HandleFunc("/", StartPage)
	http.HandleFunc("/builds", ShowBuilds)
	http.HandleFunc("/build/", BuildDetails)
	http.HandleFunc("/pkg/", PkgDetails)
	http.HandleFunc("/_ah/mail/", ingest.HandleIncomingMail)

	for path, endpoint := range json.Mux {
		path = fmt.Sprintf("/json/%s/", path)
		http.Handle(path, endpoint)
	}
	appengine.Main()
}

func StartPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	builds, err := data.LatestBuilds(ctx)
	if err != nil {
		log.Errorf(ctx, "failed to read latest builds: %s", err)
		w.WriteHeader(500)
		return
	}

	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)
	io.WriteString(w, templates.StartPageLead)
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

func writeBuildListAll(c context.Context, w http.ResponseWriter, builds []bulk.Build) {
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

func BuildDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/build/"), "/")
	if len(paths) == 0 {
		return
	}
	key, err := datastore.DecodeKey(paths[0])
	if err != nil {
		log.Warningf(ctx, "error decoding key: %s", err)
		return
	}

	b := &bulk.Build{}
	err = datastore.Get(ctx, key, b)
	if err != nil {
		log.Warningf(ctx, "getting build record: %s", err)
		return
	}
	templates.BulkBuildInfo(w, b)
	if r.URL.Query().Get("a") == "reindex" {
		ingest.FetchReport.Call(ctx, key, b.ReportURL)
		io.WriteString(w, templates.ReindexOK)
		return
	}

	templates.DataTable(w, `"order": [3, "desc"]`)

	if len(paths) > 1 {
		category := paths[1] + "/"
		it := datastore.NewQuery("pkg").Ancestor(key).Filter("Category =", category).Order("Dir").Order("PkgName").Limit(1000).Run(ctx)
		templates.Heading(w, category)
		writePackageList(ctx, w, it)
		return
	}

	var categories []bulk.Pkg
	_, err = datastore.NewQuery("pkg").Ancestor(key).Project("Category").Distinct().GetAll(ctx, &categories)
	if len(categories) == 0 {
		templates.NoDetails(w, r.URL.Path)
		return
	}
	templates.CategoryList(w, categories, r.URL.Path)

	templates.Heading(w, "Packages breaking most other packages")

	it := datastore.NewQuery("pkg").Ancestor(key).Filter("BuildStatus >", bulk.Prefailed).Order("BuildStatus").Order("-Breaks").Limit(100).Run(ctx)
	writePackageList(ctx, w, it)
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
