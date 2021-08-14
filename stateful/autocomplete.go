/*-
 * Copyright (c) 2014-2018, 2021
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
	"errors"
	"strings"
	"time"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/memcache"
)

const (
	memcacheKey = "allpkgnames"
	lockKey     = "allpkgnames-lock"

	maxInFlight = 5
)

var (
	ch         chan AutocompleteRequest
	errNoCache = errors.New("stateful: no cached data available")
)

// TODO regularly reload the dataset.

func load(ctx context.Context) error {
	var allPkgNames []string
	val, err := memcache.Get(ctx, memcacheKey)
	if err != nil {
		log.Debugf(ctx, "failed to load allpkgnames from cache: %v", err)
		PrefillCache.Call(ctx)
		return errNoCache
	}

	ch = make(chan AutocompleteRequest)
	allPkgNames = strings.Split(strings.TrimSuffix(string(val.Value), "\n"), "\n")
	go func() {
		for req := range ch {
			lookup(allPkgNames, req)
		}
	}()
	return nil
}

func loadFromDatastore(ctx context.Context) ([]string, error) {
	var categories []bulk.Pkg
	_, err := datastore.NewQuery("pkg").Project("Category").Distinct().GetAll(ctx, &categories)
	if err != nil {
		return nil, err
	}

	// Consider 0 package names as an invalid result.
	if len(categories) == 0 {
		return nil, errors.New("got 0 package categories")
	}

	g, ctx := errgroup.WithContext(ctx)

	in := make(chan string)
	var out [maxInFlight][]string
	for i := 0; i < maxInFlight; i++ {
		i := i
		g.Go(func() error {
			for cat := range in {
				var dirs []bulk.Pkg
				_, err := datastore.NewQuery("pkg").Filter("Category =", cat).Project("Dir").Distinct().GetAll(ctx, &dirs)
				if err != nil {
					log.Warningf(ctx, "query for %s failed: %v", cat, err)
					return err
				}
				for _, p := range dirs {
					out[i] = append(out[i], cat+p.Dir)
				}

			}
			return nil
		})
	}
	for _, c := range categories {
		in <- c.Category
	}
	close(in)
	if err = g.Wait(); err != nil {
		return nil, err
	}
	n := 0
	for i := range out {
		n += len(out[i])
	}
	allPkgNames := make([]string, 0, n)
	for i := range out {
		allPkgNames = append(allPkgNames, out[i]...)
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
		log.Warningf(ctx, "error loading from datastore: %v", err)
		return
	}
	err = memcache.Set(ctx, &memcache.Item{
		Key:        memcacheKey,
		Value:      []byte(strings.Join(allPkgNames, "\n")),
		Expiration: 24 * time.Hour,
	})
	if err != nil {
		log.Errorf(ctx, "error writing data to memcache: %v", err)
	}
})

// MaybePrefillCache calls PrefillCache opportunistically if the item is not
// in memcache yet.
func MaybePrefillCache(ctx context.Context) {
	_, err := memcache.Get(ctx, memcacheKey)
	if err == memcache.ErrCacheMiss {
		PrefillCache.Call(ctx)
	}
}

func lookup(allPkgNames []string, req AutocompleteRequest) {
	resp := AutocompleteResponse{Results: []Result{}}
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
func Autocomplete(ctx context.Context, search string) (*AutocompleteResponse, error) {
	if ch == nil {
		log.Infof(ctx, "loading autocomplete data")
		start := time.Now()
		if err := load(ctx); err != nil {
			return nil, err
		}
		log.Infof(ctx, "done loading autocomplete data, took %v", time.Now().Sub(start))
	}
	retCh := make(chan AutocompleteResponse, 1)
	ch <- AutocompleteRequest{
		Ctx:    ctx,
		Search: search,
		Ret:    retCh,
	}
	retval := <-retCh
	return &retval, nil
}
