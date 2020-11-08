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
