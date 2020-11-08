module github.com/bsiegert/BulkTracker/expireold

go 1.15

require (
	cloud.google.com/go/datastore v1.3.0
	github.com/bsiegert/BulkTracker v0.0.0-20201024194704-90c091dc2286
)

replace github.com/bsiegert/BulkTracker => ../