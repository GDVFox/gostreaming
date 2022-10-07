package config

import (
	"time"

	"github.com/GDVFox/gostreaming/util"
)

// Conf глобальный конфиг синглтон.
var Conf = NewConfig()

// Config конфигурация сервиса.
type Config struct {
	HTTP    *util.HTTPConfig    `yaml:"http"`
	Logging *util.LoggingConfig `yaml:"logging"`
	ETCD    *ETCDConfig         `yaml:"etcd"`
}

// NewConfig создает конфиг с настройками по-умолчанию
func NewConfig() *Config {
	return &Config{
		Logging: util.NewLoggingConfig(),
		ETCD:    newETCDConfig(),
	}
}

// ETCDConfig конфигурация подключения к etcd.
type ETCDConfig struct {
	Endpoints []string          `yaml:"endpoints"`
	Timeout   util.Duration     `yaml:"timeout"`
	Retry     *util.RetryConfig `yaml:"retry"`
}

func newETCDConfig() *ETCDConfig {
	return &ETCDConfig{
		Endpoints: []string{},
		Timeout:   util.Duration(1 * time.Second),
		Retry:     util.NewRetryConfig(),
	}
}
