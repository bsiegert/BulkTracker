/*-
 * Copyright (c) 2023
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

package pages

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPkgDetailsArg(t *testing.T) {
	var tests = []struct {
		path     string
		resultID int64
		wantErr  bool
	}{
		{
			path:     "/pkg/1234",
			resultID: 1234,
		}, {
			path:     "/bulktracker/pkg/4321",
			resultID: 4321,
		}, {
			path:    "/pkg/",
			wantErr: true,
		}, {
			path:    "/pkg/snacks",
			wantErr: true,
		},
	}

	p := PkgDetails{}
	for _, tc := range tests {
		r, err := http.NewRequest(http.MethodGet, "http://localhost"+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := p.arg(r)
		if tc.wantErr && err == nil {
			t.Errorf("arg(%q) expected to return an error but didn't", tc.path)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("arg(%q): %v", tc.path, err)
		}
		if want := tc.resultID; got != want {
			t.Errorf("arg(%q): got %v, want %v", tc.path, got, want)
		}
	}
}

func TestDirArgs(t *testing.T) {
	var tests = []struct {
		basePath string
		path     string
		want     []string
	}{
		{
			basePath: "/",
			path:     "/foo/",
			want:     []string{"foo"},
		}, {
			basePath: "/",
			path:     "/",
			want:     []string{""},
		}, {
			basePath: "/bulktracker",
			path:     "/net/wget",
			want:     []string{"net", "wget"},
		},
	}

	for _, tc := range tests {
		d := Dirs{
			BasePath: tc.basePath,
		}
		r, err := http.NewRequest(http.MethodGet, "http://localhost"+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := d.args(r)
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("args(%q): unexpected result (+got -want):\n%s", tc.path, diff)
		}
	}
}
