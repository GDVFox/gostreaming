package util

import (
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggingConfig параметры логирования
type LoggingConfig struct {
	Logfile string `yaml:"logfile"`
	Level   string `yaml:"level"`
}

// NewLoggingConfig создает LoggingConfig с настройками по-умолчанию.
func NewLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Logfile: "stdout",
		Level:   "info",
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
		f, err = os.Open(cfg.Logfile)
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
