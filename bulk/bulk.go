/*-
 * Copyright (c) 2014-2018, 2022-2024
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

// Package bulk contains data types for handling bulk build reports and their
// metadata. It is not supposed to depend on any App Engine package.
package bulk

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bsiegert/BulkTracker/ddao"
)

// Status of a package build.
const (
	// The package was successfully built.
	OK = iota
	// No attempt was made to build the package
	// (e.g. not available on this platform)
	Prefailed
	// The package failed to build.
	Failed
	// One of the dependencies failed to build.
	IndirectFailed
	// One of the dependencies was prefailed.
	IndirectPrefailed
)

var statuses = map[string]int64{
	"done":               OK,
	"prefailed":          Prefailed,
	"failed":             Failed,
	"indirect-failed":    IndirectFailed,
	"indirect-prefailed": IndirectPrefailed,
}

var ErrParse = errors.New("bulk: parse error")

// BuildFromReport parses the start of a bulk report email to fill in the
// fields.
func BuildFromReport(from string, r io.Reader) (*ddao.Build, error) {
	b := &ddao.Build{BuildUser: from}
	s := bufio.NewScanner(r)
	for {
		if !s.Scan() {
			return nil, s.Err()
		}
		if s.Text() == "pkgsrc bulk build report" {
			break
		}
	}
	if !s.Scan() {
		return nil, s.Err()
	}
	if s.Bytes()[0] != '=' {
		return nil, ErrParse
	}
	if !s.Scan() {
		return nil, s.Err()
	}
	if !s.Scan() {
		return nil, s.Err()
	}
	b.Platform = s.Text()

	for {
		if !s.Scan() {
			break
		}
		if !strings.Contains(s.Text(), ":") {
			continue
		}
		parts := strings.SplitN(s.Text(), ":", 2)
		val := strings.TrimSpace(parts[1])
		switch strings.TrimSpace(parts[0]) {
		case "Compiler":
			b.Compiler = val
		case "Build start":
			b.BuildTs, _ = time.Parse("2006-01-02 15:04", val)
		case "Machine readable version":
			if val == "" {
				if !s.Scan() {
					break
				}
				val = strings.TrimSpace(s.Text())
			}
			b.ReportUrl = val
		case "Successfully built":
			b.NumOk, _ = strconv.ParseInt(val, 10, 64)
		case "Failed to build":
			b.NumFailed, _ = strconv.ParseInt(val, 10, 64)
		case "Depending on failed package":
			b.NumIndirectFailed, _ = strconv.ParseInt(val, 10, 64)
		case "Explicitly broken or masked":
			b.NumPrefailed, _ = strconv.ParseInt(val, 10, 64)
		case "Depending on masked package":
			b.NumIndirectPrefailed, _ = strconv.ParseInt(val, 10, 64)
		}
	}
	return b, s.Err()
}

func ResultsFromReport(r io.Reader) ([]ddao.PkgResult, error) {
	var pkgs []ddao.PkgResult
	var p *ddao.PkgResult
	n := 0

	s := bufio.NewScanner(r)
	for s.Scan() {
		b := s.Bytes()
		split := bytes.IndexRune(b, '=')
		if split == -1 {
			continue
		}
		key, val := b[:split], b[split+1:]
		switch {
		case bytes.Equal(key, []byte("PKGNAME")):
			// Next package, finish the one before.
			pkgs = append(pkgs, ddao.PkgResult{})
			p = &pkgs[n]
			n++
			p.PkgName = string(val)
		case bytes.Equal(key, []byte("PKG_LOCATION")):
			p.Category, p.Dir = path.Split(string(val))
		case bytes.Equal(key, []byte("BUILD_STATUS")):
			p.BuildStatus = statuses[string(val)]
		case bytes.Equal(key, []byte("DEPENDS")):
			p.FailedDeps = string(val)
		}
	}

	return pkgs, s.Err()
}

// FixUpDependencies does another run over all indirect-failed packages and only keeps
// dependencies that actually failed.
func FixUpDependencies(pkgs []ddao.PkgResult) {
	// indices maps package name to its index in pkgs.
	indices := make(map[string]int)
	for i := range pkgs {
		indices[pkgs[i].PkgName] = i
	}

	for i := range pkgs {
		if pkgs[i].BuildStatus != IndirectFailed && pkgs[i].BuildStatus != IndirectPrefailed {
			pkgs[i].FailedDeps = ""
			continue

		}
		failedDeps := strings.Fields(pkgs[i].FailedDeps)
		f := make([]string, 0, len(failedDeps))
		for _, dep := range failedDeps {
			if d, ok := indices[dep]; ok && pkgs[d].BuildStatus != OK {
				f = append(f, dep)
				// TODO: if pkgs[fp] is indirect-failed, add to the counter of
				// _its_ failed dependencies.
				pkgs[d].Breaks++
			}
		}
		if len(f) == 0 {
			f = nil
		}
		pkgs[i].FailedDeps = strings.Join(f, " ")
	}
}
