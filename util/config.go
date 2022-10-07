package util

import (
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadConfig загружает конфиг в формате yaml из файла filename.
func LoadConfig(filename string, cfg interface{}) error {
	cfgFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(cfgFile, cfg)
}

// Duration алиас для time.Duration, позволяющий
// использовать time.Duration в yaml конфиге.
type Duration time.Duration

// UnmarshalYAML загружает из конфига time.Duration
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	durationStr := ""
	if err := unmarshal(&durationStr); err != nil {
		return err
	}

	t, err := time.ParseDuration(durationStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse '%s' to time.Duration", durationStr)
	}

	*d = Duration(t)
	return nil
}
