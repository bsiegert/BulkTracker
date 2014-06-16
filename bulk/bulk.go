package bulk

import (
	"appengine"

	"bufio"
	"bytes"
	"errors"
	"io"
	//"net/url"
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
	NumOK, NumPrefailed, NumFailed, NumIndirectFailed int64
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

func PkgsFromReport(c appengine.Context, r io.Reader) ([]Pkg, error) {
	var pkgs []Pkg
	var failedPkgs = make(map[string]*Pkg)
	var indFailed []*Pkg
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
			if p != nil {
				switch p.BuildStatus {
				case IndirectFailed, IndirectPrefailed:
					indFailed = append(indFailed, p)
				case Failed, Prefailed:
					p.FailedDeps = p.FailedDeps[:0]
					failedPkgs[p.PkgName] = p
				default:
					p.FailedDeps = p.FailedDeps[:0]
				}
			}
			pkgs = append(pkgs, Pkg{})
			p = &pkgs[n]
			n++
			p.PkgName = string(val)
		case bytes.Equal(key, []byte("PKG_LOCATION")):
			p.Category, p.Dir = path.Split(string(val))
		case bytes.Equal(key, []byte("BUILD_STATUS")):
			p.BuildStatus = statuses[string(val)]
			//c.Infof("%s, %d", val, p.BuildStatus)
		case bytes.Equal(key, []byte("DEPENDS")):
			p.FailedDeps = strings.Fields(string(val))
		}
	}
	// Do another run over all indirect-failed packages, only keep
	// dependencies that actually failed.
	for _, p = range indFailed {
		f := make([]string, 0, len(p.FailedDeps))
		for _, dep := range p.FailedDeps {
			if fp, ok := failedPkgs[dep]; ok {
				f = append(f, dep)
				fp.Breaks++
			}
		}
		p.FailedDeps = f
	}
	return pkgs, s.Err()
}
