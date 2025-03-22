package ddao

import "testing"

func TestBaseURL(t *testing.T) {
	testCases := []struct {
		reportURL string
		want      string
	}{
		{
			"https://example.com/meta/",
			"https://example.com/",
		}, {
			"https://example.com/meta/report.xz",
			"https://example.com/",
		}, {
			"http://shadow.netbsd.org/pub/pkgsrc/packages/reports/HEAD/NetBSD-9.0-i386/20250320.0214/meta/report.xz",
			"https://releng.netbsd.org/pkgreports/shadow/HEAD/NetBSD-9.0-i386/20250320.0214/",
		}, {
			"http://localhost:9876/report.xz",
			"http://localhost:9876/",
		},
	}

	for _, tc := range testCases {
		row := GetSingleResultRow{
			ReportUrl: tc.reportURL,
		}
		got := row.BaseURL()
		if got != tc.want {
			t.Errorf("BaseURL(%q) = %q, want %q", tc.reportURL, got, tc.want)
		}
	}
}
