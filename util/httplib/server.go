package httplib

import (
	"context"
	"net/http"
	"strconv"

	"github.com/GDVFox/gostreaming/util"
)

// HTTPConfig настройки для работы встроенного http сервера.
type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// NewtHTTPConfig создает HTTPConfig с настройками по-умолчанию.
func NewtHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Host: "0.0.0.0",
		Port: 8080,
	}
}

// GetAddr возвращает адрес в виде host:port
func (c *HTTPConfig) GetAddr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// StartServer запускает http сервер с обработчиком h.
func StartServer(h http.Handler, cfg *HTTPConfig, logger *util.Logger, stopChannel <-chan struct{}) {
	srv := &http.Server{
		Addr:    cfg.GetAddr(),
		Handler: h,
	}

	errChannel := make(chan error)
	go func() {
		defer close(errChannel)
		errChannel <- srv.ListenAndServe()
	}()

	logger.Infof("started server at %s", cfg.GetAddr())
	select {
	case <-stopChannel:
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("failed to gracefully shut down server: ", err)
		}
	case err := <-errChannel:
		logger.Error(err)
	}
	logger.Info("server was stopped")
}
