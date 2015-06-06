// Package data contains functions to fetch BulkTracker data items from the
// datastore.
package data

import (
	"github.com/bsiegert/BulkTracker/bulk"

	"appengine"
	"appengine/datastore"
)

// LatestBuilds fetches the list of latest builds to show on the landing page.
func LatestBuilds(c appengine.Context) (builds []bulk.Build, err error) {
	keys, err := datastore.NewQuery("build").Order("-Timestamp").Limit(10).GetAll(c, &builds)
	if err != nil {
		return nil, err
	}
	for i := range builds {
		builds[i].Key = keys[i].Encode()
	}
	return builds, nil
}
