package bulktracker

import (
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/dsbatch"

	"appengine"
	"appengine/datastore"
	"appengine/delay"
	"appengine/urlfetch"

	"compress/bzip2"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/mail"
	"path"
	"sort"
	"strings"
	"time"
)

func init() {
	http.HandleFunc("/", StartPage)
	http.HandleFunc("/builds", ShowBuilds)
	http.HandleFunc("/build/", BuildDetails)
	http.HandleFunc("/grid", ShowGrid) // in grid.go
	http.HandleFunc("/pkg/", PkgDetails)
	http.HandleFunc("/_ah/mail/", HandleIncomingMail)
}

func StartPage(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, PageHeader)
	defer io.WriteString(w, PageFooter)
	io.WriteString(w, StartPageLead)

	c := appengine.NewContext(r)
	it := datastore.NewQuery("build").Order("-Timestamp").Limit(10).Run(c)
	writeBuildList(c, w, it)
}

func ShowBuilds(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, PageHeader)
	defer io.WriteString(w, PageFooter)
	Heading.Execute(w, "List of Builds")

	c := appengine.NewContext(r)
	it := datastore.NewQuery("build").Order("-Timestamp").Run(c)
	writeBuildList(c, w, it)
	DataTable.Execute(w, nil)
}

func writeBuildList(c appengine.Context, w http.ResponseWriter, it *datastore.Iterator) {
	TableBegin.Execute(w, []string{"Date", "Branch", "Platform", "Stats", "User"})
	b := &bulk.Build{}
	for {
		key, err := it.Next(b)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Errorf("failed to read build: %s", err)
			w.WriteHeader(500)
			return
		}
		b.Key = key.Encode()
		TableBuilds.Execute(w, b)
	}
	io.WriteString(w, TableEnd)
}

// writePackageList writes a table of package results from the iterator it to w.
func writePackageList(c appengine.Context, w http.ResponseWriter, it *datastore.Iterator) {
	TableBegin.Execute(w, []string{"Location", "Package Name", "Status", "Breaks"})
	p := &bulk.Pkg{}
	for {
		key, err := it.Next(p)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Errorf("failed to read pkg: %s", err)
			w.WriteHeader(500)
			return
		}
		p.Key = key.Encode()
		TablePkgs.Execute(w, p)
	}
	io.WriteString(w, TableEnd)
}

func BuildDetails(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	io.WriteString(w, PageHeader)
	defer io.WriteString(w, PageFooter)

	paths := strings.Split(strings.TrimPrefix(r.URL.Path, "/build/"), "/")
	if len(paths) == 0 {
		return
	}
	key, err := datastore.DecodeKey(paths[0])
	if err != nil {
		c.Warningf("error decoding key: %s", err)
		return
	}

	b := &bulk.Build{}
	err = datastore.Get(c, key, b)
	if err != nil {
		c.Warningf("getting build record: %s", err)
		return
	}
	BulkBuildInfo.Execute(w, b)
	if r.URL.Query().Get("a") == "reindex" {
		FetchReport.Call(c, key, b.ReportURL)
		fmt.Fprintf(w, ReindexOK)
		return
	}

	DataTable.Execute(w, template.JS(`"order": [3, "desc"]`))

	if len(paths) > 1 {
		category := paths[1] + "/"
		it := datastore.NewQuery("pkg").Ancestor(key).Filter("Category =", category).Order("Dir").Order("PkgName").Run(c)
		Heading.Execute(w, category)
		writePackageList(c, w, it)
		return
	}

	var categories []bulk.Pkg
	_, err = datastore.NewQuery("pkg").Ancestor(key).Project("Category").Distinct().GetAll(c, &categories)
	if len(categories) == 0 {
		NoDetails.Execute(w, r.URL.Path)
		return
	}
	CategoryList.Execute(w, struct {
		Categories []bulk.Pkg
		CurrentURL string
	}{categories, r.URL.Path})

	Heading.Execute(w, "Packages the breaking most other packages")

	it := datastore.NewQuery("pkg").Ancestor(key).Filter("BuildStatus >", bulk.Prefailed).Order("BuildStatus").Order("-Breaks").Run(c)
	writePackageList(c, w, it)
}

