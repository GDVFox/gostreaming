package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"

	"github.com/GDVFox/gostreaming/meta_node/api/actions"
	"github.com/GDVFox/gostreaming/meta_node/api/schemas"
	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/meta_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "", "Name of config file")
}

func main() {
	flag.Parse()
	if err := util.LoadConfig(configFile, config.Conf); err != nil {
		fmt.Printf("can not read config file: %v", err)
		return
	}

	logger, err := util.NewLogger(config.Conf.Logging)
	if err != nil {
		fmt.Printf("can not init logger: %v", err)
		return
	}

	if err := external.InitExternal(config.Conf); err != nil {
		logger.Fatalf("can not init external resources: %v", err)
		return
	}

	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	if err := watcher.StartWatcher(watcherCtx, logger, config.Conf.Watcher); err != nil {
		logger.Fatalf("can not start watcher: %v", err)
		return
	}

	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/actions", httplib.CreateHandler(actions.ListActions, logger)).Methods(http.MethodGet)
	r.HandleFunc("/actions/{action_name:[a-zA-z0-9\\-]+}", httplib.CreateHandler(actions.GetAction, logger)).Methods(http.MethodGet)
	r.HandleFunc("/actions", httplib.CreateHandler(actions.CreateScheme, logger)).Methods(http.MethodPost)
	r.HandleFunc("/actions/{action_name:[a-zA-z0-9\\-]+}", httplib.CreateHandler(actions.DeleteAction, logger)).Methods(http.MethodDelete)

	r.HandleFunc("/schemas", httplib.CreateHandler(schemas.ListSchemas, logger)).Methods(http.MethodGet)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}", httplib.CreateHandler(schemas.GetScheme, logger)).Methods(http.MethodGet)
	r.HandleFunc("/schemas", httplib.CreateHandler(schemas.CreateScheme, logger)).Methods(http.MethodPost)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}", httplib.CreateHandler(schemas.DeleteScheme, logger)).Methods(http.MethodDelete)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/run", httplib.CreateHandler(schemas.RunScheme, logger)).Methods(http.MethodPut)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/stop", httplib.CreateHandler(schemas.StopScheme, logger)).Methods(http.MethodPut)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/dashboard", httplib.CreateHandler(schemas.GetDashboard, logger)).Methods(http.MethodGet)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/send_dashboard", httplib.CreateWSHandler(schemas.SendDashboard, logger)).Methods(http.MethodGet)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	stopChannel := make(chan struct{})
	go func() {
		defer close(stopChannel)
		defer watcherCancel()
		sig := <-signalChannel
		logger.Info("got signal: ", sig)
	}()

	httplib.StartServer(r, config.Conf.HTTP, logger, stopChannel)
}
