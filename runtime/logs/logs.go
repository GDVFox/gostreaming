package logs

import "github.com/GDVFox/gostreaming/util"

// Logger синлтон-объект для логирования.
var Logger *util.Logger

// InitLogger инициализирует синлтон-объект для логирования.
func InitLogger(cfg *util.LoggingConfig) error {
	var err error
	cfg.TruncateFile = true // всегда чистим файлы в runtime.
	Logger, err = util.NewLogger(cfg)
	return err
}
