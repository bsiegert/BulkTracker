/*-
 * Copyright (c) 2014-2018, 2022-2023
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
	"context"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
)

// Build holds aggregate information about a single bulk build.
type Build = ddao.Build

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
func BuildFromReport(from string, r io.Reader) (*Build, error) {
	b := &Build{BuildUser: from}
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
		log.Infof(context.Background(), "%s", strings.TrimSpace(parts[0]))
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

// Pkg holds a single build result for a package.
type Pkg struct {
	// Key is the string representation of the datastore key of this record.
	Key string `datastore:"-"`
	// PkgID is the numeric ID of this result.
	PkgID int
	// BuildID is the numeric ID of the build that this belongs to.
	BuildID int

	// The first and last part of the package location. For example,
	// if the location is "devel/libtool", Category would be "devel/"
	// and Dir "libtool".
	Category, Dir string
	PkgName       string
	BuildStatus   int8
	// Dependencies are not important, only the failed ones for
	// indirect-failed packages, and the _number_ of breaking packages for
	// failed ones.
	FailedDeps []string
	// Number of packages broken by this one.
	Breaks int
}

func PkgsFromReport(r io.Reader) ([]ddao.PkgResult, error) {
	var pkgs []ddao.PkgResult
	// Failed packages. The key is the name, the value an index into pkgs.
	var failedPkgs = make(map[string]int)
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
			switch p.BuildStatus {
			case Failed, Prefailed:
				failedPkgs[p.PkgName] = n - 1
			}
		case bytes.Equal(key, []byte("DEPENDS")):
			p.FailedDeps = string(val)
		}
	}
	// Do another run over all indirect-failed packages, only keep
	// dependencies that actually failed.
	for i := range pkgs {
		if pkgs[i].BuildStatus != IndirectFailed && pkgs[i].BuildStatus != IndirectPrefailed {
			pkgs[i].FailedDeps = ""
		}
		failedDeps := strings.Fields(pkgs[i].FailedDeps)
		f := make([]string, 0, len(failedDeps))
		for _, dep := range failedDeps {
			if fp, ok := failedPkgs[dep]; ok {
				f = append(f, dep)
				pkgs[fp].Breaks++
			}
		}
		if len(f) == 0 {
			f = nil
		}
		pkgs[i].FailedDeps = strings.Join(f, " ")
	}
	return pkgs, s.Err()
}

// PkgsByName allows sorting a list of Pkgs by their package names.
type PkgsByName []Pkg

func (p PkgsByName) Len() int {
	return len(p)
}

func (p PkgsByName) Less(i, j int) bool {
	return p[i].PkgName < p[j].PkgName
}

func (p PkgsByName) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// PkgResult is a pair of build and package.
type PkgResult struct {
	Build *Build
	Pkg   *Pkg
}
