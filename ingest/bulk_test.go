/*-
 * Copyright (c) 2014-2018, 2023-2026
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

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/google/go-cmp/cmp"
)

func toPkgResult(r []ddao.Result) []ddao.PkgResult {
	ret := make([]ddao.PkgResult, len(r))
	for i := range r {
		ret[i].Result = r[i]
	}
	return ret
}

func TestFixUpDependencies(t *testing.T) {
	var tests = []struct {
		name string
		in   []ddao.Result
		want []ddao.Result
	}{
		{
			name: "no change",
			in: []ddao.Result{{
				PkgName:     "a",
				BuildStatus: Failed,
			}},
			want: []ddao.Result{{
				PkgName:     "a",
				BuildStatus: Failed,
			},
			},
		}, {
			name: "indirect failed",
			in: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "b",
				}, {
					PkgName:     "b",
					BuildStatus: Failed,
					FailedDeps:  "some other stuff",
				},
			},
			want: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "b",
				}, {
					PkgName:     "b",
					BuildStatus: Failed,
					Breaks:      1,
				},
			},
		}, {
			name: "depending on indirect failed",
			in: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "b",
				}, {
					PkgName:     "b",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "c",
					BuildStatus: Failed,
					FailedDeps:  "some other stuff",
				},
			},
			want: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "b",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "c",
					BuildStatus: Failed,
					Breaks:      2,
				},
			},
		}, {
			name: "double counting",
			in: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "b c",
				}, {
					PkgName:     "b",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "c",
					BuildStatus: Failed,
					FailedDeps:  "some other stuff",
				},
			},
			want: []ddao.Result{
				{
					PkgName:     "a",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "b",
					BuildStatus: IndirectFailed,
					FailedDeps:  "c",
				}, {
					PkgName:     "c",
					BuildStatus: Failed,
					Breaks:      2,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := toPkgResult(test.want)
			got := toPkgResult(test.in)
			FixUpDependencies(got)
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("FixUpDependencies(): unexpected diff (+got -want)\n%s", diff)
			}
		})
	}
}

const pkgFoo = `
PKGNAME=foo-1.0
BUILD_STATUS=indirect-failed
DEPENDS=bar-2.0
`

const pkgBar = `
PKGNAME=bar-2.0
BUILD_STATUS=failed
DEPENDS=
`

const pkgDoubleIndirect = `
PKGNAME=double-3.0
BUILD_STATUS=indirect-failed
DEPENDS=foo-1.0
`

const pkgFailed = `
PKGNAME=failed-4.0
BUILD_STATUS=prefailed
PKG_FAIL_REASON="failed-4.0 is marked as broken:" "Expects X11R6"
PKG_SKIP_REASON=
`

const pkgSkipped = `
PKGNAME=skipped-5.0
BUILD_STATUS=prefailed
PKG_FAIL_REASON="skipped-5.0 is not available for NetBSD-11.0-x86_64"
PKG_SKIP_REASON=
`

func TestResultsFromReport(t *testing.T) {
	tests := []struct {
		name   string
		report string
		want   []ddao.PkgResult
	}{
		{
			name:   "double-indirect",
			report: pkgDoubleIndirect + pkgFoo + pkgBar,
			want: []ddao.PkgResult{
				{
					Result: ddao.Result{
						PkgName:     "double-3.0",
						BuildStatus: IndirectFailed,
						FailedDeps:  "foo-1.0",
					},
				}, {
					Result: ddao.Result{
						PkgName:     "foo-1.0",
						BuildStatus: IndirectFailed,
						FailedDeps:  "bar-2.0",
					},
				}, {
					Result: ddao.Result{
						PkgName:     "bar-2.0",
						BuildStatus: Failed,
					},
				},
			},
		}, {
			name:   "fail reason",
			report: pkgFailed,
			want: []ddao.PkgResult{
				{
					Result: ddao.Result{
						PkgName:     "failed-4.0",
						BuildStatus: Prefailed,
						FailureMsg: sql.NullString{
							String: "failed-4.0 is marked as broken: Expects X11R6",
							Valid:  true,
						},
					},
				},
			},
		}, {
			name:   "skip reason",
			report: pkgSkipped,
			want: []ddao.PkgResult{
				{
					Result: ddao.Result{
						PkgName:     "skipped-5.0",
						BuildStatus: Prefailed,
						FailureMsg: sql.NullString{
							String: "skipped-5.0 is not available for NetBSD-11.0-x86_64",
							Valid:  true,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ResultsFromReport(strings.NewReader(tc.report))
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("ResultsFromReport(): unexpected diff (+got -want)\n%s", diff)
			}
		})
	}
}

func TestVariablesJSONPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			"http://localhost:9876/report.txt",
			"http://localhost:9876/variables.json",
		},
		{
			"https://reports.pkgci.org/Darwin/14.5/arm64/20260409T193201Z/report.html",
			"https://reports.pkgci.org/Darwin/14.5/arm64/20260409T193201Z/variables.json",
		},
	}

	for _, tc := range tests {
		got, err := variablesJSONpath(tc.url)
		if err != nil {
			t.Errorf("variablesJSONPath(%q): %v", tc.url, err)
			continue
		}
		if got != tc.want {
			t.Errorf("variablesJSONPath(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
