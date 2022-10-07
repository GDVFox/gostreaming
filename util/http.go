package util

import "strconv"

// HTTPConfig настройки для работы http сервера.
type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// NewtHTTPConfig создает HTTPConfig с настройками по-умолчанию.
func NewtHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Host: "127.0.0.1",
		Port: 8080, 
	}
}

// GetAddr возвращает адрес в виде host:port
func (c *HTTPConfig) GetAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}
