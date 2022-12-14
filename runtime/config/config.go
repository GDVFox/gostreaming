package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GDVFox/gostreaming/util"
)

// Conf синглтон конфигурации
var Conf = &Config{}

// ActionOptions опции для запуска действия
type ActionOptions struct {
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	ConnWhitelist []string          `json:"conn_whitelist"`
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

	Name             string
	ActionPath       string
	Replicas         int
	Port             int
	ServiceSock      string
	InRaw            string
	OutRaw           string
	ActionOptionsRaw string

	ACKPeriodRaw  string
	ForwardLogDir string

	In            []string
	Out           []string
	ActionOptions *ActionOptions

	ACKPeriod time.Duration
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

	dur, err := time.ParseDuration(c.ACKPeriodRaw)
	if err != nil {
		return fmt.Errorf("can not parse ack period: %w", err)
	}
	c.ACKPeriod = dur

	return nil
}
