package json

import (
	"github.com/bsiegert/bulktracker/bulk"

	"appengine"
	"appengine/datastore"

	"encoding/json"
	"net/http"
	"strings"
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

	b := &bulk.Build{}
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

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/json/build/"), "/")
	if len(paths) < 2 {
		return
	}
	category, dir := paths[0]+"/", paths[1]
	it := datastore.NewQuery("pkg").Filter("Category =", category).Filter("Dir =", dir).Limit(10).Run(c)
	results := make([]PkgResult, 10)
	var i int
	for {
		r := &results[i]
		i++
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
	}
	results = results[:i]
	json.NewEncoder(w).Encode(results)
}
