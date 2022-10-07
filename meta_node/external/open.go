package external

import "github.com/GDVFox/gostreaming/meta_node/config"

// ETCD объект синглтон для работы с etcd.
var ETCD *ETCDClient

// InitExternal инициализирует подключения к внешним ресурсам.
func InitExternal(cfg *config.Config) error {
	var err error
	ETCD, err = NewETCDClient(cfg.ETCD)
	if err != nil {
		return err
	}
	return nil
}
