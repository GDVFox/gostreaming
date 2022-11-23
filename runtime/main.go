package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/GDVFox/gostreaming/runtime/config"
	"github.com/GDVFox/gostreaming/runtime/external"
	upstreambackup "github.com/GDVFox/gostreaming/runtime/upstream_backup"
	"github.com/GDVFox/gostreaming/util"
	"golang.org/x/sync/errgroup"
)

func init() {
	flag.StringVar(&config.Conf.Name, "name", "", "Name of action")
	flag.StringVar(&config.Conf.ActionPath, "action", "", "Path to action")
	flag.IntVar(&config.Conf.Replicas, "replicas", 1, "Number of replicas")
	flag.IntVar(&config.Conf.Port, "port", 0, "Port of action")
	flag.StringVar(&config.Conf.ServiceSock, "service-sock", "", "UDP socket for runtime-machine IPC")
	flag.StringVar(&config.Conf.Logger.Logfile, "log-file", "runtime.log", "File for logging")
	flag.StringVar(&config.Conf.Logger.Level, "log-level", "info", "Level for logging, default is info")
	flag.StringVar(&config.Conf.InRaw, "in", "", "Input addresses")
	flag.StringVar(&config.Conf.OutRaw, "out", "", "Output addresses")
	flag.StringVar(&config.Conf.ActionOptionsRaw, "action-opt", "", "Action args and env variables in JSON format")
}

func main() {
	var err error

	flag.Parse()
	if err := config.Conf.Parse(); err != nil {
		fmt.Fprintf(os.Stderr, "can not parse config arguments: %v\n", err)
		os.Exit(1)
	}

	inConfig := external.NewTCPConnectionConfig()
	inConfig.NoDelay = false

	outConfig := external.NewTCPConnectionConfig()
	outConfig.NoDelay = false

	// всегда чистим файлы в runtime.
	config.Conf.Logger.TruncateFile = true
	logger, err := util.NewLogger(&config.Conf.Logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not init logger: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	go func() {
		sig := <-signalChannel
		logger.Infof("got signal: %s, stopping...", sig)
		cancel()
	}()

	receiverConfig := &upstreambackup.DefaultReceiverConfig{
		UpstreamConfig: &upstreambackup.UpstreamReceiverConfig{
			AckBufferSize: 100,
			TCPConfig:     inConfig,
		},
	}
	receiver := upstreambackup.NewDefaultReceiver(":"+strconv.Itoa(config.Conf.Port), config.Conf.In, receiverConfig, logger)

	forwarderConfig := &upstreambackup.DefaultForwarderConfig{
		ACKPeriod:     5 * time.Second,
		ForwardLogDir: "/tmp/gostreaming-logs",
		DownstreamConfig: &upstreambackup.DownstreamForwarderConfig{
			MessagesBufferSize: 100,
			TCPConfig:          outConfig,
		},
	}
	forwarder, err := upstreambackup.NewDefaultForwarder(config.Conf.Name, config.Conf.Out, forwarderConfig, logger)
	if err != nil {
		logger.Errorf("can not init forwarder: %v", err)
		fmt.Fprintf(os.Stderr, "can not init forwarder: %v\n", err)
		os.Exit(1)
	}

	isSource := len(config.Conf.In) == 0
	runtime := NewRuntime(config.Conf.ActionPath, isSource, receiver, forwarder, config.Conf.ActionOptions, logger)
	serviceServer := NewServiceServer(config.Conf.ServiceSock, runtime, logger)

	wg, runCtx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		defer cancel() // завершение runtime должно завершать все.
		return runtime.Run(runCtx)
	})
	wg.Go(func() error {
		return serviceServer.Run(runCtx)
	})

	logger.Infof("runtime started for action: %s", config.Conf.ActionPath)
	if err := wg.Wait(); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		logger.Errorf("action run error: %v", err)
		fmt.Fprintf(os.Stderr, "action run error: %v\n", err)
		os.Exit(1)
	}
	logger.Infof("runtime stopped for action: %s", config.Conf.ActionPath)
}
