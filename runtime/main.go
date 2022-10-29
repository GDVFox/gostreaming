package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/GDVFox/gostreaming/runtime/config"
	"github.com/GDVFox/gostreaming/runtime/external"
	"github.com/GDVFox/gostreaming/runtime/logs"
	"golang.org/x/sync/errgroup"
)

func init() {
	flag.StringVar(&config.Conf.ActionPath, "action", "", "Path to action")
	flag.IntVar(&config.Conf.Replicas, "replicas", 1, "Number of replicas")
	flag.IntVar(&config.Conf.Port, "port", 0, "Port of action")
	flag.StringVar(&config.Conf.ServiceSock, "service-sock", "", "UDP socket for runtime-machine IPC")
	flag.StringVar(&config.Conf.Logger.Logfile, "log-file", "runtime.log", "File for logging")
	flag.StringVar(&config.Conf.Logger.Level, "log-level", "info", "Level for logging, default is info")
	flag.StringVar(&config.Conf.InRaw, "in", "", "Input addresses")
	flag.StringVar(&config.Conf.OutRaw, "out", "", "Output addresses")
}

func main() {
	flag.Parse()
	if err := config.Conf.Parse(); err != nil {
		logs.Logger.Errorf("can not parse config arguments: %v", err)
		fmt.Fprintf(os.Stderr, "can not parse config arguments: %v\n", err)
		os.Exit(1)
	}

	if err := logs.InitLogger(&config.Conf.Logger); err != nil {
		logs.Logger.Errorf("can not init logger: %v", err)
		fmt.Fprintf(os.Stderr, "can not init logger: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	go func() {
		sig := <-signalChannel
		logs.Logger.Infof("got signal: %s, stopping...", sig)
		cancel()
	}()

	serverCfg := external.NewTCPConnectionConfig()
	server := external.NewTCPServer(":"+strconv.Itoa(config.Conf.Port), serverCfg)

	outs := make([]*external.TCPClient, len(config.Conf.Out))
	for i, outAddr := range config.Conf.Out {
		cfg := external.NewTCPConnectionConfig()
		cfg.NoDelay = false

		outs[i] = external.NewTCPClient(outAddr, cfg)
	}

	action := NewAction(config.Conf.ActionPath, server, outs)
	serviceServer := NewServiceServer(config.Conf.ServiceSock, action)

	wg, runCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		return server.Run(runCtx)
	})
	for i := range outs {
		out := outs[i]
		wg.Go(func() error {
			return out.Run(runCtx)
		})
	}
	wg.Go(func() error {
		return action.Run(runCtx)
	})
	wg.Go(func() error {
		return serviceServer.Run(runCtx)
	})

	logs.Logger.Infof("runtime started for action: %s", config.Conf.ActionPath)
	if err := wg.Wait(); err != nil {
		logs.Logger.Errorf("action run error: %v", err)
		fmt.Fprintf(os.Stderr, "action run error: %v\n", err)
		os.Exit(1)
	}
}
