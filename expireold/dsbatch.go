/*-
 * Copyright (c) 2014-2021
 *	Benny Siegert <bsiegert@gmail.com>
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

package main

import (
	"context"
	"strings"

	"cloud.google.com/go/datastore"
	"golang.org/x/sync/errgroup"
)

// Maximum number of records per call to PutMulti.
const MaxPerCall = 500

func DeleteMulti(ctx context.Context, client *datastore.Client, keys []*datastore.Key) error {
	l := len(keys)
	ch := make(chan []*datastore.Key)
	group, cctx := errgroup.WithContext(ctx)

	deleteChunk := func() error {
		for k := range ch {
			_, err := client.RunInTransaction(cctx, func(tx *datastore.Transaction) error {
				return tx.DeleteMulti(k)
			})
			if err != nil {
				return err
			}
		}
		return nil
	}
	for i := 0; i < *numParallel; i++ {
		group.Go(deleteChunk)
	}

	for n := 0; n < l; n += MaxPerCall {
		m := n + MaxPerCall
		if m > l {
			m = l
		}
		k := keys[n:m]
		ch <- k
	}
	close(ch)
	return group.Wait()
}

// rpc error: code = Aborted desc = too much contention on these datastore entities. please try again.

func isRetryable(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "try again") || strings.Contains(errStr, "retry")
}
