package log

import (
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	log "github.com/sirupsen/logrus"
)

func InitLogger() {
	log.SetFormatter(stackdriver.NewFormatter())
	log.Info("Added Stackdriver Logging")
}
