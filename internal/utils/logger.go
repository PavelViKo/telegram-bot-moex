package utils

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger интерфейс для логирования
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	Sync() error
}

// NewLogger создает новый логгер
func NewLogger(logFile, level string, jsonFormat bool) Logger {
	// Определяем уровень логирования
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	// Создаем конфигурацию
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Выбираем энкодер
	var encoder zapcore.Encoder
	if jsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Создаем писателей
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
		Compress:   true,
	})

	stdoutWriter := zapcore.AddSync(os.Stdout)

	// Создаем ядро логгера
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, writer, zapLevel),
		zapcore.NewCore(encoder, stdoutWriter, zapLevel),
	)

	// Создаем логгер
	logger := zap.New(core, zap.AddCaller())
	return &zapLogger{logger.Sugar()}
}

// zapLogger обертка для zap.SugaredLogger
type zapLogger struct {
	*zap.SugaredLogger
}

// Info логирует информационное сообщение
func (l *zapLogger) Info(msg string, fields ...interface{}) {
	l.SugaredLogger.Infow(msg, fields...)
}

// Error логирует сообщение об ошибке
func (l *zapLogger) Error(msg string, fields ...interface{}) {
	l.SugaredLogger.Errorw(msg, fields...)
}

// Warn логирует предупреждение
func (l *zapLogger) Warn(msg string, fields ...interface{}) {
	l.SugaredLogger.Warnw(msg, fields...)
}

// Debug логирует отладочное сообщение
func (l *zapLogger) Debug(msg string, fields ...interface{}) {
	l.SugaredLogger.Debugw(msg, fields...)
}

// Fatal логирует фатальную ошибку и завершает программу
func (l *zapLogger) Fatal(msg string, fields ...interface{}) {
	l.SugaredLogger.Fatalw(msg, fields...)
}

// Sync синхронизирует логгер
func (l *zapLogger) Sync() error {
	return l.SugaredLogger.Sync()
}
