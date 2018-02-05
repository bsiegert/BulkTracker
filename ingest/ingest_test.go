/*-
 * Copyright (c) 2014-2018
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

package ingest

import "testing"

var fileSuffixTests = [][2]string{
	{"foo", ""},
	{"foo.gz", "gz"},
	{"foo.bz2", "bz2"},
	{"foo.", ""},
	{"https://www.example.com/index.html", "html"},
	{"https://www.example.com/", ""},
}

func TestFileSuffix(t *testing.T) {
	for _, test := range fileSuffixTests {
		if got, want := fileSuffix(test[0]), test[1]; got != want {
			t.Errorf("fileSuffix(%q): got %q, want %q", test[0], got, want)
		}
	}
}
