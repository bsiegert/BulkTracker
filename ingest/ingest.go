/*-
 * Copyright (c) 2014-2023
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

// Package ingest contains the ingestion pipeline to go from an incoming
// bulk report mail to having its contents fully in the datastore.
package ingest

import (
	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/ddao"
	"github.com/bsiegert/BulkTracker/log"
	ftp "github.com/smira/go-ftp-protocol/protocol"
	"github.com/ulikunitz/xz"

	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/mail"
	"strings"
)

// Constants for the current status.
const (
	Fetching = iota
	Failed
	Writing
	// Done: when the Status no longer exists in the datastore.
)

type Status struct {
	URL     string
	Current int // Current status; one of the constants above.
	// If Current == Writing, statistics for how many package records
	// have been written.
	PkgsWritten, PkgsTotal int
	// If Current == Failed, the last error encountered.
	LastErr error

	// key      *datastore.Key `json:"-"`
	// cacheKey string `json:"-"`
}

// NewStatus allocates a new Status for report ingestion. As a side effect,
// it also deletes old records, if any.
func NewStatus(ctx context.Context, buildID int64) *Status {
	// s := &Status{
	// 	key:      datastore.NewIncompleteKey(ctx, "status", build),
	// 	cacheKey: "/json/status/" + build.String(),
	// }

	// // TODO delete from memcache.
	// keys, err := datastore.NewQuery("status").Ancestor(build).KeysOnly().GetAll(ctx, nil)
	// if err != nil {
	// 	log.Warningf(ctx, "failed to query for old statuses: %s", err)
	// 	return s
	// }
	// if len(keys) > 0 {
	// 	log.Infof(ctx, "Deleting %d records", len(keys))
	// 	dsbatch.DeleteMulti(ctx, keys)
	// }
	// return s
	return &Status{}
}

// Put writes s into the datastore and memcache.
func (s *Status) Put(ctx context.Context) {
	// datastore.Put(ctx, s.key, s)

	// var buf bytes.Buffer
	// json.NewEncoder(&buf).Encode(s)
	// err := memcache.Set(ctx, &memcache.Item{
	// 	Key:        s.cacheKey,
	// 	Value:      buf.Bytes(),
	// 	Expiration: 2 * time.Minute,
	// })
	// if err != nil {
	// 	log.Warningf(ctx, "failed to write %q to cache: %s", s.cacheKey, err)
	// }
}

// UpdateProgress sets the # of packages written and calls Put.
func (s *Status) UpdateProgress(ctx context.Context, written int) {
	s.PkgsWritten = written
	s.Put(ctx)
}

// Done marks the ingestion as done by removing the Status entry.
func (s *Status) Done(ctx context.Context) {
	// log.Infof(ctx, "%v", s.key)
	// datastore.Delete(ctx, s.key)
	// TODO delete from memcache.
}

// All these names mean HEAD.
var headAliases = map[string]bool{
	"current":          true,
	"upstream-trunk32": true,
	"upstream-trunk64": true,
}

// IncomingMailHandler provides an endpoint that is called (with a POST request)
// when a new mail comes in. It tries to parse it as a bulk build report and
// ingests it, if successful.
type IncomingMailHandler struct {
	DB *ddao.DB
}

func (i *IncomingMailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msg, err := mail.ReadMessage(r.Body)
	if err != nil {
		log.Errorf(ctx, "failed to read mail message: %s", err)
		w.WriteHeader(500)
		return
	}
	fromAddr, err := msg.Header.AddressList("From")
	if err != nil {
		log.Warningf(ctx, "unable to parse From header: %s", err)
		return
	}
	from := &mail.Address{}
	if len(fromAddr) > 0 {
		from = fromAddr[0]
	}
	log.Infof(ctx, "new mail from %s", from)
	if strings.Contains(from.Address, "majordomo") {
		body, _ := ioutil.ReadAll(msg.Body)
		log.Infof(ctx, "%s", body)
		return
	}
	body, err := ParseMultipartMail(msg)
	if err != nil {
		log.Errorf(ctx, "failed to read mail body: %s", err)
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
	log.Infof(ctx, "%#v, %s", build, err)

	id, err := i.DB.PutBuild(ctx, ddao.PutBuildParams{
		Platform:             build.Platform,
		BuildTs:              build.BuildTs,
		Branch:               build.Branch,
		Compiler:             build.Compiler,
		BuildUser:            build.BuildUser,
		ReportUrl:            build.ReportUrl,
		NumOk:                build.NumOk,
		NumPrefailed:         build.NumPrefailed,
		NumFailed:            build.NumFailed,
		NumIndirectFailed:    build.NumIndirectFailed,
		NumIndirectPrefailed: build.NumIndirectPrefailed,
	})
	log.Infof(ctx, "wrote entry %v: %v", id, err)
	i.FetchReport(ctx, id, build.ReportUrl)
}

// fileSuffix returns the "file type" suffix of the file name, possibly
// containing a full URL.
func fileSuffix(name string) string {
	// Take basename, without removing trailing slashes.
	name = name[strings.LastIndexByte(name, byte('/'))+1:]
	s := strings.LastIndexByte(name, byte('.'))
	if s == -1 {
		return ""
	}
	return name[s+1:]
}

// decompressingReader returns a Reader with the right type of decompression
// wrapper.
func decompressingReader(r io.Reader, url string) (io.Reader, error) {
	switch fileSuffix(url) {
	case "bz2":
		return bzip2.NewReader(r), nil
	case "gz":
		return gzip.NewReader(r)
	case "xz", "lzma":
		return xz.NewReader(r)
	}
	// Uncompressed, or unknown.
	return r, nil
}

// httpGet tries http.Get and falls back to using an App Engine urlfetch
// transport if it fails.
func httpGet(ctx context.Context, url string) (*http.Response, error) {
	transport := &http.Transport{}
	transport.RegisterProtocol("ftp", &ftp.FTPRoundTripper{})
	client := http.Client{
		Transport: transport,
	}
	return client.Get(url)
}

// FetchReport fetches the machine-readable build report, hands it off to the
// parser and writes the result into the datastore.
func (i *IncomingMailHandler) FetchReport(ctx context.Context, buildID int64, url string) {
	status := NewStatus(ctx, buildID)
	status.URL = url
	status.Current = Fetching
	status.Put(ctx)
	resp, err := httpGet(ctx, url)
	if err != nil {
		log.Warningf(ctx, "failed to fetch report at %q: %s", url, err)
		status.LastErr = err
		status.Current = Failed
		status.Put(ctx)
		return
	}
	defer resp.Body.Close()
	r, err := decompressingReader(resp.Body, url)
	if err != nil {
		log.Errorf(ctx, "failed to uncompress report at %q: %s", url, err)
		status.LastErr = err
		status.Current = Failed
		status.Put(ctx)
		return
	}
	pkgs, err := bulk.PkgsFromReport(r)
	if err != nil {
		log.Errorf(ctx, "failed to parse report at %q: %s", url, err)
		status.LastErr = err
		status.Current = Failed
		status.Put(ctx)
		return
	}

	status.Current = Writing
	status.PkgsTotal = len(pkgs)
	// sort.Sort(bulk.PkgsByName(pkgs))
	if err = i.DB.PutResults(ctx, pkgs, buildID); err != nil {
		status.Current = Failed
		status.LastErr = err
		status.Put(ctx)
		log.Warningf(ctx, "%s", err)
	}
	status.Done(ctx)
}

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
