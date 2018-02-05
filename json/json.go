/*-
 * Copyright (c) 2014-2018
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
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/data"
	"github.com/bsiegert/BulkTracker/stateful"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// The default duration until a cache entry expires.
const CacheExpiration = 30 * time.Minute

// Endpoint is the standard function signature of a JSON API endpoint.
// params is a list of path components; the format of the URLs is
// /json/endpointname/param1/param2. The function returns a result to be
// marshalled to JSON, or an error.
type Endpoint func(c context.Context, params []string, form url.Values) (interface{}, error)

func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)
	r.ParseForm()
	r.Form.Del("_") // used by some of the JSON calls to prevent caching, hah!

	cacheKey := strings.TrimSuffix(r.URL.Path, "/")
	if len(r.Form) > 0 {
		cacheKey += "?" + r.Form.Encode()
	}
	paths := strings.Split(strings.TrimPrefix(cacheKey, "/json/"), "/")
	if len(paths) < 1 {
		w.WriteHeader(404)
		return
	}
	if CacheGet(c, cacheKey, w) {
		return
	}

	result, err := e(c, paths[1:], r.Form)
	if err != nil {
		if result != nil {
			json.NewEncoder(w).Encode(result)
		}
		log.Errorf(c, err.Error())
		return
	}
	CacheAndWrite(c, result, cacheKey, w)
}

// Mux maps endpoint names to their implementations. It assumes that the results
// of a successful endpoint call are cacheable.
var Mux = map[string]Endpoint{
	"build":         BuildDetails,
	"allbuilds":     AllBuildDetails,
	"pkgresults":    PkgResults,
	"allpkgresults": AllPkgResults,
	"dir":           Dir,
	"autocomplete":  Autocomplete,
}

// CacheAndWrite stores the JSON representation of v in the App Engine
// memcache and writes it to w.
func CacheAndWrite(c context.Context, v interface{}, cacheKey string, w io.Writer) {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(v)
	err := memcache.Set(c, &memcache.Item{
		Key:        cacheKey,
		Value:      buf.Bytes(),
		Expiration: CacheExpiration,
	})
	if err != nil {
		log.Warningf(c, "failed to write %q to cache: %s", cacheKey, err)
	}
	io.Copy(w, &buf)
}

// CacheGet tries fetching the value with the given cacheKey from memcache and
// writes it to w if it exists. It returns true if there was a cache hit.
func CacheGet(c context.Context, cacheKey string, w http.ResponseWriter) bool {
	item, err := memcache.Get(c, cacheKey)
	if err != nil {
		if err != memcache.ErrCacheMiss {
			log.Warningf(c, "get from memcache: %s", err)
		}
		return false
	}
	log.Debugf(c, "%s: used cached result", cacheKey)
	w.Write(item.Value)
	return true
}

func BuildDetails(c context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}
	key, err := datastore.DecodeKey(params[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %s", err)
	}

	b := &bulk.Build{Key: key.Encode()}
	err = datastore.Get(c, key, b)
	if err != nil {
		return nil, fmt.Errorf("getting build record: %s", err)
	}
	return b, nil
}

func AllBuildDetails(c context.Context, params []string, _ url.Values) (interface{}, error) {
	var builds []bulk.Build
	keys, err := datastore.NewQuery("build").Order("-Timestamp").GetAll(c, &builds)
	if err != nil {
		return nil, fmt.Errorf("failed to query builds: %s", err)
	}
	for i := range keys {
		builds[i].Key = keys[i].Encode()
	}
	return builds, nil
}

type PkgResult struct {
	Build *bulk.Build
	Pkg   *bulk.Pkg
}

func PkgResults(c context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) < 2 {
		return nil, nil
	}
	category, dir := params[0]+"/", params[1]

	// Get results for each of the LatestBuilds.
	builds, err := data.LatestBuilds(c)
	if err != nil {
		return nil, fmt.Errorf("getting LatestBuilds: %s", err)
	}
	// Limit for now.
	if len(builds) > 40 {
		builds = builds[:40]
	}

	// Fan out to datastore, all in parallel.
	ch := make(chan []PkgResult, 10)
	for i := range builds {
		go func(b *bulk.Build) {
			var pkgs []bulk.Pkg
			key, err := datastore.DecodeKey(b.Key)
			if err != nil {
				log.Errorf(c, "unable to decode key: %s", err)
			}
			pkgkeys, err := datastore.NewQuery("pkg").Ancestor(key).Filter("Category =", category).Filter("Dir =", dir).Limit(10).GetAll(c, &pkgs)
			if err != nil {
				log.Errorf(c, "failed to query packages: %s", err)
			}
			for j := range pkgkeys {
				pkgs[j].Key = pkgkeys[j].Encode()
			}
			results := make([]PkgResult, len(pkgs))
			for j := range results {
				results[j].Build = b
				results[j].Pkg = &pkgs[j]
			}
			ch <- results
		}(&builds[i])
	}
	var results []PkgResult
	for range builds {
		results = append(results, <-ch...)
	}
	return results, nil
}

func AllPkgResults(c context.Context, params []string, _ url.Values) (interface{}, error) {
	if len(params) < 2 {
		return []bulk.Pkg{}, nil
	}
	category, dir := params[0]+"/", params[1]

	var pkgs []bulk.Pkg
	pkgkeys, err := datastore.NewQuery("pkg").Filter("Category =", category).Filter("Dir =", dir).Limit(1000).GetAll(c, &pkgs)
	if err != nil {
		log.Errorf(c, "failed to query packages: %s", err)
	}
	for j := range pkgkeys {
		pkgs[j].Key = pkgkeys[j].Encode()
	}
	results := make([]PkgResult, len(pkgs))
	for j := range results {
		results[j].Pkg = &pkgs[j]

		// TODO(bsiegert) do this in parallel and/or cache repeated values.
		// One way would be to build a list of empty build records and desired
		// keys, then call GetMulti.
		buildKey := pkgkeys[j].Parent()
		b := &bulk.Build{Key: buildKey.Encode()}
		err = datastore.Get(c, buildKey, b)
		if err != nil {
			log.Errorf(c, "getting build record: %s", err)
		}
		results[j].Build = b
	}
	return results, nil
}

func Dir(c context.Context, params []string, _ url.Values) (interface{}, error) {
	var category string
	if len(params) > 0 {
		category = params[0]
	}
	if category != "" && !strings.HasSuffix(category, "/") {
		category += "/"
	}

	var pkgs []bulk.Pkg
	var result []string
	if category == "" {
		// List all categories.
		_, err := datastore.NewQuery("pkg").Project("Category").Distinct().GetAll(c, &pkgs)
		if err != nil {
			return nil, fmt.Errorf("failed to query packages: %s", err)
		}
		result = make([]string, len(pkgs))
		for i := range pkgs {
			result[i] = pkgs[i].Category
		}
	} else {
		// List all pkgnames in a category (union of all builds).
		_, err := datastore.NewQuery("pkg").Filter("Category =", category).Project("Dir").Distinct().GetAll(c, &pkgs)
		if err != nil {
			return nil, fmt.Errorf("failed to query packages: %s", err)
		}
		result = make([]string, len(pkgs))
		for i := range pkgs {
			result[i] = pkgs[i].Dir
		}
	}

	sort.Strings(result)
	return result, nil
}

func Autocomplete(c context.Context, _ []string, form url.Values) (interface{}, error) {
	term := form.Get("term")
	if term == "" {
		return stateful.AutocompleteResponse{}, nil
	}
	ch := make(chan stateful.AutocompleteResponse)
	if err := stateful.Autocomplete(stateful.AutocompleteRequest{Ctx: c, Search: term, Ret: ch}); err != nil {
		return nil, err
	}
	return <-ch, nil
}
