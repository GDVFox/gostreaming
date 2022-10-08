package config

import (
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/GDVFox/gostreaming/util/storage"
)

// Conf глобальный конфиг синглтон.
var Conf = NewConfig()

// Config конфигурация сервиса.
type Config struct {
	HTTP     *httplib.HTTPConfig `yaml:"http"`
	Logging  *util.LoggingConfig `yaml:"logging"`
	ETCD     *storage.ETCDConfig `yaml:"etcd"`
	Machines []*Machine          `yaml:"machines"`
}

// NewConfig создает конфиг с настройками по-умолчанию
func NewConfig() *Config {
	return &Config{
		HTTP:     httplib.NewtHTTPConfig(),
		Logging:  util.NewLoggingConfig(),
		ETCD:     storage.NewETCDConfig(),
		Machines: make([]*Machine, 0),
	}
}

// Machine отображение Host машины в Port, на котором запущен machine_node
type Machine struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
