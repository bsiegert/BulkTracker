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
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"

	"github.com/bsiegert/BulkTracker/dao"
	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/ingest"
	"github.com/bsiegert/BulkTracker/json"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/pages"
)

var (
	port        = flag.Int("port", 8080, "The port to use.")
	metricsAddr = flag.String("metrics_addr", "", "host:port for serving Prometheus metrics, or 'main' to serve them on the main port")
	dbPath      = flag.String("db_path", "BulkTracker.db", "The path to the SQLite database file.")
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

func registerCategories(ctx context.Context, ddb *ddao.DB, handler http.Handler) error {
	categories, err := ddb.GetCategories(ctx)
	if err != nil {
		return err
	}
	for _, c := range categories {
		log.Infof(ctx, "Handling %v", c)
		handle("/"+c, handler)
	}
	return nil
}

// handle wraps http.Handle.
func handle(path string, handler http.Handler) {
	http.Handle(path, handler)
}

// handleFunc wraps http.HandleFunc.
func handleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	http.HandleFunc(path, handler)
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

	handle("/", &pages.StartPage{
		DB: &ddb,
	})
	handle("/build/", &pages.BuildDetails{
		DB: &ddb,
	})
	handleFunc("/builds", pages.ShowBuilds)
	handle("/robots.txt", http.FileServer(http.FS(staticContent)))
	handle("/images/", http.FileServer(http.FS(staticContent)))
	handle("/mock/", http.FileServer(http.FS(staticContent)))
	handle("/static/", http.FileServer(http.FS(staticContent)))
	handle("/_ah/mail/", &ingest.IncomingMailHandler{
		DB: &ddb,
	})
	handle("/json/", &json.API{
		DB: &ddb,
	})
	handle("/pkg/", &pages.PkgDetails{
		DB: &ddb,
	})

	h, err := fileHandler("static/favicon.ico")
	if err != nil {
		log.Errorf(ctx, "failed to create /favicon.ico handler: %s", err)
		os.Exit(1)
	}
	handleFunc("/favicon.ico", h)

	h, err = fileHandler("static/pkgresults.html")
	if err != nil {
		log.Errorf(ctx, "failed to create /pkgresults handler: %s", err)
		os.Exit(1)
	}
	handleFunc("/pkgresults/", h)

	err = registerCategories(ctx, &ddb, &pages.Dirs{
		DB:         &ddb,
		PkgResults: h,
	})
	if err != nil {
		log.Errorf(ctx, "%s", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	switch *metricsAddr {
	case "":
		log.Infof(context.Background(), "Not exporting Prometheus metrics")
	case "main":
		log.Infof(context.Background(), "Exporting Prometheus metrics on /metrics")
		handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	default:
		a := *metricsAddr
		if _, metricsPort, ok := strings.Cut(a, ":"); ok {
			a = "localhost:" + metricsPort
		}
		log.Infof(context.Background(), "Exporting Prometheus metrics on http://%v/metrics", a)

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		landingPage, err := web.NewLandingPage(web.LandingConfig{
			Name: "BulkTracker",
			Links: []web.LandingLinks{
				{
					Address: fmt.Sprintf("http://localhost:%v/", *port),
					Text:    "Web UI",
				},
				{
					Address: "/metrics",
					Text:    "Metrics",
				},
			},
		})
		if err != nil {
			log.Errorf(context.Background(), "Setting up metrics landing page: %v", err)
			os.Exit(1)
		}
		mux.Handle("/", landingPage)
		go func() {
			log.Errorf(context.Background(), "%v", http.ListenAndServe(*metricsAddr, mux))
		}()
	}

	log.Infof(ctx, "Listening on port %d", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Errorf(ctx, "%s", err)
	}
}
