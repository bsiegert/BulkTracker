/*-
 * Copyright (c) 2014-2018, 2021-2023
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

package templates

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
)

//go:embed *.html
var emb embed.FS

// BasePath is the path at which the UI is served. It is set from a flag in
// main.
var BasePath = "/"

// bp is a shim type that avoids allocations in common templates.
type bp struct{}

func (bp) BasePath() string { return BasePath }

// ID returns an HTML element ID.
func ID(s string) *id {
	return &id{s}
}

type id struct {
	ID string
}

// t is the top-level template object.
var t = template.Must(template.ParseFS(emb, "*.html"))

func PageHeader(w io.Writer) {
	t.ExecuteTemplate(w, "header.html", bp{})
}

func PageFooter(w io.Writer) {
	t.ExecuteTemplate(w, "footer.html", nil)
}

func PkgResults(w io.Writer) {
	t.ExecuteTemplate(w, "pkgresults.html", bp{})
}

func StartPageLead(w io.Writer) {
	t.ExecuteTemplate(w, "start_page_lead.html", bp{})
}

func StartPageLead2(w io.Writer) {
	t.ExecuteTemplate(w, "start_page_lead2.html", nil)
}

func ReindexOK(w io.Writer) {
	t.ExecuteTemplate(w, "reindex_ok.html", nil)
}

func TableEnd(w io.Writer) {
	t.ExecuteTemplate(w, "table_end.html", nil)
}

type tableBeginParams struct {
	ID      string
	Columns []string
}

func TableBegin(w io.Writer, columns ...string) {
	// TODO: replace usage with TableBeginID.
	err := t.ExecuteTemplate(w, "table_begin.html", tableBeginParams{
		Columns: columns,
	})
	if err != nil {
		log.Errorf(context.TODO(), "templates.TableBegin: %v", err)
	}
}

func TableBeginID(w io.Writer, id *id, columns ...string) {
	err := t.ExecuteTemplate(w, "table_begin.html", tableBeginParams{
		ID:      id.ID,
		Columns: columns,
	})
	if err != nil {
		log.Errorf(context.TODO(), "templates.TableBeginID: %v", err)
	}
}

func TableBuilds(w io.Writer, b *ddao.Build) {
	s := struct {
		*ddao.Build
		bp
	}{
		Build: b,
		bp:    bp{},
	}
	err := t.ExecuteTemplate(w, "table_builds.html", s)
	if err != nil {
		log.Errorf(context.TODO(), "templates.TableBeginClass: %v", err)
	}
}

func TablePkgs(w io.Writer, rows []ddao.GetResultsInCategoryRow) {
	t.ExecuteTemplate(w, "table_pkgs.html", rows)
}

func BulkBuildInfo(w io.Writer, b *bulk.Build) {
	t.ExecuteTemplate(w, "bulk_build_info.html", b)
}

func PkgInfo(w io.Writer, res ddao.GetSingleResultRow) {
	t.ExecuteTemplate(w, "pkg_info.html", res)
}

func NoDetails(w io.Writer, path string) {
	t.ExecuteTemplate(w, "no_details.html", path)
}

func DatastoreError(w io.Writer, err error) {
	t.ExecuteTemplate(w, "datastore_error.html", err)
}

func CategoryList(w io.Writer, categories []string, path string) {
	t.ExecuteTemplate(w, "category_list.html", struct {
		Categories []string
		CurrentURL string
	}{categories, path})
}

func Heading(w io.Writer, text string) {
	t.ExecuteTemplate(w, "heading.html", text)
}

type dataTableParams struct {
	ID       string
	Settings *template.JS
}

func DataTable(w io.Writer, id *id, settings string) {
	params := dataTableParams{}
	if id != nil {
		params.ID = id.ID
	}
	if settings != "" {
		s := template.JS(settings)
		params.Settings = &s
	}
	t.ExecuteTemplate(w, "data_table.html", params)
}

func LoadScript(w io.Writer, filename string) {
	fmt.Fprintf(w, `<script src="%sstatic/%s"></script>`, BasePath, filename)
}
