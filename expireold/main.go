/*-
 * Copyright (c) 2020-2022
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
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/expireold/dsbatch"
	"google.golang.org/api/iterator"
)

var (
	numResults  = flag.Int("n", 1, "Number of results")
	numParallel = flag.Int("parallel", 2, "Number of parallel datastore calls")
	minAge      = flag.Duration("min_age", 365*24*time.Hour, "Minimum age for expiring builds")
)

var ErrNoDetails = errors.New("no details")

func main() {
	flag.Parse()
	ctx := context.Background()

	client, err := datastore.NewClient(ctx, "bulktracker")
	if err != nil {
		log.Fatal(err)
	}
	// Oldest builds.
	q := OldestBuilds()
	iter := client.Run(ctx, q)

	var build bulk.Build
	keys := make([]*datastore.Key, 0, *numResults)

	for i := 0; i < *numResults; {
		key, err := iter.Next(&build)
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s (%v on %v)\n", key.Encode(), build.Platform, build.Date())

		if time.Since(build.Timestamp) < *minAge {
			fmt.Println("\ttoo new, stopping")
			break
		}

		keys = append(keys, key)
		err = RemoveDetails(ctx, client, key)
		if err == ErrNoDetails {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		i++
	}

	fmt.Printf("Deleting %d builds\n", len(keys))
	err = dsbatch.DeleteMulti(ctx, client, *numParallel, keys)
	if err != nil {
		log.Fatal(err)
	}

	client.Close()
}

func OldestBuilds() *datastore.Query {
	return datastore.NewQuery("build").Order("Timestamp").Limit(1000)
}

func RemoveDetails(ctx context.Context, client *datastore.Client, buildKey *datastore.Key) error {
	startTime := time.Now()
	q := datastore.NewQuery("pkg").Ancestor(buildKey).KeysOnly()
	keys, err := client.GetAll(ctx, q, nil)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return ErrNoDetails
	}
	fmt.Printf("\tRemoving %v records (%s) ... ", len(keys), time.Since(startTime))
	err = dsbatch.DeleteMulti(ctx, client, *numParallel, keys)
	if err != nil {
		return err
	}
	fmt.Printf("done in %s.\n", time.Since(startTime))
	return nil
}
