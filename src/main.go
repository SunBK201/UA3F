package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/server/netlink"
	"github.com/sunbk201/ua3f/internal/statistics"
	"github.com/sunbk201/ua3f/internal/usergroup"
)

var appVersion = "Development"

func main() {
	cfg, showVer := config.Parse()

	log.SetLogConf(cfg.LogLevel)

	if showVer {
		fmt.Printf("UA3F version %s\n", appVersion)
		return
	}

	log.LogHeader(appVersion, cfg)

	if err := usergroup.SetUserGroup(cfg); err != nil {
		slog.Error("usergroup.SetUserGroup", slog.Any("error", err))
		return
	}

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

	shutdown := func() {
		if err := helper.Close(); err != nil {
			slog.Error("helper.Close", slog.Any("error", err))
		}
		if err := srv.Close(); err != nil {
			slog.Error("srv.Close", slog.Any("error", err))
		}
		slog.Info("UA3F exit")
	}

	go statistics.StartRecorder()

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
