// Package json contains handlers for BulkTracker API methods that return
// JSON data.
package json

import (
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/data"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// TODO(bsiegert) There is a lot of duplicated code in these functions.
// Abstract some of it away.

func BuildDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)

	// begin copy+paste
	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/json/build/"), "/")
	if len(paths) == 0 {
		return
	}
	key, err := datastore.DecodeKey(paths[0])
	if err != nil {
		c.Warningf("error decoding key: %s", err)
		return
	}

	b := &bulk.Build{Key: key.Encode()}
	err = datastore.Get(c, key, b)
	if err != nil {
		c.Warningf("getting build record: %s", err)
		return
	}
	// end copy+paste

	json.NewEncoder(w).Encode(b)
}

type PkgResult struct {
	Build *bulk.Build
	Pkg   *bulk.Pkg
}

func PkgResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/json/pkgresults/"), "/")
	if len(paths) < 2 {
		return
	}
	category, dir := paths[0]+"/", paths[1]
	cacheKey := fmt.Sprintf("/json/pkgresults/%s%s", category, dir)

	item, err := memcache.Get(c, cacheKey)
	if err == nil {
		c.Debugf("%s: used cached result", cacheKey)
		w.Write(item.Value)
		return
	} else if err != memcache.ErrCacheMiss {
		c.Warningf("get from memcache: %s", err)
	}

	// Get results for each of the LatestBuilds.
	builds, err := data.LatestBuilds(c)
	if err != nil {
		c.Errorf("getting LatestBuilds: %s", err)
		io.WriteString(w, "[]")
		// Do not cache.
		return
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
				c.Errorf("unable to decode key: %s", err)
			}
			pkgkeys, err := datastore.NewQuery("pkg").Ancestor(key).Filter("Category =", category).Filter("Dir =", dir).Limit(10).GetAll(c, &pkgs)
			if err != nil {
				c.Errorf("failed to query packages: %s", err)
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

	if len(results) == 0 {
		io.WriteString(w, "[]")
		return
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(results)
	err = memcache.Set(c, &memcache.Item{
		Key:        cacheKey,
		Value:      buf.Bytes(),
		Expiration: 30 * time.Minute,
	})
	if err != nil {
		c.Warningf("failed to write to cache: %s", err)
	}

	io.Copy(w, &buf)
}

func AllPkgResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/json/allpkgresults/"), "/")
	if len(paths) < 2 {
		return
	}
	category, dir := paths[0]+"/", paths[1]
	cacheKey := fmt.Sprintf("/json/allpkgresults/%s%s", category, dir)

	item, err := memcache.Get(c, cacheKey)
	if err == nil {
		c.Debugf("%s: used cached result", cacheKey)
		w.Write(item.Value)
		return
	} else if err != memcache.ErrCacheMiss {
		c.Warningf("get from memcache: %s", err)
	}
	
	var pkgs []bulk.Pkg
	pkgkeys, err := datastore.NewQuery("pkg").Filter("Category =", category).Filter("Dir =", dir).Limit(1000).GetAll(c, &pkgs)
	if err != nil {
		c.Errorf("failed to query packages: %s", err)
	}
	for j := range pkgkeys {
		pkgs[j].Key = pkgkeys[j].Encode()
	}
	results := make([]PkgResult, len(pkgs))
	for j := range results {
		results[j].Pkg = p
		
		// TODO(bsiegert) do this in parallel and/or cache repeated values.
		// One way would be to build a list of empty build records and desired
		// keys, then call GetMulti.
		buildKey := pkgkeys[j].Ancestor()
		b := &bulk.Build{Key: buildKey.Encode()}
		err = datastore.Get(c, buildKey, b)
		if err != nil {
			c.Errorf("getting build record: %s", err)
		}
		results[j].Build = b
	}
	if len(results) == 0 {
		io.WriteString(w, "[]")
		return
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(results)
	err = memcache.Set(c, &memcache.Item{
		Key:        cacheKey,
		Value:      buf.Bytes(),
		Expiration: 30 * time.Minute,
	})
	if err != nil {
		c.Warningf("failed to write to cache: %s", err)
	}

	io.Copy(w, &buf)
}

func Dir(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)

	// TODO(bsiegert) handle /json/dir/dirname to return a list of pkgnames in
	// that directory (union of all builds).
	category := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/json/dir/"), "/", 2)[0]
	if category != "" && !strings.HasSuffix(category, "/") {
		category += "/"
	}
	cacheKey := "/json/dir/" + category
	item, err := memcache.Get(c, cacheKey)
	if err == nil {
		c.Debugf("%s: used cached result", cacheKey)
		w.Write(item.Value)
		return
	} else if err != memcache.ErrCacheMiss {
		c.Warningf("get from memcache: %s", err)
	}

	var pkgs []bulk.Pkg
	var result []string
	if category == "" {
		// List all categories.
		_, err = datastore.NewQuery("pkg").Project("Category").Distinct().GetAll(c, &pkgs)
		if err != nil {
			c.Errorf("failed to query packages: %s", err)
		}
		result = make([]string, len(pkgs))
		for i := range pkgs {
			result[i] = pkgs[i].Category
		}
	} else {
		// List all pkgnames in a category (union of all builds).
		_, err = datastore.NewQuery("pkg").Filter("Category =", category).Project("Dir").Distinct().GetAll(c, &pkgs)
		if err != nil {
			c.Errorf("failed to query packages: %s", err)
		}
		result = make([]string, len(pkgs))
		for i := range pkgs {
			result[i] = pkgs[i].Dir
		}
	}

	sort.Strings(result)

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(result)
	err = memcache.Set(c, &memcache.Item{
		Key:        cacheKey,
		Value:      buf.Bytes(),
		Expiration: 30 * time.Minute,
	})
	if err != nil {
		c.Warningf("failed to write to cache: %s", err)
	}

	io.Copy(w, &buf)
}
