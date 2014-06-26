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
)

// Build holds aggregate information about a single bulk build.
type Build struct {
	Platform  string
	Timestamp time.Time
	Branch    string
	Compiler  string
	User      string
	ReportURL string
	// The following are aggregate statistics giving
	// the number of packages with each status.
	NumOK, NumPrefailed, NumFailed, NumIndirectFailed, NumIndirectPrefailed int64
}

// Date returns the date part of the build timestamp.
func (b *Build) Date() string {
	return b.Timestamp.Format("2006-01-02")
}

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

var statuses = map[string]int8{
	"done": OK,
	"prefailed": Prefailed,
	"failed": Failed,
	"indirect-failed": IndirectFailed,
	"indirect-prefailed": IndirectPrefailed,
}

var ErrParse = errors.New("bulk: parse error")

// BuildFromReport parses the start of a bulk report email to fill in the
// fields.
func BuildFromReport(r io.Reader) (*Build, error) {
	b := &Build{}
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
		if strings.Index(s.Text(), ":") == -1 { continue }
		parts := strings.SplitN(s.Text(), ":", 2)
		val := strings.TrimSpace(parts[1])
		switch strings.TrimSpace(parts[0]) {
		case "Compiler":
			b.Compiler = val
		case "Build start":
			b.Timestamp, _ = time.Parse("2006-01-02 15:04", val)
		case "Machine readable version":
			if val == "" {
				if !s.Scan() {
					break
				}
				val = strings.TrimSpace(s.Text())
			}
			b.ReportURL = val
			break
		case "Successfully built":
			b.NumOK, _ = strconv.ParseInt(val, 10, 64)
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
// TODO(bsiegert) Does this need a field for the build key,
// or is the datastore "ancestor" enough?
type Pkg struct {
	//Build *datastore.Key
	// The first and last part of the package location. For example,
	// if the location is "devel/libtool", Category would be "devel"
	// and Dir "libtool".
	Category, Dir string
	PkgName       string
	BuildStatus   int8
	// This is probably not needed, Category+Dir+PkgName should be unique.
	//MultiVersion string
	//Depends []string `datastore:",noindex"`
	// Dependencies are not important, only the failed ones for
	// indirect-failed packages, and the _number_ of breaking packages for
	// failed ones.
	FailedDeps []string
	// Number of packages broken by this one.
	Breaks int
}

func get(pkgs []Pkg, pkgname string) (*Pkg, bool) {
	for i := range pkgs {
		if pkgs[i].PkgName == pkgname {
			return &pkgs[i], true
		}
	}
	return nil, false
}

func PkgsFromReport(r io.Reader) ([]Pkg, error) {
	var pkgs []Pkg
	// Failed packages. The key is the name, the value an index into pkgs.
	var failedPkgs = make(map[string]int)
	var p *Pkg
	n := 0

	s := bufio.NewScanner(r)
	for {
		if !s.Scan() {
			break
		}
		b := s.Bytes()
		split := bytes.IndexRune(b, '=')
		if split == -1 { continue }
		key, val := b[:split], b[split+1:]
		switch {
		case bytes.Equal(key, []byte("PKGNAME")):
			// Next package, finish the one before.
			pkgs = append(pkgs, Pkg{})
			p = &pkgs[n]
			n++
			p.PkgName = string(val)
		case bytes.Equal(key, []byte("PKG_LOCATION")):
			p.Category, p.Dir = path.Split(string(val))
		case bytes.Equal(key, []byte("BUILD_STATUS")):
			p.BuildStatus = statuses[string(val)]
			switch p.BuildStatus {
			case Failed, Prefailed:
				failedPkgs[p.PkgName] = n-1
			}
		case bytes.Equal(key, []byte("DEPENDS")):
			p.FailedDeps = strings.Fields(string(val))
		}
	}
	// Do another run over all indirect-failed packages, only keep
	// dependencies that actually failed.
	for i := range pkgs {
		if pkgs[i].BuildStatus != IndirectFailed && pkgs[i].BuildStatus != IndirectPrefailed {
			pkgs[i].FailedDeps = nil
		}
		f := make([]string, 0, len(pkgs[i].FailedDeps))
		for _, dep := range pkgs[i].FailedDeps {
			if fp, ok := failedPkgs[dep]; ok {
				f = append(f, dep)
				pkgs[fp].Breaks++
			}
		}
		if len(f) == 0 {
			f = nil
		}
		pkgs[i].FailedDeps = f
	}
	return pkgs, s.Err()
}
