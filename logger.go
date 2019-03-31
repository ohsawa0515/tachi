package tachi

import (
	"os"

	"github.com/Sirupsen/logrus"
)

// NewLogger -
func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/1/2 15:04:05 Z07:00",
	})
	log.Out = os.Stderr
	log.Level = logrus.InfoLevel

	return log
}
