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
	"time"
)

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

func Dir(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	c := appengine.NewContext(r)

	// TODO(bsiegert) handle /json/dir/dirname to return a list of pkgnames in
	// that directory (union of all builds).
	cacheKey := "/json/dir"
	item, err := memcache.Get(c, cacheKey)
	if err == nil {
		c.Debugf("%s: used cached result", cacheKey)
		w.Write(item.Value)
		return
	} else if err != memcache.ErrCacheMiss {
		c.Warningf("get from memcache: %s", err)
	}

	var result []bulk.Pkg
	_, err = datastore.NewQuery("pkg").Project("Category").Distinct().GetAll(c, &result)
	if err != nil {
		c.Errorf("failed to query packages: %s", err)
	}
	categories := make([]string, len(result))
	for i := range result {
		categories[i] = result[i].Category
	}
	sort.Strings(categories)

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(categories)
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
