/*-
 * Copyright (c) 2014-2021, 2024
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
	"io"
	"net/http"
	"strings"

	"github.com/bsiegert/BulkTracker/templates"

	"google.golang.org/appengine"
)

const newAddress = "https://releng.netbsd.org/bulktracker/"

func main() {
	http.HandleFunc("/", StartPage)
	http.Handle("/pkgresults/", &redirector{"/pkgresults/"})
	appengine.Main()
}

type redirector struct {
	pathPrefix string
}

func (r *redirector) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, r.pathPrefix) {
		http.NotFound(w, req)
		return
	}
	path := strings.TrimPrefix(req.URL.Path, r.pathPrefix)
	http.Redirect(w, req, newAddress+path, http.StatusMovedPermanently)
}

func StartPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	io.WriteString(w, templates.PageHeader)
	defer io.WriteString(w, templates.PageFooter)
	io.WriteString(w, templates.StartPageLead)
}
