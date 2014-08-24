package bulktracker

import (
	"bulk"

	"appengine"
	"appengine/datastore"

	"bytes"
	"io"
	"net/http"
)


func ShowGrid(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, PageHeader)
	defer io.WriteString(w, PageFooter)
	Heading.Execute(w, "Grid")

	c := appengine.NewContext(r)
	// TODO(bsiegert) Handle selection of build IDs for the grid view.
	// Either as a giantly long URL containing individual build IDs, or as
	// an id of a new BuildIDList entity maybe? Or a memcache datastore
	// key?
	it := datastore.NewQuery("build").Order("-Timestamp").Limit(5).Run(c)
	columns := []string{"Location", "Package Name"}
	b := &bulk.Build{}
	buf := bytes.Buffer{}
	buildkeys := []*datastore.Key{}
	for {
		key, err := it.Next(b)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Errorf("failed to read build: %s", err)
			return
		}
		buf.Reset()
		GridHeader.Execute(&buf, struct {
			Key   string
			Build *bulk.Build
		}{key.Encode(), b})
		columns = append(columns, buf.String())
		buildkeys = append(buildkeys, key)
	}
	TableBegin.Execute(w, columns)

	m := NewMultiIterator(c, buildkeys)
	for {
		row, done := m.Next()
		if done {
			break
		}
		GridEntry.Execute(w, row)
	}

	io.WriteString(w, TableEnd)
}

// PkgResult holds a Pkg and the coresponding datastore key.
// TODO(bsiegert) Move somewhere else.
type PkgResult struct {
	Key string
	Pkg *bulk.Pkg
}

// MultiIterator implements iterating over multiple datastore queries that
// return Pkg elements. All of the queries must be sorted by the same
// criteria.
type MultiIterator []struct {
	it  *datastore.Iterator
	key *datastore.Key
	Pkg bulk.Pkg
}

// NewMultiIterator creates a MultiIterator with queries for the given
// list of ancestors.
func NewMultiIterator(c appengine.Context, ancestors []*datastore.Key) MultiIterator {
	m := make(MultiIterator, len(ancestors))
	for i := range ancestors {
		m[i].it = datastore.NewQuery("pkg").Ancestor(ancestors[i]).Order("Category").Order("Dir").Order("PkgName").Run(c)
	}
	return m
}

// lowestPkgName returns the PkgName that comes first in the sorting order.
// When all iterators are done, it returns the empty string.
func (m MultiIterator) lowestPkgName() string {
	l := ""
	for i := range m {
		if m[i].key == nil {
			if m[i].it == nil {
				continue
			}
			var err error
			m[i].key, err = m[i].it.Next(&m[i].Pkg)
			if err != nil {
				// TODO(bsiegert) This swallows errors other than
				// datastore.Done. However, we would need a context
				// to log errors.
				m[i].it = nil
			}
		}
		current := m[i].Pkg.PkgName
		if l == "" || current < l {
			l = current
		}
	}
	return l
}

// Next returns a row of results. Some of the elements may be nil. The second
// value is true if the iteration is done.
func (m MultiIterator) Next() ([]PkgResult, bool) {
	l := m.lowestPkgName()
	if l == "" {
		return nil, true
	}
	result := make([]PkgResult, len(m))
	for i := range m {
		if m[i].Pkg.PkgName == l {
			result[i].Pkg = &m[i].Pkg
			result[i].Key = m[i].key.Encode()
		}
	}
	return result, false
}
