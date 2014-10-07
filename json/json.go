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
