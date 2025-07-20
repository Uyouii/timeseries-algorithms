package utils

import (
	"context"
	"runtime"

	"go.uber.org/zap"
)

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func GetLogger(ctx context.Context) *zap.Logger {
	return zap.L()
}

func GetPanicInfo() string {
	buf := make([]byte, 16384)
	l := runtime.Stack(buf, false)
	return string(buf[:l])
}
