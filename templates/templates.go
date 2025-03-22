/*-
 * Copyright (c) 2014-2018, 2021-2024
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
	"net/http"
	"sync"
	"time"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
)

//go:embed *.html
var emb embed.FS

// BasePath is the path at which the UI is served. It is set from a flag in
// main.
var BasePath = "/"

// Varsion is the current version string for this app. It is set from the main package.
var Version string

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
	t.ExecuteTemplate(w, "footer.html", &Version)
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

func TablePkgs[T any](w io.Writer, rows []T) {
	s := struct {
		Rows []T
		bp
	}{
		Rows: rows,
	}
	t.ExecuteTemplate(w, "table_pkgs.html", s)
}

func BulkBuildInfo(w io.Writer, b *ddao.Build) {
	t.ExecuteTemplate(w, "bulk_build_info.html", b)
}

func PkgInfo(w io.Writer, res ddao.GetSingleResultRow) {
	// Define what to fetch - these represent the stages of the build process
	// and are used to fetch the corresponding log files.
	stages := []string{
		"work.log",
		"pre-clean.log",
		"checksum.log",
		"depends.log",
		"configure.log",
		"build.log",
		"install.log",
	}

	// Create results slice
	results := make([]URLRequestResult, len(stages))

	// Create a WaitGroup to wait for all goroutines
	var wg sync.WaitGroup
	wg.Add(len(stages))

	// Create a context with a 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch URLs in parallel
	for i, s := range stages {
		// Construct the URL for the current stage
		url := fmt.Sprintf("%s%s/%s", res.BaseURL(), res.PkgName, s)

		// Start a goroutine for each URL to fetch stage data
		go func(index int, stage string, reqURL string) {
			defer wg.Done()

			// Initialize result with ID and URL
			results[index] = URLRequestResult{
				ID:  stage,
				URL: reqURL,
			}

			// Create a new request with the context
			req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
			if err != nil {
				results[index].StatusCode = http.StatusInternalServerError
				results[index].Error = err.Error()
				return
			}

			// Make the request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[index].StatusCode = http.StatusInternalServerError
				results[index].Error = err.Error()
				return
			}
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results[index].StatusCode = resp.StatusCode
				results[index].Error = err.Error()
				return
			}

			// Store the result
			results[index].StatusCode = resp.StatusCode
			results[index].Data = string(body)
		}(i, s, url) // i is index, s is stage, url is the constructed URL
	}

	// Wait for all requests to complete
	wg.Wait()

	// Create the structure to pass to the template
	s := struct {
		Res     *ddao.GetSingleResultRow
		URLData []URLRequestResult
		bp
	}{
		Res:     &res,
		URLData: results,
	}

	// Execute the template
	err := t.ExecuteTemplate(w, "pkg_info.html", s)
	if err != nil {
		log.Errorf(ctx, "templates.PkgInfo: %v", err)
	}
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

type buildDetailsInitParams struct {
	Selector string
	APIName  string
	Number   int64
}

func BuildDetailsInit(w io.Writer, selector string, apiName string, number int64) {
	t.ExecuteTemplate(w, "builddetails_init.html", buildDetailsInitParams{
		Selector: selector,
		APIName:  apiName,
		Number:   number,
	})
}

func SentinelsInit(w io.Writer, selector string, apiName string, number int64) {
	t.ExecuteTemplate(w, "sentinels_init.html", buildDetailsInitParams{
		Selector: selector,
		APIName:  apiName,
		Number:   number,
	})
}

// URLRequestResult represents the result of a URL request
type URLRequestResult struct {
	ID         string
	URL        string
	StatusCode int
	Data       string
	Error      string
}
