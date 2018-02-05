/*-
 * Copyright (c) 2014-2018
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

// Package dsbatch contains batch operations for the App Engine datastore.
// Unlike their upstream counterparts, they can handle arbitrary numbers
// of records.
package dsbatch

import (
	"errors"
	"context"
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
