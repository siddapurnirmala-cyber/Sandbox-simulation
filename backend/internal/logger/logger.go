package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger(logPath string) {
	// Ensure folder exists
	dir := filepath.Dir(logPath)
	if dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0755)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Write to both stdout and file
	var syncers []zapcore.WriteSyncer
	syncers = append(syncers, zapcore.AddSync(os.Stdout))

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		syncers = append(syncers, zapcore.AddSync(logFile))
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(syncers...)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		zap.DebugLevel,
	)

	Log = zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(Log)
}
