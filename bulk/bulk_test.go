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

package bulk

import (
	"reflect"
	"strings"
	"testing"
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

var pkgsFromReportTests = []struct {
	report string
	want   []Pkg
}{
	{
		pkgFoo,
		[]Pkg{{PkgName: "foo-1.0", BuildStatus: IndirectFailed}},
	},
	{
		pkgFoo + pkgBar,
		[]Pkg{
			{PkgName: "foo-1.0", BuildStatus: IndirectFailed, FailedDeps: []string{"bar-2.0"}},
			{PkgName: "bar-2.0", BuildStatus: Failed, Breaks: 1},
		},
	},
}

func TestPkgsFromReport(t *testing.T) {
	for _, test := range pkgsFromReportTests {
		got, _ := PkgsFromReport(strings.NewReader(test.report))
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("PkgsFromReport(%q): got %#v, want %#v", test.report, got, test.want)
		}
	}
}
