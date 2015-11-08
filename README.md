# BulkTracker

[![Build Status](https://travis-ci.org/bsiegert/BulkTracker.svg?branch=master)](https://travis-ci.org/bsiegert/BulkTracker)

BulkTracker is a web application to track the status of bulk builds in NetBSD 
[pkgsrc](http://www.pkgsrc.org) on various platforms. It is written in Go and
runs on Google App Engine. The production instance is running at

https://bulktracker.appspot.com/

It uses the App Engine datastore, task queues and the URL Fetch service.
The application is subscribed to the pkgsrc-bulk mailing list and receives
build reports via email. It parses the email and tries to fetch the machine-readable
report from the given URL. The report is split into records and saved in the
datastore. The web UI allows examining aggregate and per-package results.

The main program and the App Engine configuration is located in the `app/`
subdirectory.

This is an open project. Drop me a line if you are interested in participating!
