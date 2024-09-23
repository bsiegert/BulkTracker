/*-
 * Copyright (c) 2014-2018, 2023-2024
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

package bulk

import (
	"strings"
	"testing"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/google/go-cmp/cmp"
)

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

func TestResultsFromReport(t *testing.T) {
	var tests = []struct {
		name   string
		report string
		want   []ddao.PkgResult
	}{
		{
			"single package",
			pkgFoo,
			[]ddao.PkgResult{{
				Result: ddao.Result{
					PkgName:     "foo-1.0",
					BuildStatus: IndirectFailed,
				},
			}},
		},
		{
			"indirect failed",
			pkgFoo + pkgBar,
			[]ddao.PkgResult{
				{
					Result: ddao.Result{
						PkgName:     "foo-1.0",
						BuildStatus: IndirectFailed,
						FailedDeps:  "bar-2.0",
					},
				}, {
					Result: ddao.Result{
						PkgName:     "bar-2.0",
						BuildStatus: Failed,
						Breaks:      1,
					},
				},
			},
		},
		{
			"depending on indirect failed",
			pkgDoubleIndirect + pkgFoo + pkgBar,
			[]ddao.PkgResult{
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
						Breaks:      1,
					},
				}, {
					Result: ddao.Result{
						PkgName:     "bar-2.0",
						BuildStatus: Failed,
						// TODO: arguably, this should be 2, since it breaks the two other packages.
						Breaks: 1,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := ResultsFromReport(strings.NewReader(test.report))
			FixUpDependencies(got)
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("PkgsFromReport(): unexpected diff (+got -want)\n%s", diff)
			}
		})
	}
}
