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
	flag.StringVar(&config.Conf.InRaw, "in", "", "Input addresses")
	flag.StringVar(&config.Conf.OutRaw, "out", "", "Output addresses")
}

func main() {
	flag.Parse()
	if err := config.Conf.Parse(); err != nil {
		fmt.Printf("can not parse config arguments: %v\n", err)
		return
	}

	if err := logs.InitLogger(); err != nil {
		fmt.Printf("can not init logger: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	go func() {
		<-signalChannel
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

	if err := wg.Wait(); err != nil {
		fmt.Printf("action run error: %v\n", err)
		return
	}
}
