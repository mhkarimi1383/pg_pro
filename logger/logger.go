package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func init() {
	var err error
	logger, err = zap.NewProduction(zap.AddCallerSkip(1))
	if err != nil {
		panic(errors.Wrap(err, "initializing logger"))
	}
}

func Panic(msg string, fields ...zapcore.Field) {
	logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).Fatal(
		msg,
		fields...,
	)
}

func Info(msg string, fields ...zapcore.Field) {
	logger.Info(
		msg,
		fields...,
	)
}

func Debug(msg string, fields ...zapcore.Field) {
	logger.Debug(
		msg,
		fields...,
	)
}

func Warn(msg string, fields ...zapcore.Field) {
	logger.Warn(
		msg,
		fields...,
	)
}

func Sync() {
	logger.Sync()
}
