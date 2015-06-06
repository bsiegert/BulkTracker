package json

import (
	"github.com/bsiegert/BulkTracker/bulk"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Build bulk.Build
	Pkg   bulk.Pkg
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

	item, err := memcache.Get(c, fmt.Sprintf("/json/pkgresults/%s%s", category, dir))
	if err == nil {
		c.Debugf("PkgResults: used cached result")
		w.Write(item.Value)
		return
	} else if err != memcache.ErrCacheMiss {
		c.Warningf("get from memcache: %s", err)
	}

	it := datastore.NewQuery("pkg").Filter("Category =", category).Filter("Dir =", dir).Limit(10).Run(c)
	var results []*PkgResult
	for {
		r := PkgResult{}
		key, err := it.Next(&r.Pkg)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Warningf("failed to read package result: %s", err)
			continue
		}
		r.Pkg.Key = key.Encode()
		buildID := key.Parent()
		// TODO(bsiegert) cache builds by key to avoid repeated Get
		// calls.
		err = datastore.Get(c, buildID, &r.Build)
		if err != nil {
			c.Warningf("failed to read build: %s", err)
			continue
		}
		r.Build.Key = buildID.Encode()
		results = append(results, &r)
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
