package util

import (
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggingConfig параметры логирования
type LoggingConfig struct {
	Logfile      string `yaml:"logfile"`
	Level        string `yaml:"level"`
	TruncateFile bool   `yaml:"truncate-file"`
}

// NewLoggingConfig создает LoggingConfig с настройками по-умолчанию.
func NewLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Logfile:      "stdout",
		Level:        "info",
		TruncateFile: false,
	}
}

// Logger структура, предназначенная для записи логов.
type Logger struct {
	*zap.SugaredLogger
}

// NewLogger создает новый логгер
func NewLogger(cfg *LoggingConfig) (*Logger, error) {
	var err error

	lvl := zap.NewAtomicLevel()
	if err := lvl.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, errors.Wrap(err, "can not set logging level")
	}

	var f *os.File
	if cfg.Logfile == "stdout" {
		f = os.Stdout
	} else {
		flags := os.O_RDWR | os.O_CREATE | os.O_APPEND
		if cfg.TruncateFile {
			flags |= os.O_TRUNC
		}
		f, err = os.OpenFile(cfg.Logfile, flags, 0660)
		if err != nil {
			return nil, errors.Wrap(err, "can not open logfile")
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder

	ws := zapcore.Lock(zapcore.AddSync(f))
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), ws, lvl)
	return &Logger{
		SugaredLogger: zap.New(core).Sugar(),
	}, nil
}

// WithName добавляет имя в путь.
func (l *Logger) WithName(name string) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.Named(name),
	}
}
