package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"cloud.google.com/go/datastore"
	"github.com/bsiegert/BulkTracker/bulk"
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
	q := datastore.NewQuery("build").Order("Timestamp").Limit(100)
	iter := client.Run(ctx, q)

	var build bulk.Build

	for i := 0; i < *numResults; {
		key, err := iter.Next(&build)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s (%v on %v)\n", key.Encode(), build.Platform, build.Date())
		i++
	}

	client.Close()
}
