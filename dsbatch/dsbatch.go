// Package dsbatch contains batch operations for the App Engine datastore.
// Unlike their upstream counterparts, they can handle arbitrary numbers
// of records.
package dsbatch

import (
	"errors"
	"reflect"

	"appengine"
	"appengine/datastore"
)

// Maximum number of records per call to PutMulti.
const MaxPerCall = 500

// TODO(bsiegert) also implement checking maximum request size (currently 1MB).

func PutMulti(c appengine.Context, keys []*datastore.Key, values interface{}) error {
	v := reflect.ValueOf(values)
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		// OK.
	default:
		return errors.New("PutMulti: value type is not an array or slice")
	}
	l := v.Len()
	for n := 0; n < l; n += MaxPerCall {
		m := n + MaxPerCall
		if m > l {
			m = l
		}
		k := keys[n:m]
		c.Debugf("writing records %d-%d", n, m)
		_, err := datastore.PutMulti(c, k, v.Slice(n, m).Interface())
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteMulti(c appengine.Context, keys []*datastore.Key) error {
	l := len(keys)
	for n := 0; n < l; n += MaxPerCall {
		m := n + MaxPerCall
		if m > l {
			m = l
		}
		k := keys[n:m]
		c.Debugf("deleting records %d-%d", n, m)
		err := datastore.DeleteMulti(c, k)
		if err != nil {
			return err
		}
	}
	return nil
}
