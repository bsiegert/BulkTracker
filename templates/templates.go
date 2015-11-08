package templates

import (
	"github.com/bsiegert/BulkTracker/bulk"

	"html/template"
	"io"
	"io/ioutil"
)

const templateDir = "../templates/"

func readFile(name string) string {
	s, err := ioutil.ReadFile(templateDir + name)
	if err != nil {
		panic(err)
	}
	return string(s)
}

var (
	PageHeader    = readFile("header.html")
	PageFooter    = readFile("footer.html")
	ReindexOK     = readFile("reindex_ok.html")
	StartPageLead = readFile("start_page_lead.html")
	TableEnd      = readFile("table_end.html")
)

// t is the top-level template object.
var t = template.Must(template.ParseFiles(
	templateDir+"table_begin.html",
	templateDir+"table_builds.html",
	templateDir+"table_pkgs.html",
	templateDir+"bulk_build_info.html",
	templateDir+"pkg_info.html",
	templateDir+"no_details.html",
	templateDir+"category_list.html",
	templateDir+"heading.html",
	templateDir+"data_table.html"))

func TableBegin(w io.Writer, columns ...string) {
	t.ExecuteTemplate(w, "table_begin.html", columns)
}

func TableBuilds(w io.Writer, b *bulk.Build) {
	t.ExecuteTemplate(w, "table_builds.html", b)
}

func TablePkgs(w io.Writer, p *bulk.Pkg) {
	t.ExecuteTemplate(w, "table_pkgs.html", p)
}

func BulkBuildInfo(w io.Writer, b *bulk.Build) {
	t.ExecuteTemplate(w, "bulk_build_info.html", b)
}

func PkgInfo(w io.Writer, p *bulk.Pkg, b *bulk.Build) {
	t.ExecuteTemplate(w, "pkg_info.html", struct {
		Pkg   *bulk.Pkg
		Build *bulk.Build
	}{p, b})
}

func NoDetails(w io.Writer, path string) {
	t.ExecuteTemplate(w, "no_details.html", path)
}

func CategoryList(w io.Writer, categories []bulk.Pkg, path string) {
	t.ExecuteTemplate(w, "category_list.html", struct {
		Categories []bulk.Pkg
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
