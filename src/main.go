package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/server/netlink"
	"github.com/sunbk201/ua3f/internal/statistics"
)

const version = "1.8.3"

func main() {
	cfg, showVer := config.Parse()

	log.SetLogConf(cfg.LogLevel)

	if showVer {
		fmt.Printf("UA3F version %s\n", version)
		return
	}

	log.LogHeader(version, cfg)

	rw, err := rewrite.New(cfg)
	if err != nil {
		slog.Error("rewrite.New", slog.Any("error", err))
		return
	}

	srv, err := server.NewServer(cfg, rw)
	if err != nil {
		slog.Error("server.NewServer", slog.Any("error", err))
		return
	}

	helper := netlink.New(cfg)
	if err := helper.Start(); err != nil {
		slog.Error("helper.Start", slog.Any("error", err))
		if err := srv.Close(); err != nil {
			slog.Error("srv.Close", slog.Any("error", err))
		}
		return
	}

	cleanup := make(chan os.Signal, 1)
	signal.Notify(cleanup, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			if err := helper.Close(); err != nil {
				slog.Error("helper.Close", slog.Any("error", err))
			}
			if err := srv.Close(); err != nil {
				slog.Error("srv.Close", slog.Any("error", err))
			}
			slog.Info("UA3F exited gracefully")
		})
	}

	go statistics.StartRecorder()

	go func() {
		<-cleanup
		shutdown()
		os.Exit(0)
	}()

	defer shutdown()

	if err := srv.Start(); err != nil {
		slog.Error("srv.Start", slog.Any("error", err))
		return
	}
}
