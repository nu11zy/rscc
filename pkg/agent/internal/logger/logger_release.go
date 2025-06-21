//go:build !debug
// +build !debug

package logger

import "os"

type releaseLogger struct{}

func (l releaseLogger) Info(format string, a ...any) {}

func (l releaseLogger) Warn(format string, a ...any) {}

func (l releaseLogger) Error(format string, a ...any) {}

func (l releaseLogger) Fatal(format string, a ...any) {
	os.Exit(1)
}

func GetLogger() Logger {
	return releaseLogger{}
}
