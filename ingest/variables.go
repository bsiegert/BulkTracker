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
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/bsiegert/BulkTracker/ddao"
)

// Variables contains the information from the variables.json file in
// the bob build results.
type Variables struct {
	// Counts contains package counts by state.
	Counts map[string]int64 `json:"counts"`
	// Vars contains the environment used in the build.
	Vars map[string]string `json:"pkgsrc"`
	// Report has information about the bulk build results.
	Report Report `json:"report"`
	// VCS contains the version at which the build was done.
	VCS VCS `json:"vcs"`
}

// Report contains information about bulk build reports in the bob format.
type Report struct {
	Date          string `json:"date"`
	Duration      int    `json:"duration"`
	ReportURL     string `json:"raw"`
	HTMLReportURL string `json:"url"`
}

// VCS contains information about the source version.
type VCS struct {
	Format       string `json:"format"`
	LocalBranch  string `json:"local_branch"`
	RemoteBranch string `json:"remote_branch"`
	RemoteURL    string `json:"remote_url"`
	Revision     string `json:"revision"`
	FullRevision string `json:"revision_full"`
}

// ParseVariables reads the contents of a variables.json file and returns the
// parsed version.
func ParseVariables(r io.Reader) (*Variables, error) {
	var v Variables
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&v)
	return &v, err
}

// ToBulkBuild converts the variables into a bulk build report header.
func (v *Variables) ToBulkBuild(from string) (*ddao.Build, error) {
	buildTS, err := time.Parse("20060102T150405Z0700", v.Report.Date)
	if err != nil {
		return nil, err
	}
	c := v.Counts

	// TODO: consider adding more fields for counts

	b := ddao.Build{
		BuildUser:            from,
		Platform:             buildPlatform(v.Vars),
		BuildTs:              buildTS,
		Branch:               v.VCS.RemoteBranch,
		Compiler:             v.Vars["PKGSRC_COMPILER"],
		ReportUrl:            v.Report.ReportURL,
		NumOk:                c["success"] + c["up-to-date"],
		NumPrefailed:         c["pre-failed"] + c["pre-skipped"],
		NumFailed:            c["failed"] + c["unresolved"],
		NumIndirectFailed:    c["indirect-failed"] + c["indirect-unresolved"],
		NumIndirectPrefailed: c["indirect-pre-failed"] + c["indirect-pre-skipped"],
	}
	return &b, nil
}

func buildPlatform(vars map[string]string) string {
	return fmt.Sprintf("%s %s/%s", vars["OPSYS"], vars["OS_VERSION"], vars["MACHINE_ARCH"])
}
