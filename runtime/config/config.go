package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GDVFox/gostreaming/util"
)

// Conf синглтон конфигурации
var Conf = &Config{}

// ActionOptions опции для запуска действия
type ActionOptions struct {
	Args []string          `json:"args"`
	Env  map[string]string `json:"env"`
}

// EnvAsSlice возвращает Env в формате слайса строк вида "name=value".
func (o *ActionOptions) EnvAsSlice() []string {
	result := make([]string, 0, len(o.Env))
	for name, value := range o.Env {
		result = append(result, name+"="+value)
	}
	return result
}

// Config конфигурация для запуска runtime
type Config struct {
	Logger util.LoggingConfig

	ActionPath       string
	Replicas         int
	Port             int
	ServiceSock      string
	InRaw            string
	OutRaw           string
	ActionOptionsRaw string

	In            []string
	Out           []string
	ActionOptions *ActionOptions
}

// Parse загружает данные конфига.
func (c *Config) Parse() error {
	if c.InRaw != "" {
		c.In = strings.Split(c.InRaw, ",")
	}
	if c.OutRaw != "" {
		c.Out = strings.Split(strings.TrimSpace(c.OutRaw), ",")
	}
	c.ActionOptions = &ActionOptions{}
	if err := json.Unmarshal([]byte(c.ActionOptionsRaw), c.ActionOptions); err != nil {
		return fmt.Errorf("can not parse action options: %w", err)
	}
	return nil
}
