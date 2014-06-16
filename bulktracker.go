package bulktracker

import (
	"bulk"

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
	"path"
	"strings"
	"time"
)

// Maximum number of records per call to PutMulti.
const maxRec = 500

func init() {
	http.HandleFunc("/", StartPage)
	http.HandleFunc("/build/", BuildDetails)
	http.HandleFunc("/_ah/mail/", HandleIncomingMail)
}


func StartPage(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, PageHeader)
	io.WriteString(w, StartPageLead)
	TableBegin.Execute(w, []string{"Date", "Platform", "Stats", "User"})
	c := appengine.NewContext(r)
	// TODO(bsiegert) Integrate this better with the template.
	b := &bulk.Build{}
	it := datastore.NewQuery("build").Order("-Timestamp").Limit(10).Run(c)
	for {
		key, err := it.Next(b)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Errorf("failed to read build: %s", err)
			w.WriteHeader(500)
			return
		}
		TableBuilds.Execute(w, struct{
			Key string
			Build *bulk.Build
		}{key.Encode(), b})
	}
	io.WriteString(w, TableEnd)
	io.WriteString(w, PageFooter)
}

func BuildDetails(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	io.WriteString(w, PageHeader)
	io.WriteString(w, "<pre>")
	defer func() {
		io.WriteString(w, "</pre>")
		io.WriteString(w, PageFooter)
	}()
	key, err := datastore.DecodeKey(path.Base(r.URL.Path))
	if err != nil {
		fmt.Fprintf(w, "error decoding key: %s", err)
		return
	}
	b := &bulk.Build{}
	err = datastore.Get(c, key, b)
	if err != nil {
		fmt.Fprintf(w, "getting build record: %s", err)
		return
	}
	fmt.Fprintf(w, "%#v", b)
	if false {// r.URL.Query().Get("a") == "reindex" {
		// Delete all current entries.
		current, err := datastore.NewQuery("pkg").Ancestor(key).KeysOnly().GetAll(c, nil)
		if err != nil {
			fmt.Fprintf(w, "getting current records: %s", err)
			return
		}
		fmt.Fprintf(w, "Deleting %d current records.", len(current))
		err = datastore.DeleteMulti(c, current)
		if err != nil {
			fmt.Fprintf(w, "deleting current records: %s", err)
			return
		}
		FetchReport.Call(c, key, b.ReportURL)
		fmt.Fprintf(w, "\nok\n")
		return
	}
	io.WriteString(w, "</pre><h2>Packages the breaking most other packages</h2><pre>")

	it := datastore.NewQuery("pkg").Ancestor(key).Order("-Breaks").Limit(50).Run(c)
	p := &bulk.Pkg{}
	for {
		_, err := it.Next(p)
		if err == datastore.Done {
			break
		} else if err != nil {
			c.Errorf("failed to read pkg: %s", err)
			w.WriteHeader(500)
			return
		}
		fmt.Fprintf(w, "%#v\n\n", p)
	}
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
	if err == nil && len(fromAddr) > 0 {
		c.Infof("new mail from %s", fromAddr[0])
	}
	if strings.Index(fromAddr[0].Address, "majordomo") != -1 {
		body, _ := ioutil.ReadAll(msg.Body)
		c.Infof("%s", body)
		return
	}
	body, err := ParseMultipartMail(msg)
	if err != nil {
		c.Errorf("failed to read mail body: %s", err)
		w.WriteHeader(500)
		return
	}
	build, err := bulk.BuildFromReport(body)
	c.Infof("%#v, %s", build, err)

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
			Context: c,
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
	pkgs, err := bulk.PkgsFromReport(c, r)
	if err != nil {
		c.Errorf("failed to parse report at %q: %s", url, err)
		return
	}
	keys := make([]*datastore.Key, maxRec)
	for n := 0; n < len(pkgs); n += maxRec {
		m := n + maxRec
		if m > len(pkgs) {
			m = len(pkgs)
		}
		keys = keys[:m-n]
		for i := range keys {
			keys[i] = datastore.NewIncompleteKey(c, "pkg", build)
		}
		c.Infof("inserting records %d-%d", n, m)
		_, err = datastore.PutMulti(c, keys, pkgs[n:m])
		if err != nil {
			c.Warningf("writing pkgs: %s", err)
		}
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
	return nil, nil  // XXX
}
