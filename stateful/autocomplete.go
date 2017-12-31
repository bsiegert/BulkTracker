// Package stateful contains stateful components for BulkTracker, i.e. those
// that keep state across requests.
package stateful

import (
	"context"
	"strings"

	"github.com/bsiegert/BulkTracker/bulk"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

var ch chan AutocompleteRequest

// TODO regularly reload the dataset.

func load(ctx context.Context) error {
	var pkgs []bulk.Pkg
	_, err := datastore.NewQuery("pkg").Project("Category", "Dir").Distinct().GetAll(ctx, &pkgs)
	if err != nil {
		return err
	}

	allPkgNames := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		allPkgNames = append(allPkgNames, p.Category+p.Dir)
	}

	newChan := make(chan AutocompleteRequest)
	go func() {
		for req := range newChan {
			lookup(allPkgNames, req)
		}
	}()
	ch = newChan
	return nil
}

func lookup(allPkgNames []string, req AutocompleteRequest) {
	resp := AutocompleteResponse{}
	for _, p := range allPkgNames {
		if strings.Contains(p, req.Search) {
			resp.Results = append(resp.Results, Result{
				ID:   p,
				Text: p,
			})
		}
	}
	req.Ret <- resp
}

// Result encodes a single line of the select2 JSON response format.
type Result struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// AutocompleteResponse is the overall return type of the select2 JSON response.
type AutocompleteResponse struct {
	Results    []Result `json:"results"`
	Pagination struct {
		More bool `json:"more"`
	} `json:"pagination,omitempty"`
}

// AutocompleteRequest has the parameters for a single autocomplete request.
type AutocompleteRequest struct {
	// ctx is the request context. It must be an App Engine context. It is
	// only used for this one request.
	Ctx context.Context
	// search is the search string.
	Search string
	// ret is the channel in which to return the reponse.
	Ret chan<- AutocompleteResponse
}

// Autocomplete sends an autocomplete request. The response will be delivered
// over the channel in req.Ret. The function returns an error that occurs while
// loading the dataset.
func Autocomplete(req AutocompleteRequest) error {
	if ch == nil {
		log.Infof(req.Ctx, "loading data")
		if err := load(req.Ctx); err != nil {
			return err
		}
		log.Infof(req.Ctx, "done loading data")
	}
	ch <- req
	return nil
}
