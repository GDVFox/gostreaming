package config

import (
	"strings"

	"github.com/GDVFox/gostreaming/util"
)

// Conf синглтон конфигурации
var Conf = &Config{}

// Config конфигурация для запуска runtime
type Config struct {
	Logger util.LoggingConfig

	ActionPath  string
	Replicas    int
	Port        int
	ServiceSock string
	InRaw       string
	OutRaw      string

	In  []string
	Out []string
}

// Parse загружает данные конфига.
func (c *Config) Parse() error {
	if c.InRaw != "" {
		c.In = strings.Split(c.InRaw, ",")
	}
	if c.OutRaw != "" {
		c.Out = strings.Split(strings.TrimSpace(c.OutRaw), ",")
	}
	return nil
}
