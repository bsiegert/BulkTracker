/*-
 * Copyright (c) 2020
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
	"flag"
	"fmt"
	"log"

	"cloud.google.com/go/datastore"
	"github.com/bsiegert/BulkTracker/bulk"
	"google.golang.org/api/iterator"
)

var numResults = flag.Int("n", 5, "Number of results")

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

	for i := 0; i < *numResults; {
		key, err := iter.Next(&build)
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s (%v on %v)\n", key.Encode(), build.Platform, build.Date())
		if HasDetails(ctx, client, key) {
			fmt.Println("\thas detailed records")
			i++
		}
	}

	client.Close()
}

func OldestBuilds() *datastore.Query {
	return datastore.NewQuery("build").Order("Timestamp").Limit(100)
}

func HasDetails(ctx context.Context, client *datastore.Client, buildKey *datastore.Key) bool {
	q := datastore.NewQuery("pkg").Ancestor(buildKey).KeysOnly().Limit(1)
	keys, err := client.GetAll(ctx, q, nil)
	if err != nil {
		log.Fatal(err)
	}
	return len(keys) > 0
}
