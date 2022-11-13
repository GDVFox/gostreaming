package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GDVFox/gostreaming/machine_node/api"
	"github.com/GDVFox/gostreaming/machine_node/config"
	"github.com/GDVFox/gostreaming/machine_node/external"
	"github.com/GDVFox/gostreaming/machine_node/watcher"
	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/httplib"
	"github.com/gorilla/mux"
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

	if _, err := os.Stat(config.Conf.RuntimeLogsDir); os.IsNotExist(err) {
		if err := os.Mkdir(config.Conf.RuntimeLogsDir, 0700); err != nil {
			fmt.Printf("can not create logging dir: %v", err)
			return
		}
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

	watcherContext, watcherCancel := context.WithCancel(context.Background())
	if err := watcher.StartWatcher(watcherContext, logger, config.Conf.Watcher); err != nil {
		logger.Fatalf("can not init external resources: %v", err)
		return
	}

	r := mux.NewRouter().PathPrefix("/v1").Subrouter()

	r.HandleFunc("/ping", httplib.CreateHandler(api.Ping, logger)).Methods(http.MethodGet)
	r.HandleFunc("/run", httplib.CreateHandler(api.RunAction, logger)).Methods(http.MethodPost)
	r.HandleFunc("/stop", httplib.CreateHandler(api.StopAction, logger)).Methods(http.MethodPost)
	r.HandleFunc("/change_out", httplib.CreateHandler(api.ChangeActionOut, logger)).Methods(http.MethodPost)

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
