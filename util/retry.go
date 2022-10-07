package util

import (
	"context"
	"time"
)

// RetryConfig параметры ретраев.
type RetryConfig struct {
	Delay Duration `yaml:"delay"`
	Count int      `yaml:"count"`
}

// NewRetryConfig создает RetryConfig с настройками по-умолчанию.
func NewRetryConfig() *RetryConfig {
	return &RetryConfig{
		Delay: Duration(1 * time.Second),
		Count: 5,
	}
}

// Retry wrapper to retry f with ctx to cancel.
// Send conf.Count = 0 for infinite retry
func Retry(ctx context.Context, cfg *RetryConfig, f func() error) error {
	var err error
	var i int
	for {
		if err = f(); err == nil {
			return nil
		}
		if i++; i == cfg.Count {
			return err
		}
		select {
		case <-time.After(time.Duration(cfg.Delay)):
			continue
		case <-ctx.Done():
			return err
		}
	}
}
