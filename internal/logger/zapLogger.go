package logger

import (
	"errors"
	"os"

	"go.uber.org/zap"
)

type zapLogger struct {
	l *zap.Logger
}

var _ Logger = (*zapLogger)(nil)

func (z *zapLogger) Sync() error {
	if z == nil || z.l == nil {
		return nil
	}
	err := z.l.Sync()
	if err == nil {
		return nil
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// для dev на stderr можно забить
		return nil
	}

	return err
}

func newZapDevLogger() (Logger, error) {
	l, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	l = l.WithOptions(zap.AddCallerSkip(1)) // на уровень ниже чтобы видеть не обертку а вызывающего
	return &zapLogger{l: l}, nil
}

func newZapProdLogger() (Logger, error) {
	l, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	l = l.WithOptions(zap.AddCallerSkip(1))
	return &zapLogger{l: l}, nil
}

func (z *zapLogger) Named(name string) Logger {
	return &zapLogger{z.l.Named(name)}
}

func (z *zapLogger) Debug(msg string, kv ...any) {
	z.l.Debug(msg, kvToFields(kv...)...)
}

func (z *zapLogger) Info(msg string, kv ...any) {
	z.l.Info(msg, kvToFields(kv...)...)
}

func (z *zapLogger) Warn(msg string, kv ...any) {
	z.l.Warn(msg, kvToFields(kv...)...)
}

func (z *zapLogger) Error(msg string, kv ...any) {
	z.l.Error(msg, kvToFields(kv...)...)
}

func (z *zapLogger) With(kv ...any) Logger {
	child := z.l.With(kvToFields(kv...)...)
	return &zapLogger{l: child}
}

func kvToFields(kv ...any) []zap.Field {
	n := len(kv) / 2
	if n == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, n)

	if len(kv)%2 == 1 {
		// последний без пары
		fields = append(fields, zap.String("_log_error", "odd number of logging arguments"))
	}

	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			fields = append(fields, zap.Any("_bad_key", kv[i])) // метим хуйню

			continue
		}
		value := kv[i+1]
		fields = append(fields, zap.Any(key, value))
	}

	return fields
}
