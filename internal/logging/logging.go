package logging

import (
	"context"
	"go.uber.org/zap"
)

var logger *zap.Logger

func SetLogger(l *zap.Logger) {
	logger = l
}

type loggingCtxKey int

const (
	logKey = loggingCtxKey(iota)
)

func FromContextS(ctx context.Context) *zap.SugaredLogger {
	return FromContext(ctx).Sugar()
}

func FromContext(ctx context.Context) *zap.Logger {
	v := ctx.Value(logKey)
	if v == nil {
		return logger
	}
	if vlog, ok := v.(*zap.Logger); ok {
		return vlog
	} else {
		return logger
	}
}

func NewContextS(ctx context.Context, fields ...interface{}) (nctx context.Context) {
	nctx, _ = NewContextSL(ctx, fields...)
	return
}

func NewContextSL(ctx context.Context, fields ...interface{}) (nctx context.Context, slog *zap.SugaredLogger) {
	slog = FromContextS(ctx).With(fields...)
	nctx = context.WithValue(ctx, logKey, slog.Desugar())
	return
}

func CopyContext(from, to context.Context) (nctx context.Context) {
	return context.WithValue(to, logKey, FromContext(from))
}
