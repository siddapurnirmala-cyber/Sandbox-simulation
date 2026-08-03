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

	// Production JSON encoder configurations for the log file
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.TimeKey = "timestamp"
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// Colorized Console encoder configurations for the standard output
	consoleEncoderConfig := zap.NewProductionEncoderConfig()
	consoleEncoderConfig.TimeKey = "timestamp"
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Green INFO, Yellow WARN, Red ERROR
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	var cores []zapcore.Core

	// 1. Stdout: Console encoder (human-readable)
	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel))

	// 2. Log File: JSON encoder (machine-readable structured format)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zap.DebugLevel))
	}

	combinedCore := zapcore.NewTee(cores...)

	Log = zap.New(combinedCore, zap.AddCaller())
	zap.ReplaceGlobals(Log)
}
