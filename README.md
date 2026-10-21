# BulkTracker

BulkTracker is a web application to track the status of bulk builds in NetBSD 
[pkgsrc](http://www.pkgsrc.org) on various platforms. It is written in Go and
uses SQLite3 as its database. The production instance is running at

https://releng.netbsd.org/bulktracker/

The application is subscribed to the pkgsrc-bulk mailing list and receives
build reports via email. It parses the email and tries to fetch the
machine-readable report from the given URL. The report is split into records
and saved in the database. The web UI allows examining aggregate and
per-package results.

This is an open project. Drop me a line if you are interested in participating!

## Getting started with development

To go from zero to a running and working test instance, do the following steps
inside the source directory:

```shell
# Create the database
sqlite3 BulkTracker.db < schema.sql
# Build the app
go build -v .
# Start instance
./BulkTracker

# In a second terminal, run the helper tool to insert a build record
cd testing/btinject
go run btinject.go
```
