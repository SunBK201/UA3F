package main

import (
	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server"
)

const version = "1.6.0"

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

	if err := srv.Start(); err != nil {
		logrus.Fatal(err)
	}
}
