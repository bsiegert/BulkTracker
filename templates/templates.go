/*-
 * Copyright (c) 2014-2018, 2021-2022
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
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/ddao"

	"embed"
	"html/template"
	"io"
)

var (
	//go:embed header.html
	PageHeader string

	//go:embed footer.html
	PageFooter string

	//go:embed reindex_ok.html
	ReindexOK string

	//go:embed start_page_lead.html
	StartPageLead string

	//go:embed table_end.html
	TableEnd string
)

//go:embed *.html
var emb embed.FS

// t is the top-level template object.
var t = template.Must(template.ParseFS(emb, "*.html"))

func TableBegin(w io.Writer, columns ...string) {
	t.ExecuteTemplate(w, "table_begin.html", columns)
}

func TableBuilds(w io.Writer, b *ddao.Build) {
	t.ExecuteTemplate(w, "table_builds.html", b)
}

func TablePkgs(w io.Writer, p *bulk.Pkg) {
	t.ExecuteTemplate(w, "table_pkgs.html", p)
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

func DataTable(w io.Writer, settings string) {
	var js *template.JS
	if settings != "" {
		s := template.JS(settings)
		js = &s
	}
	t.ExecuteTemplate(w, "data_table.html", js)
}
