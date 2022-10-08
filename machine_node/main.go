package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GDVFox/gostreaming/machine_node/api"
	"github.com/GDVFox/gostreaming/machine_node/config"
	"github.com/GDVFox/gostreaming/machine_node/external"
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

	r.HandleFunc("/run", httplib.CreateHandler(api.RunAction, logger)).Methods(http.MethodPost)
	r.HandleFunc("/stop", nil).Methods(http.MethodPost)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	stopChannel := make(chan struct{})
	go func() {
		defer close(stopChannel)
		sig := <-signalChannel
		logger.Info("got signal: ", sig)
	}()

	httplib.StartServer(r, config.Conf.HTTP, logger, stopChannel)
}
