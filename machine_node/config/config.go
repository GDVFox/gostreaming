package config

import (
	"time"

	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
)

// Conf глобальный конфиг синглтон.
var Conf = NewConfig()

// Config конфигурация сервиса.
type Config struct {
	HTTP    *httplib.HTTPConfig `yaml:"http"`
	Logging *util.LoggingConfig `yaml:"logging"`
	ETCD    *storage.ETCDConfig `yaml:"etcd"`
	Watcher *watcher.Config     `yaml:"watcher"`
	Runtime *RuntimeConfig      `yaml:"runtime"`
}

// NewConfig создает конфиг с настройками по-умолчанию
func NewConfig() *Config {
	return &Config{
		HTTP:    httplib.NewtHTTPConfig(),
		Logging: util.NewLoggingConfig(),
		ETCD:    storage.NewETCDConfig(),
		Watcher: watcher.NewConfig(),
	}
}

// RuntimeConfig набор опций, с которым будут запускаться все рантаймы.
type RuntimeConfig struct {
	// ActionStartRetry настройки ретрая для признания действия
	// запущенным. Если рантайм не ответил за N попыток, то считается
	// что он не запущен.
	ActionStartRetry *util.RetryConfig `yaml:"action-start-retry"`
	// BinaryPath путь к бинарному файлу рантайма.
	BinaryPath string `yaml:"binary-path"`
	// LogsDir путь к директории, в которой будут хранится логи
	// запущенных рантаймов.
	LogsDir string `yaml:"logs-dir"`
	// LogsLevel уровень логирования в загруженных рантаймах
	LogsLevel string `yaml:"logs-level"`
	// Timeout таймаут на операции с рантаймом.
	Timeout util.Duration `yaml:"timeout"`
	// AckPeriod период отправки ack.
	AckPeriod util.Duration `yaml:"ack-period"`
	// AckPeriod период отправки ack.
	ForwardLogDir string `yaml:"forward-log-dir"`
}

// NewRuntimeConfig возвращает RuntimeConfig с настройками по умолчанию.
func NewRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		ActionStartRetry: util.NewRetryConfig(),
		BinaryPath:       "runtime",
		LogsDir:          "runtime-logs",
		LogsLevel:        "info",
		Timeout:          util.Duration(5 * time.Second),
		AckPeriod:        util.Duration(5 * time.Second),
		ForwardLogDir:    "/tmp/gostreaming-log",
	}
}
