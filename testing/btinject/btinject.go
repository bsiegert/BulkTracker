// btinject is a tool to inject an entry into the BulkTracker database,
// with no network access needed. It uses the report in the data
// subdirectory.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	dataDir = flag.String("datadir", "data", "Path to the location of test data.")
	report  = flag.String("report", "report.txt", "Path (relative to datadir) to the report mail in text format.")
	wait    = flag.Bool("wait", false, "If true, do not exit after a request has been received.")
)

// dir checks if the data directory exists and returns it for use with net/http.
func dir() (http.Dir, error) {
	st, err := os.Stat(*dataDir)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("data dir %q is not a directory", *dataDir)
	}
	return http.Dir(*dataDir), nil
}

// A fileServer is a http.Handler that serves from Dir and logs each request.
// It signals on ch after a request has been handled.
type fileServer struct {
	Dir http.Dir
	Ch  chan bool
	h   http.Handler
}

func (f fileServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if f.h == nil {
		f.h = http.FileServer(f.Dir)
	}
	log.Print(req)
	f.h.ServeHTTP(w, req)
	f.Ch <- true
}

func readReport() ([]byte, error) {
	fname := *report
	if !filepath.IsAbs(fname) {
		fname = filepath.Join(*dataDir, fname)
	}
	return ioutil.ReadFile(fname)
}

func postReport(body []byte) error {
	log.Print("POST http://localhost:8080/_ah/mail/")
	resp, err := http.Post("http://localhost:8080/_ah/mail/builds@bulktracker.appspotmail.com", "message/rfc822", bytes.NewReader(body))
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s\n%s", resp.Status, rbody)
}

func main() {
	flag.Parse()

	// Prepare an HTTP server. BulkTracker will call back to read the
	// machine-readable report.
	dir, err := dir()
	if err != nil {
		log.Fatal(err)
	}
	// Channel to signal that a request has been handled.
	ch := make(chan bool)
	http.Handle("/", fileServer{Dir: dir, Ch: ch})
	go func() {
		log.Fatal(http.ListenAndServe(":9876", nil))
	}()

	report, err := readReport()
	if err != nil {
		log.Fatal(err)
	}
	err = postReport(report)
	if err != nil {
		log.Print(err)
	}

	// Wait for callback, then exit
	<-ch
	if *wait {
		for {
			<-ch
		}
	}
}
