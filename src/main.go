package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/daemon"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/server/desync"
	"github.com/sunbk201/ua3f/internal/server/netlink"
)

var (
	appVersion    = "Development"
	shutdownChain []func() error
)

func main() {
	cfg, showVer := config.Parse()

	log.SetLogConf(cfg.LogLevel)

	if showVer {
		fmt.Printf("UA3F version %s\n", appVersion)
		return
	}

	log.LogHeader(appVersion, cfg)

	if err := daemon.SetUserGroup(cfg); err != nil {
		slog.Error("daemon.SetUserGroup", slog.Any("error", err))
		return
	}
	if err := daemon.SetOOMScoreAdj(-800); err != nil {
		slog.Warn("daemon.SetOOMScoreAdj", slog.Any("error", err))
	}

	helper := netlink.New(cfg)
	addShutdown("helper.Close", helper.Close)
	if err := helper.Start(); err != nil {
		slog.Error("helper.Start", slog.Any("error", err))
		shutdown()
		return
	}

	if cfg.TCPDesync.Enabled {
		desync := desync.New(cfg)
		addShutdown("desync.Close", desync.Close)
		if err := desync.Start(); err != nil {
			slog.Error("desync.Start", slog.Any("error", err))
			shutdown()
			return
		}
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		slog.Error("server.NewServer", slog.Any("error", err))
		shutdown()
		return
	}
	addShutdown("srv.Close", srv.Close)
	if err := srv.Start(); err != nil {
		slog.Error("srv.Start", slog.Any("error", err))
		shutdown()
		return
	}

	cleanup := make(chan os.Signal, 1)
	signal.Notify(cleanup, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	for {
		s := <-cleanup
		slog.Info("Received signal", slog.String("signal", s.String()))
		switch s {
		case syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM:
			shutdown()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}

func addShutdown(name string, fn func() error) {
	shutdownChain = append(shutdownChain, func() error {
		if err := fn(); err != nil {
			slog.Error(name, slog.Any("error", err))
			return err
		}
		return nil
	})
}

func shutdown() {
	for i := len(shutdownChain) - 1; i >= 0; i-- {
		_ = shutdownChain[i]()
	}
	slog.Info("UA3F exit")
}
