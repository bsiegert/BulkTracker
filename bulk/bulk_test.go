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
		[]Pkg{Pkg{PkgName: "foo-1.0", BuildStatus: IndirectFailed}},
	},
	{
		pkgFoo + pkgBar,
		[]Pkg{
			Pkg{PkgName: "foo-1.0", BuildStatus: IndirectFailed, FailedDeps: []string{"bar-2.0"}},
			Pkg{PkgName: "bar-2.0", BuildStatus: Failed, Breaks: 1},
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
