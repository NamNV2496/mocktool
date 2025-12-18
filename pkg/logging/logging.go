package logging

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const (
	traceIDKey ctxKey = "trace_id"
	prefixKey  ctxKey = "prefix"
	loggerKey  ctxKey = "logger"
)

type Logger struct {
	logger *zap.SugaredLogger
}

var defaultLogger *Logger

func init() {
	defaultLogger = newDefaultLogger("")
}

func newDefaultLogger(name string) *Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderConfig

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{
		logger: l.Named(name).Sugar(),
	}
}

// override name
func SetName(name string) { defaultLogger.logger.Named(name) }

// ===== inject trace id =====
func InjectTraceId(ctx context.Context) context.Context {
	trace := uuid.New().String()
	logger := defaultLogger.logger.With("trace_id", trace)

	ctx = context.WithValue(ctx, traceIDKey, trace)
	ctx = context.WithValue(ctx, loggerKey, logger)
	return ctx
}

func getLogger(ctx context.Context) *zap.SugaredLogger {
	if l, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return l
	}
	return defaultLogger.logger
}

// ===== prefix logic =====
func ResetPrefix(ctx context.Context, funcName string) context.Context {
	return context.WithValue(ctx, prefixKey, "["+funcName+"] ")
}

func AppendPrefix(ctx context.Context, funcName string) context.Context {
	old, _ := ctx.Value(prefixKey).(string)
	newPrefix := old + "[" + funcName + "] "
	return context.WithValue(ctx, prefixKey, newPrefix)
}

func getPrefix(ctx context.Context) string {
	p, _ := ctx.Value(prefixKey).(string)
	return p
}

// ===== log funcs =====
func Infof(ctx context.Context, template string, args ...any) {
	prefix := getPrefix(ctx)
	getLogger(ctx).Infof(prefix + fmt.Sprintf(template, args...))
}

func Debugf(ctx context.Context, template string, args ...any) {
	prefix := getPrefix(ctx)
	getLogger(ctx).Debugf(prefix + fmt.Sprintf(template, args...))
}
func Errorf(ctx context.Context, template string, args ...any) {
	prefix := getPrefix(ctx)
	getLogger(ctx).Errorf(prefix + fmt.Sprintf(template, args...))
}
