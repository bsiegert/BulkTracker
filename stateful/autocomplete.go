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

// Package stateful contains stateful components for BulkTracker, i.e. those
// that keep state across requests.
package stateful

import (
	"context"
	"strings"
	"time"

	"github.com/bsiegert/BulkTracker/bulk"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

const (
	memcacheKey = "allpkgnames"
	lockKey     = "allpkgnames-lock"
)

var ch chan AutocompleteRequest

// TODO regularly reload the dataset.

func load(ctx context.Context) error {
	// Replace ch with the new one. In the future, when reloading (i.e.
	// ch != nil at this point), consider moving the setting to the end.
	ch = make(chan AutocompleteRequest)

	var allPkgNames []string
	if val, err := memcache.Get(ctx, "allpkgnames"); err == nil {
		allPkgNames = strings.Split(strings.TrimSuffix(string(val.Value), "\n"), "\n")
	} else {
		log.Debugf(ctx, "failed to load allpkgnames from cache: %v", err)
		allPkgNames, err = loadFromDatastore(ctx)
		if err != nil {
			return err
		}
		err = memcache.Set(ctx, &memcache.Item{
			Key:        "allpkgnames",
			Value:      []byte(strings.Join(allPkgNames, "\n")),
			Expiration: 24 * time.Hour,
		})
		if err != nil {
			log.Warningf(ctx, "failed to write to cache: %v", err)
		}
	}

	go func() {
		for req := range ch {
			lookup(allPkgNames, req)
		}
	}()
	return nil
}

func loadFromDatastore(ctx context.Context) ([]string, error) {
	var pkgs []bulk.Pkg
	_, err := datastore.NewQuery("pkg").Project("Category", "Dir").Distinct().GetAll(ctx, &pkgs)
	if err != nil {
		return nil, err
	}

	allPkgNames := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		allPkgNames = append(allPkgNames, p.Category+p.Dir)
	}
	return allPkgNames, nil
}

var PrefillCache = delay.Func("PrefillCache", func(ctx context.Context) {
	_, err := memcache.Get(ctx, lockKey)
	if err == nil {
		// Another instance is holding the "poor man's lock".
		return
	}
	err = memcache.Set(ctx, &memcache.Item{
		Key:        lockKey,
		Value:      []byte{'*'},
		Expiration: time.Minute,
	})
	if err != nil {
		log.Warningf(ctx, "cannot set memcache lock: %v", err)
	}
	defer func() {
		if err := memcache.Delete(ctx, lockKey); err != nil {
			log.Warningf(ctx, "cannot remove memcache lock: %v", err)
		}
	}()

	allPkgNames, err := loadFromDatastore(ctx)
	if err != nil {
		log.Errorf(ctx, "error loading from datastore: %v", err)
		return
	}
	err = memcache.Set(ctx, &memcache.Item{
		Key:        "allpkgnames",
		Value:      []byte(strings.Join(allPkgNames, "\n")),
		Expiration: 24 * time.Hour,
	})
	if err != nil {
		log.Errorf(ctx, "error writing data to memcache: %v", err)
	}
})

func lookup(allPkgNames []string, req AutocompleteRequest) {
	resp := AutocompleteResponse{}
	for _, p := range allPkgNames {
		if strings.Contains(p, req.Search) {
			resp.Results = append(resp.Results, Result{
				ID:   p,
				Text: p,
			})
		}
	}
	req.Ret <- resp
}

// Result encodes a single line of the select2 JSON response format.
type Result struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// AutocompleteResponse is the overall return type of the select2 JSON response.
type AutocompleteResponse struct {
	Results    []Result `json:"results"`
	Pagination struct {
		More bool `json:"more"`
	} `json:"pagination,omitempty"`
}

// AutocompleteRequest has the parameters for a single autocomplete request.
type AutocompleteRequest struct {
	// ctx is the request context. It must be an App Engine context. It is
	// only used for this one request.
	Ctx context.Context
	// search is the search string.
	Search string
	// ret is the channel in which to return the reponse.
	Ret chan<- AutocompleteResponse
}

// Autocomplete sends an autocomplete request. The response will be delivered
// over the channel in req.Ret. The function returns an error that occurs while
// loading the dataset.
func Autocomplete(req AutocompleteRequest) error {
	if ch == nil {
		log.Infof(req.Ctx, "loading autocomplete data")
		start := time.Now()
		if err := load(req.Ctx); err != nil {
			return err
		}
		log.Infof(req.Ctx, "done loading autocomplete data, took %v", time.Now().Sub(start))
	}
	ch <- req
	return nil
}
