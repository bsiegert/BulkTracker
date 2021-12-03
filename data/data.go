/*-
 * Copyright (c) 2014-2021
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

// Package data contains functions to fetch BulkTracker data items from the
// datastore.
package data

import (
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/log"

	"context"
	"fmt"
	"time"

	"google.golang.org/appengine/v2/datastore"
	"google.golang.org/appengine/v2/memcache"
)

const latestBuildsKey = "latestBuildsPerPlatform"

// LatestBuilds fetches the list of latest builds to show on the landing page.
func LatestBuilds(ctx context.Context) (builds []bulk.Build, err error) {
	_, err = memcache.Gob.Get(ctx, latestBuildsKey, &builds)
	if err != nil && err != memcache.ErrCacheMiss {
		log.Warningf(ctx, "get latest builds from memcache: %s", err)
	}
	if err == nil {
		log.Debugf(ctx, "latestBuilds: used cached result")
		return builds, nil
	}

	it := datastore.NewQuery("build").Order("-Timestamp").Limit(1000).Run(ctx)
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

	err = memcache.Gob.Set(ctx, &memcache.Item{
		Key:        latestBuildsKey,
		Object:     &builds,
		Expiration: 30 * time.Minute,
	})
	if err != nil {
		log.Warningf(ctx, "failed to Set latestBuilds: %s", err)
	}

	return builds, nil
}
