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

// Binary bulktracker serves the BulkTracker web app.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/bsiegert/BulkTracker/dao"
	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/ingest"
	"github.com/bsiegert/BulkTracker/json"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/pages"
)

var (
	port   = flag.Int("port", 8080, "The port to use.")
	dbPath = flag.String("db_path", "BulkTracker.db", "The path to the SQLite database file.")
)

//go:embed images mock static robots.txt
var staticContent embed.FS

// fileHandler returns a HTTP handler for a file from static content.
func fileHandler(name string) (http.HandlerFunc, error) {
	f, err := staticContent.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, name, stat.ModTime(), f.(io.ReadSeeker))
	}, nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	db, err := dao.New(ctx, "sqlite3", *dbPath)
	if err != nil {
		log.Errorf(ctx, "failed to open database: %s", err)
		os.Exit(1)
	}
	var ddb ddao.DB
	ddb.Queries = *ddao.New(db.DB)

	http.Handle("/", &pages.StartPage{
		DB: &ddb,
	})
	http.Handle("/build/", &pages.BuildDetails{
		DB: &ddb,
	})
	http.HandleFunc("/builds", pages.ShowBuilds)
	http.Handle("/robots.txt", http.FileServer(http.FS(staticContent)))
	http.Handle("/images/", http.FileServer(http.FS(staticContent)))
	http.Handle("/mock/", http.FileServer(http.FS(staticContent)))
	http.Handle("/static/", http.FileServer(http.FS(staticContent)))
	http.Handle("/_ah/mail/", &ingest.IncomingMailHandler{
		DB: db,
	})
	http.Handle("/json/", &json.API{
		DB: &ddb,
	})
	// http.HandleFunc("/pkg/", pages.PkgDetails)

	h, err := fileHandler("static/pkgresults.html")
	if err != nil {
		log.Errorf(ctx, "failed to create /pkgresults handler: %s", err)
		os.Exit(1)
	}
	http.HandleFunc("/pkgresults/", h)

	log.Infof(ctx, "Listening on port %d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Errorf(ctx, "%s", err)
	}
}
