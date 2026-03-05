package logger

import (
	"os"

	"go.uber.org/zap/zapcore"
)

func newStdoutSyncer() zapcore.WriteSyncer {
	return zapcore.Lock(os.Stdout)
}
