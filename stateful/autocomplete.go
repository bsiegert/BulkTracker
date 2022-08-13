/*-
 * Copyright (c) 2014-2018, 2021-2022
 *      Benny Siegert <bsiegert@gmail.com>
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

// Package stateful contains stateful components for BulkTracker, i.e. those
// that keep state across requests.
package stateful

import (
	"context"
)

// MaybePrefillCache calls PrefillCache opportunistically if the item is not
// in memcache yet.
func MaybePrefillCache(ctx context.Context) {}

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
