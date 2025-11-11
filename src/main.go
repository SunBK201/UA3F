package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/statistics"
)

const version = "1.7.0"

func main() {
	cfg, showVer := config.Parse()

	log.SetLogConf(cfg.LogLevel)

	if showVer {
		logrus.Infof("UA3F version: %s", version)
		return
	}

	rw, err := rewrite.New(cfg)
	if err != nil {
		logrus.Fatal(err)
	}

	srv, err := server.NewServer(cfg, rw)
	if err != nil {
		logrus.Fatal(err)
	}

	log.LogHeader(version, cfg)

	go statistics.StartRecorder()

	cleanup := make(chan os.Signal, 1)
	signal.Notify(cleanup, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-cleanup
		logrus.Info("Shutting down UA3F...")
		if err := srv.Close(); err != nil {
			logrus.Errorf("Error during UA3F close: %v", err)
		}
		logrus.Info("UA3F exited gracefully")
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		logrus.Fatal(err)
	}
}
