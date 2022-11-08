/*-
 * Copyright (c) 2022
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

package dao

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	_ "github.com/mattn/go-sqlite3"
)

func setup(t *testing.T) *DB {
	t.Helper()

	tempfile, err := os.CreateTemp("", "bulktracker*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Remove(tempfile.Name())
		tempfile.Close()
	})

	schema, err := os.Open("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sqlite3", tempfile.Name())
	cmd.Stdin = schema
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	db, err := New(ctx, "sqlite3", tempfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestNew(t *testing.T) {
	setup(t)
}

var compareBuilds = cmpopts.IgnoreFields(bulk.Build{}, "Key", "BuildID")

func TestGetPutBuild(t *testing.T) {
	myBuilds := []*bulk.Build{
		{
			Platform:             "Linux",
			Timestamp:            time.Now(),
			Branch:               "HEAD",
			Compiler:             "gcc",
			User:                 "a@b.com",
			ReportURL:            "",
			NumOK:                12345,
			NumPrefailed:         9,
			NumFailed:            87,
			NumIndirectFailed:    65,
			NumIndirectPrefailed: 43,
		},
	}
	db := setup(t)
	ctx := context.Background()

	t.Logf("Putting %d builds", len(myBuilds))
	var buildIDs []int
	for _, b := range myBuilds {
		id, err := db.PutBuild(ctx, b)
		if err != nil {
			t.Fatal(err)
		}
		buildIDs = append(buildIDs, id)
	}

	t.Logf("Getting %d builds back", len(buildIDs))
	for i := range buildIDs {
		b, err := db.GetBuild(ctx, buildIDs[i])
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(myBuilds[i], b, compareBuilds); diff != "" {
			t.Errorf("[%d] Unexpected diff (-want +got):\n%s", i, diff)
		}
	}
}
