//go:build debug
// +build debug

package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var inf = fmt.Sprintf("[%s] ", color.New(color.FgBlue).Sprint("INF"))
var wrn = fmt.Sprintf("[%s] ", color.New(color.FgYellow).Sprint("WRN"))
var err = fmt.Sprintf("[%s] ", color.New(color.FgRed).Sprint("ERR"))
var ftl = fmt.Sprintf("[%s] ", color.New(color.FgRed).Sprint("FTL"))

type debugLogger struct{}

func (l debugLogger) Info(format string, a ...any) {
	fmt.Printf(inf+format+"\n", a...)
}

func (l debugLogger) Warn(format string, a ...any) {
	fmt.Printf(wrn+format+"\n", a...)
}

func (l debugLogger) Error(format string, a ...any) {
	fmt.Printf(err+format+"\n", a...)
}

func (l debugLogger) Fatal(format string, a ...any) {
	fmt.Printf(ftl+format+"\n", a...)
	os.Exit(1)
}

func GetLogger() Logger {
	return debugLogger{}
}
