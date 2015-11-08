// Package ingest contains the ingestion pipeline to go from an incoming
// bulk report mail to having its contents fully in the datastore.
package ingest

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
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/mail"
	"sort"
	"strings"
	"time"
)

// All these names mean HEAD.
var headAliases = map[string]bool{
	"current":          true,
	"upstream-trunk32": true,
	"upstream-trunk64": true,
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

	if build == nil {
		return
	}

	subj := msg.Header.Get("Subject")
	s := strings.SplitN(subj, " ", 2)[0]
	if s == "pkgsrc" {
		build.Branch = "HEAD"
	} else if strings.HasPrefix(s, "pkgsrc-") {
		build.Branch = strings.TrimPrefix(s, "pkgsrc-")
	}
	if headAliases[build.Branch] {
		build.Branch = "HEAD"
	}
	c.Debugf("%#v, %s", build, err)

	key, err := datastore.Put(c, datastore.NewIncompleteKey(c, "build", nil), build)
	c.Infof("wrote entry %v: %s", key, err)
	FetchReport.Call(c, key, build.ReportURL)
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
