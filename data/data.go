// Package data contains functions to fetch BulkTracker data items from the
// datastore.
package data

import (
	"github.com/bsiegert/BulkTracker/bulk"

	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

const latestBuildsKey = "latestBuilds"

// LatestBuilds fetches the list of latest builds to show on the landing page.
func LatestBuilds(c appengine.Context) (builds []bulk.Build, err error) {
	_, err = memcache.Gob.Get(c, latestBuildsKey, &builds)
	if err != nil && err != memcache.ErrCacheMiss {
		c.Warningf("get latest builds from memcache: %s", err)
	}
	if err == nil {
		c.Debugf("latestBuilds: used cached result")
		return builds, nil
	}

	keys, err := datastore.NewQuery("build").Order("-Timestamp").Limit(10).GetAll(c, &builds)
	if err != nil {
		return nil, err
	}
	for i := range builds {
		builds[i].Key = keys[i].Encode()
	}

	err = memcache.Gob.Set(c, &memcache.Item{
		Key:        latestBuildsKey,
		Object:     &builds,
		Expiration: 30 * time.Minute,
	})
	if err != nil {
		c.Warningf("failed to Set latestBuilds: %s", err)
	}

	return builds, nil
}
