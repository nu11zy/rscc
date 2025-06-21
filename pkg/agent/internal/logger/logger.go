package logger

type Logger interface {
	Info(format string, a ...any)
	Warn(format string, a ...any)
	Error(format string, a ...any)
	Fatal(format string, a ...any)
}
