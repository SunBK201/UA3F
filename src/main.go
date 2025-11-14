package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/server/netlink"
	"github.com/sunbk201/ua3f/internal/statistics"
)

const version = "1.8.2"

func main() {
	cfg, showVer := config.Parse()

	log.SetLogConf(cfg.LogLevel)

	if showVer {
		logrus.Infof("UA3F version: %s", version)
		return
	}

	log.LogHeader(version, cfg)

	rw, err := rewrite.New(cfg)
	if err != nil {
		logrus.Errorf("rewrite.New: %v", err)
		return
	}

	srv, err := server.NewServer(cfg, rw)
	if err != nil {
		logrus.Errorf("server.NewServer: %v", err)
		return
	}

	helper := netlink.New(cfg)
	if err := helper.Start(); err != nil {
		logrus.Errorf("helper.Start: %v", err)
		if err := srv.Close(); err != nil {
			logrus.Errorf("srv.Close: %v", err)
		}
		return
	}

	cleanup := make(chan os.Signal, 1)
	signal.Notify(cleanup, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)

	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			if err := helper.Close(); err != nil {
				logrus.Errorf("helper.Close: %v", err)
			}
			if err := srv.Close(); err != nil {
				logrus.Errorf("srv.Close: %v", err)
			}
			logrus.Info("UA3F exited gracefully")
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
		logrus.Errorf("srv.Start: %v", err)
		return
	}
}
