package config

import (
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
)

// Conf глобальный конфиг синглтон.
var Conf = NewConfig()

// Config конфигурация сервиса.
type Config struct {
	HTTP    *HTTPConfig                `yaml:"http"`
	Logging *util.LoggingConfig        `yaml:"logging"`
	ETCD    *storage.ETCDConfig        `yaml:"etcd"`
	Watcher *watcher.PlanWatcherConfig `yaml:"watcher"`
}

// NewConfig создает конфиг с настройками по-умолчанию
func NewConfig() *Config {
	return &Config{
		HTTP:    newHTTPConfig(),
		Logging: util.NewLoggingConfig(),
		ETCD:    storage.NewETCDConfig(),
		Watcher: watcher.NewPlanWatcherConfig(),
	}
}

// HTTPConfig конфигурация HTTP сервера
type HTTPConfig struct {
	*httplib.HTTPConfig `yaml:",inline"`
	DashboardIP         string `yaml:"dashboard_ip"`
}

func newHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		HTTPConfig:  httplib.NewtHTTPConfig(),
		DashboardIP: "127.0.0.1",
	}
}
