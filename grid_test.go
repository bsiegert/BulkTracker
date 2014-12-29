package bulktracker

import (
	//"github.com/bsiegert/BulkTracker"
	"github.com/bsiegert/BulkTracker/bulk"

	"appengine/datastore"
	"testing"
)

// A testingIterator returns a non-nil error after a given amount of calls to Next.
type testingIterator struct {
	i int
}

func (it *testingIterator) Next(dst interface{}) (*datastore.Key, error) {
	if it.i == 0 {
		return nil, datastore.Done
	}
	it.i--
	return &datastore.Key{}, nil
}

func TestMultiIterator(t *testing.T) {
	m := make(MultiIterator, 1)
	m[0].it = &testingIterator{4}
	m[0].Pkg = bulk.Pkg{PkgName: "a"}

	for {
		row, done := m.Next()
		if done {
			break
		}
		t.Logf("%#v", row)
	}
}
