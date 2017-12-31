// Package stateful contains stateful components for BulkTracker, i.e. those
// that keep state across requests.
package stateful

import (
	"context"
	"fmt"
	"net/url"

	"github.com/bsiegert/BulkTracker/bulk"
	"google.golang.org/appengine/datastore"
)

var allPkgNames []string

func load(ctx context.Context) error {
	var pkgs []bulk.Pkg
	_, err := datastore.NewQuery("pkg").Project("Category", "Dir").Distinct().GetAll(context.Background(), &pkgs)
	if err != nil {
		return err
	}

	result := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		result = append(result, fmt.Sprintf("%s/%s", p.Category, p.Dir))
	}
	allPkgNames = result
	return nil
}

// Result encodes a single line of the select2 JSON response format.
type Result struct {
	ID, Text string
}

// AutocompleteResponse is the overall return type of the select2 JSON response.
type AutocompleteResponse struct {
	Results    []Result
	Pagination struct {
		More bool
	}
}

func Autocomplete(ctx context.Context, _ []string, form url.Values) (interface{}, error) {
	// TODO
	return nil, nil
}
