/*-
 * Copyright (c) 2014-2019, 2022
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

// Package json contains handlers for BulkTracker API methods that return
// JSON data.
package json

import (
	"database/sql"
	"errors"
	"strconv"
	"sync"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
	"github.com/bsiegert/BulkTracker/stateful"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The default duration until a cache entry expires.
const CacheExpiration = 30 * time.Minute

// Endpoint is the standard function signature of a JSON API endpoint.
// params is a list of path components; the format of the URLs is
// /json/endpointname/param1/param2. The function returns a result to be
// marshalled to JSON, or an error.
type Endpoint func(ctx context.Context, params []string, form url.Values) (interface{}, error)

type cacheEntry struct {
	timestamp time.Time
	value     []byte
}

type API struct {
	DB *ddao.DB

	mu    sync.Mutex
	cache map[string]cacheEntry
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	r.ParseForm()
	r.Form.Del("_") // used by some of the JSON calls to prevent caching, hah!

	cacheKey := strings.TrimSuffix(r.URL.Path, "/")
	if len(r.Form) > 0 {
		cacheKey += "?" + r.Form.Encode()
	}
	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/json/"), "/")
	if len(paths) < 1 {
		w.WriteHeader(404)
		return
	}
	if a.CacheGet(ctx, cacheKey, w) {
		return
	}

	result, err := a.dispatch(ctx, paths[0], paths[1:], r.Form)
	if err != nil {
		if result != nil {
			json.NewEncoder(w).Encode(result)
		}
		log.Errorf(ctx, err.Error())
		return
	}
	a.CacheAndWrite(ctx, result, cacheKey, w)
}

func (a *API) dispatch(ctx context.Context, fn string, params []string, form url.Values) (interface{}, error) {
	switch fn {
	case "build":
		return a.BuildDetails(ctx, params, form)
	case "allbuilds":
		return a.AllBuildDetails(ctx, params, form)
	case "pkgresults":
		return a.PkgResults(ctx, params, form)
	case "allpkgresults":
		return a.AllPkgResults(ctx, params, form)
	case "pkgsbreakingmostothers":
		return a.PkgsBreakingMostOthers(ctx, params, form)
	case "pkgsbrokenby":
		return a.PkgsBrokenBy(ctx, params, form)
	case "dir":
		return a.Dir(ctx, params, form)
	case "autocomplete":
		return a.Autocomplete(ctx, params, form)
	}
	return nil, errors.New("unknown function name")
}

// CacheAndWrite stores the JSON representation of v in the cache and writes it
// to w.
func (a *API) CacheAndWrite(ctx context.Context, v interface{}, cacheKey string, w io.Writer) {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(v)

	a.mu.Lock()
	if a.cache == nil {
		a.cache = make(map[string]cacheEntry)
	}
	a.cache[cacheKey] = cacheEntry{
		value:     buf.Bytes(),
		timestamp: time.Now(),
	}
	a.mu.Unlock()

	io.Copy(w, &buf)
}

// CacheGet tries fetching the value with the given cacheKey from memcache and
// writes it to w if it exists. It returns true if there was a cache hit.
func (a *API) CacheGet(ctx context.Context, cacheKey string, w http.ResponseWriter) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	item, ok := a.cache[cacheKey]
	if !ok {
		log.Infof(ctx, "cache miss for %q", cacheKey)
		return false
	}
	if time.Since(item.timestamp) > CacheExpiration {
		log.Infof(ctx, "cache entry too old for %q", cacheKey)
		delete(a.cache, cacheKey)
		return false
	}
	log.Debugf(ctx, "used cached result for %q", cacheKey)
	w.Write(item.value)
	return true
}

// BuildDetails returns a single build record identified by ID.
func (a *API) BuildDetails(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	buildID, err := strconv.ParseInt(params[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing build ID %q", params[0])
	}
	return a.DB.GetBuild(ctx, buildID)
}

// AllBuildDetails returns all build records.
func (a *API) AllBuildDetails(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	return a.DB.LatestBuilds(ctx, false /* filter */)
}

func (a *API) PkgResults(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) < 2 {
		return []ddao.GetAllPkgResultsRow{}, nil
	}
	category, dir := params[0]+"/", params[1]

	all, err := a.DB.GetAllPkgResults(ctx, category, dir)
	if err != nil {
		// No results is not an error.
		if errors.Is(err, sql.ErrNoRows) {
			return []ddao.GetAllPkgResultsRow{}, nil
		}
		return nil, err
	}
	var results []ddao.GetAllPkgResultsRow

	// Only keep the first (i.e. most recent) result.
	buildsSeen := make(map[ddao.Build]bool)
	for _, r := range all {
		b := ddao.Build{
			Platform:  r.Platform,
			Branch:    r.Branch,
			Compiler:  r.Compiler,
			BuildUser: r.BuildUser,
		}
		if !buildsSeen[b] {
			results = append(results, r)
		}
		buildsSeen[b] = true
	}
	return results, nil
}

func (a *API) AllPkgResults(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) < 2 {
		return []bulk.PkgResult{}, nil
	}
	category, dir := params[0]+"/", params[1]

	return a.DB.GetAllPkgResults(ctx, category, dir)
}

func (a *API) Dir(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	var category string
	if len(params) > 0 {
		category = params[0]
	}
	if category != "" && !strings.HasSuffix(category, "/") {
		category += "/"
	}

	if category == "" {
		// List all categories.
		return a.DB.GetCategories(ctx)
	} else {
		// List all pkgnames in a category (union of all builds).
		return a.DB.GetPkgsInCategory(ctx, category)
	}
}

func (a *API) Autocomplete(ctx context.Context, _ []string, form url.Values) (interface{}, error) {
	term := form.Get("term")
	if len(term) < 2 {
		return stateful.AutocompleteResponse{
			// select2 gets confused if the value is null.
			Results: []stateful.Result{},
		}, nil
	}
	names, err := a.DB.GetAllPkgsMatching(ctx, term)
	if err != nil {
		return stateful.AutocompleteResponse{
			// select2 gets confused if the value is null.
			Results: []stateful.Result{},
		}, err
	}
	resp := &stateful.AutocompleteResponse{
		Results: make([]stateful.Result, len(names)),
	}
	for i := range names {
		resp.Results[i].ID = names[i]
		resp.Results[i].Text = names[i]
	}
	return resp, nil
}

func (a *API) PkgsBreakingMostOthers(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	var buildID sql.NullInt64
	var err error
	buildID.Int64, err = strconv.ParseInt(params[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing build ID %q", params[0])
	}
	buildID.Valid = true
	return a.DB.GetPkgsBreakingMostOthers(ctx, buildID)
}

func (a *API) PkgsBrokenBy(ctx context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	resultID, err := strconv.ParseInt(params[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing result ID %q", params[0])
	}

	return a.DB.GetPkgsBrokenBy(ctx, resultID)
}
