/*-
 * Copyright (c) 2014-2018, 2021, 2024
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
	"html/template"
	"io"
	"os"
)

func readFile(name string) string {
	s, err := os.ReadFile("templates/" + name)
	if err != nil {
		panic(err)
	}
	return string(s)
}

var (
	PageHeader    = readFile("header.html")
	PageFooter    = readFile("footer.html")
	StartPageLead = readFile("start_page_lead.html")
	TableEnd      = readFile("table_end.html")
)

// t is the top-level template object.
var t = template.Must(template.ParseFiles(
	"templates/heading.html",
	"templates/data_table.html"))

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
