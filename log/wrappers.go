package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

func Debugf(ctx context.Context, msg string, args ...interface{}) {
	logrus.WithContext(ctx).Debugf(msg, args...)
}

func Infof(ctx context.Context, msg string, args ...interface{}) {
	logrus.WithContext(ctx).Infof(msg, args...)
}

func Warningf(ctx context.Context, msg string, args ...interface{}) {
	logrus.WithContext(ctx).Warningf(msg, args...)
}

func Errorf(ctx context.Context, msg string, args ...interface{}) {
	logrus.WithContext(ctx).Errorf(msg, args...)
}
