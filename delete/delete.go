/*-
 * Copyright (c) 2020
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

// Package delete provides functions to delete bulk build records.
package delete

import (
	"context"

	"github.com/bsiegert/BulkTracker/dsbatch"
	"github.com/bsiegert/BulkTracker/log"

	"google.golang.org/appengine/v2/datastore"
	"google.golang.org/appengine/v2/delay"
)

// DeleteBuildDetails deletes the detailed records for the given build.
var DeleteBuildDetails = delay.Func("DeleteBuildDetails", func(ctx context.Context, build *datastore.Key) {
	keys, err := datastore.NewQuery("pkg").Ancestor(build).KeysOnly().GetAll(ctx, nil)
	if err != nil {
		log.Errorf(ctx, "failed to get pkg records for build %q: %v", build, err)
		return
	}

	err = dsbatch.DeleteMulti(ctx, keys)
	if err != nil {
		log.Errorf(ctx, "error deleting records for build %q: %v", build, err)
		return
	}
	log.Infof(ctx, "Deleted %v records for build %q", len(keys), build)
})