func PkgDetails(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	io.WriteString(w, PageHeader)
	defer io.WriteString(w, PageFooter)

	pkgKey, err := datastore.DecodeKey(path.Base(r.URL.Path))
	if err != nil {
		c.Warningf("error decoding pkg key: %s", err)
		return
	}
	buildKey := pkgKey.Parent()

	p := &bulk.Pkg{}
	b := &bulk.Build{}
	if err = datastore.Get(c, pkgKey, p); err != nil {
		c.Warningf("getting pkg record: %s", err)
		return
	}
	if buildKey != nil {
		if err = datastore.Get(c, buildKey, b); err != nil {
			c.Warningf("getting build record: %s", err)
			return
		}
	}

	PkgInfo.Execute(w, struct {
		Pkg   *bulk.Pkg
		Build *bulk.Build
	}{p, b})

	DataTable.Execute(w, nil)

	// Failed, breaking other packages.
	if p.Breaks > 0 {
		fmt.Fprintf(w, "<h2>This package breaks %d others:</h2>", p.Breaks)
		it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("FailedDeps =", p.PkgName).Order("Category").Order("Dir").Run(c)
		writePackageList(c, w, it)
	}

	// Failed to build because of dependencies.
	if p.FailedDeps == nil {
		return
	}
	fmt.Fprintf(w, "<h2>This package has %d failed dependencies:</h2>", len(p.FailedDeps))
	// TODO(bsiegert) Unfortunately, we save a list of package names, not a
	// list of corresponding datastore keys. So we need to fetch them one by
	// one.
	TableBegin.Execute(w, []string{"Location", "Package Name", "Status", "Breaks"})
	dp := &bulk.Pkg{}
	for _, dep := range p.FailedDeps {
		it := datastore.NewQuery("pkg").Ancestor(buildKey).Filter("PkgName =", dep).Limit(1).Run(c)
		key, err := it.Next(dp)
		if err != nil {
			continue
		}
		dp.Key = key.Encode()
		TablePkgs.Execute(w, dp)
	}
	io.WriteString(w, TableEnd)

}

func HandleIncomingMail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	msg, err := mail.ReadMessage(r.Body)
	if err != nil {
		c.Errorf("failed to read mail message: %s", err)
		w.WriteHeader(500)
		return
	}
	fromAddr, err := msg.Header.AddressList("From")
	if err != nil {
		c.Warningf("unable to parse From header: %s", err)
		return
	}
	from := &mail.Address{}
	if len(fromAddr) > 0 {
		from = fromAddr[0]
	}
	c.Infof("new mail from %s", from)
	if strings.Index(from.Address, "majordomo") != -1 {
		body, _ := ioutil.ReadAll(msg.Body)
		c.Infof("%s", body)
		return
	}
	body, err := ParseMultipartMail(msg)
	if err != nil {
		c.Errorf("failed to read mail body: %s", err)
		return
	}
	fromName := from.Name
	if fromName == "" {
		fromName = strings.SplitN(from.Address, "@", 2)[0]
	}
	build, err := bulk.BuildFromReport(fromName, body)
	c.Debugf("%#v, %s", build, err)

	if build != nil {
		key, err := datastore.Put(c, datastore.NewIncompleteKey(c, "build", nil), build)
		c.Infof("wrote entry %v: %s", key, err)
		FetchReport.Call(c, key, build.ReportURL)
	}
}

// FetchReport fetches the machine-readable build report, hands it off to the
// parser and writes the result into the datastore.
var FetchReport = delay.Func("FetchReport", func(c appengine.Context, build *datastore.Key, url string) {
	client := http.Client{
		Transport: &urlfetch.Transport{
			Context:  c,
			Deadline: time.Minute,
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		c.Warningf("failed to fetch report at %q: %s", url, err)
		return
	}
	defer resp.Body.Close()
	r := io.Reader(resp.Body)
	if strings.HasSuffix(url, ".bz2") {
		r = bzip2.NewReader(resp.Body)
	}
	pkgs, err := bulk.PkgsFromReport(r)
	if err != nil {
		c.Errorf("failed to parse report at %q: %s", url, err)
		return
	}
	sort.Sort(bulk.PkgsByName(pkgs))
	keys, err := datastore.NewQuery("pkg").Ancestor(build).Order("PkgName").KeysOnly().GetAll(c, nil)
	if err != nil {
		c.Errorf("getting current records: %s", err)
		return
	}
	for i := len(keys); i < len(pkgs); i++ {
		keys = append(keys, datastore.NewIncompleteKey(c, "pkg", build))
	}
	if k, p := len(keys), len(pkgs); k > p {
		dsbatch.DeleteMulti(c, keys[p:k])
		keys = keys[:p]
	}
	if err = dsbatch.PutMulti(c, keys, pkgs); err != nil {
		c.Warningf("%s", err)
	}
})

// ParseMultipartMail parses an email and returns a reader for the first
// text/plain element. If the message is not in multipart format, returns the
// whole message body.
func ParseMultipartMail(msg *mail.Message) (io.Reader, error) {
	ctype := msg.Header.Get("Content-Type")
	if ctype == "" || !strings.HasPrefix(ctype, "multipart") {
		return msg.Body, nil
	}
	parts := strings.SplitN(ctype, "; ", 2)
	if parts[0] != "multipart/alternative" {
		return nil, fmt.Errorf("unexpected content type: %s", parts[0])
	}
	var boundary string
	if strings.HasPrefix(parts[1], `boundary="`) {
		boundary = parts[1][10:]
	}
	if i := strings.Index(boundary, `"`); i != -1 {
		boundary = boundary[:i]
	}
	mr := multipart.NewReader(msg.Body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding multipart: %s", err)
		}
		ctype = part.Header.Get("Content-Type")
		if ctype == "" || strings.HasPrefix(ctype, "text/plain") {
			rr := io.Reader(part)
			if part.Header.Get("Content-Transfer-Encoding") == "base64" {
				rr = base64.NewDecoder(base64.StdEncoding, part)
			}
			return rr, nil
		}
	}
	return nil, nil // XXX
}
