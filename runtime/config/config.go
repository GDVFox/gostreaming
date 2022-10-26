package config

import (
	"strings"
)

// Conf синглтон конфигурации
var Conf = &Config{}

// Config конфигурация для запуска runtime
type Config struct {
	ActionPath string
	Replicas   int
	Port       int
	InRaw      string
	OutRaw     string

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
