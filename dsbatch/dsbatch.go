// Package dsbatch contains batch operations for the App Engine datastore.
// Unlike their upstream counterparts, they can handle arbitrary numbers
// of records.
package dsbatch

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"reflect"
)

// Maximum number of records per call to PutMulti.
const MaxPerCall = 500

// ProgressUpdater allows sharing the progress of the operation.
type ProgressUpdater interface {
	UpdateProgress(c context.Context, written int)
}

// TODO(bsiegert) also implement checking maximum request size (currently 1MB).

func PutMulti(c context.Context, keys []*datastore.Key, values interface{}, pu ProgressUpdater) error {
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
		log.Debugf(c, "writing records %d-%d", n, m)
		_, err := datastore.PutMulti(c, k, v.Slice(n, m).Interface())
		pu.UpdateProgress(c, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteMulti(c context.Context, keys []*datastore.Key) error {
	l := len(keys)
	for n := 0; n < l; n += MaxPerCall {
		m := n + MaxPerCall
		if m > l {
			m = l
		}
		k := keys[n:m]
		log.Debugf(c, "deleting records %d-%d", n, m)
		err := datastore.DeleteMulti(c, k)
		if err != nil {
			return err
		}
	}
	return nil
}
