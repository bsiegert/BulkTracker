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

// Binary bulktracker is the main program for BulkTracker on App Engine.
package main

import (
	"github.com/bsiegert/BulkTracker/json"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/pages"

	"google.golang.org/appengine"

	"fmt"
	"net/http"
)

func main() {
	log.InitLogger()

	http.Handle("/", &pages.StartPage{})
	http.HandleFunc("/builds", pages.ShowBuilds)
	http.HandleFunc("/build/", pages.BuildDetails)
	http.HandleFunc("/pkg/", pages.PkgDetails)
	// http.HandleFunc("/_ah/mail/", ingest.HandleIncomingMail)

	for path, endpoint := range json.Mux {
		path = fmt.Sprintf("/json/%s/", path)
		http.Handle(path, endpoint)
	}
	appengine.Main()
}
