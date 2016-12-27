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
