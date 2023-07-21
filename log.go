package stuble

import (
	"log"
)

type stdLogger struct{}

func (s *stdLogger) Debugf(format string, a ...any) {
	log.Printf(format, a...)
}

func (s *stdLogger) Infof(format string, a ...any) {
	log.Printf(format, a...)
}

func (s *stdLogger) Warnf(format string, a ...any) {
	log.Printf(format, a...)
}

func (s *stdLogger) Errorf(format string, a ...any) {
	log.Printf(format, a...)
}

func (s *stdLogger) Fatalf(format string, a ...any) {
	log.Fatalf(format, a...)
}

var std = &stdLogger{}

func getLogger() Logger {
	return std
}
