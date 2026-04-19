/*-
 * Copyright (c) 2026
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
	"os"
	"testing"
	"time"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/google/go-cmp/cmp"
)

const variablesJSON = "../testing/btinject/data/bob/variables.json"

func TestParseVariables(t *testing.T) {
	f, err := os.Open(variablesJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	vars, err := ParseVariables(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(vars)
}

func TestToBulkBuild(t *testing.T) {
	f, err := os.Open(variablesJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	vars, err := ParseVariables(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := vars.ToBulkBuild("user@host")
	if err != nil {
		t.Fatal(err)
	}

	want := &ddao.Build{
		BuildUser:            "user@host",
		Platform:             "Darwin 23.6.0/aarch64",
		BuildTs:              time.Date(2026, 4, 9, 19, 32, 01, 0, time.UTC),
		Branch:               "release/macos",
		Compiler:             "clang",
		ReportUrl:            "https://reports.pkgci.org/Darwin/14.5/arm64/20260409T193201Z/report.zst",
		NumOk:                23661,
		NumPrefailed:         726,
		NumFailed:            1753,
		NumIndirectFailed:    2610,
		NumIndirectPrefailed: 190,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ToBulkBuild returned diff (-want +got)\n%s", diff)
	}
}
