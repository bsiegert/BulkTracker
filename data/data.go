// Package data contains functions to fetch BulkTracker data items from the
// datastore.
package data

import (
	"github.com/bsiegert/BulkTracker/bulk"

	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

const latestBuildsKey = "latestBuildsPerPlatform"

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

	it := datastore.NewQuery("build").Order("-Timestamp").Limit(1000).Run(c)
	var b bulk.Build
RowLoop:
	for {
		key, err := it.Next(&b)
		if err == datastore.Done {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read build: %s", err)
		}
		// Skip old entries with empty branch.
		//if b.Branch == "" {
		//	continue RowLoop
		//}
		// Is this the first entry of this type?
		// TODO(bsiegert) eliminate O(n2) algo.
		for i := range builds {
			bb := builds[i]
			if b.Platform == bb.Platform && b.Branch == bb.Branch && b.Compiler == bb.Compiler && b.User == bb.User {
				continue RowLoop
			}
		}
		b.Key = key.Encode()
		builds = append(builds, b)
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
