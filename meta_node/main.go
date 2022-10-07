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
	"github.com/GDVFox/gostreaming/meta_node/api/common"
	"github.com/GDVFox/gostreaming/meta_node/api/schemas"
	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/meta_node/external"
	"github.com/GDVFox/gostreaming/util"
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

	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/actions", common.CreateHandler(actions.ListActions, logger)).Methods(http.MethodGet)
	r.HandleFunc("/actions/{action_name:[a-zA-z0-9]+}", common.CreateHandler(actions.GetAction, logger)).Methods(http.MethodGet)
	r.HandleFunc("/actions", common.CreateHandler(actions.CreateScheme, logger)).Methods(http.MethodPost)
	r.HandleFunc("/actions/{action_name:[a-zA-z0-9]+}", common.CreateHandler(actions.DeleteAction, logger)).Methods(http.MethodDelete)

	r.HandleFunc("/schemas", common.CreateHandler(schemas.ListSchemas, logger)).Methods(http.MethodGet)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}", common.CreateHandler(schemas.GetScheme, logger)).Methods(http.MethodGet)
	r.HandleFunc("/schemas", common.CreateHandler(schemas.CreateScheme, logger)).Methods(http.MethodPost)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}", common.CreateHandler(schemas.DeleteScheme, logger)).Methods(http.MethodDelete)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/run", nil).Methods(http.MethodPut)
	r.HandleFunc("/schemas/{scheme_name:[a-zA-z0-9]+}/stop", nil).Methods(http.MethodPut)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	stopChannel := make(chan struct{})
	go func() {
		defer close(stopChannel)
		sig := <-signalChannel
		logger.Info("got signal: ", sig)
	}()

	startServer(r, config.Conf.HTTP, logger, stopChannel)
}

func startServer(h http.Handler, cfg *util.HTTPConfig, logger *util.Logger, stopChannel <-chan struct{}) {
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
