package log

import (
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	log "github.com/sirupsen/logrus"
)

func InitLogger() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(stackdriver.NewFormatter(
		stackdriver.WithStackSkip("github.com/bsiegert/BulkTracker/log"),
	))
	log.Debug("Added Stackdriver Logging")
}
