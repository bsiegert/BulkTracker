module github.com/bsiegert/BulkTracker

go 1.15

require (
	cloud.google.com/go/datastore v1.6.0
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/jlaffaye/ftp v0.0.0-20220310202011-d2c44e311e78 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/smira/go-ftp-protocol v0.0.0-20140829150050-066b75c2b70d
	github.com/ulikunitz/xz v0.5.10
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220503163025-988cb79eb6c6 // indirect
	google.golang.org/api v0.63.0
	google.golang.org/appengine v1.6.7
	google.golang.org/protobuf v1.28.0 // indirect
)

// replace google.golang.org/appengine => ../go-appengine
