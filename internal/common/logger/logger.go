package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

var (
	ctxLoggerKey = loggerKey{}
	loggingLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	programStart = time.Now()

	debugColor = color.New(color.FgHiBlack)
	infoColor  = color.New(color.FgBlue)
	warnColor  = color.New(color.FgYellow)
	errorColor = color.New(color.FgRed)
	fatalColor = color.New(color.FgHiRed)
	panicColor = color.New(color.FgHiMagenta)
	nameColor  = color.New(color.FgHiBlue)
)

func NewLogger() (*zap.SugaredLogger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.ConsoleSeparator = " "
	cfg.EncoderConfig.EncodeLevel = consoleColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = consoleTimeAbsEncoder()
	cfg.EncoderConfig.EncodeName = func(s string, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(nameColor.Sprint(s))
	}
	cfg.Level = loggingLevel

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return logger.Sugar(), nil
}

func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey, logger)
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(ctxLoggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return nil
}

func SetDebug() {
	loggingLevel.SetLevel(zap.DebugLevel)
}

func consoleColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString(debugColor.Sprint("D"))
	case zapcore.InfoLevel:
		enc.AppendString(infoColor.Sprint("I"))
	case zapcore.WarnLevel:
		enc.AppendString(warnColor.Sprint("W"))
	case zapcore.ErrorLevel:
		enc.AppendString(errorColor.Sprint("E"))
	case zapcore.FatalLevel:
		enc.AppendString(fatalColor.Sprint("F"))
	case zap.PanicLevel:
		enc.AppendString(panicColor.Sprint("P"))
	default:
		enc.AppendString("U")
	}
}

func consoleTimeAbsEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		timeColor := color.New(color.Faint)
		enc.AppendString(timeColor.Sprintf("%s", time.Now().Format("02/01/2006 15:04:05")))
	}
}

/*
func consoleTimeDeltaEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		duration := t.Sub(programStart)
		seconds := duration / time.Second
		milliseconds := (duration % time.Second) / time.Millisecond
		secColor := color.New(color.Faint)
		msecColor := color.New(color.FgHiBlack)
		enc.AppendString(secColor.Sprintf("%03d", seconds) + msecColor.Sprintf(".%02d", milliseconds/10))
	}
}
*/
