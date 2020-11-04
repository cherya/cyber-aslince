package logger

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

var once sync.Once

// Init initialize logrus
func Init(logLevel log.Level, timeFormat string) {
	once.Do(func() {
		log.SetLevel(logLevel)

		Formatter := new(log.TextFormatter)
		Formatter.TimestampFormat = timeFormat //"02-01-2006 15:04:05"
		Formatter.FullTimestamp = true
		log.SetFormatter(Formatter)
	})
}

func NewLogger(logLevel log.Level, timeFormat string) *log.Logger {
	l := log.New()
	l.SetLevel(logLevel)
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = timeFormat //"02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	l.SetFormatter(Formatter)

	return l
}