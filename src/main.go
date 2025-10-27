package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/socks5"
)

const version = "0.8.0"

func main() {
	cfg, showVer := config.Parse()
	if showVer {
		fmt.Println("UA3F v" + version)
		return
	}
	log.SetLogConf(cfg.LogLevel)

	rw, err := rewrite.New(cfg)
	if err != nil {
		logrus.Fatal(err)
	}

	srv := socks5.New(cfg, rw)
	log.LogHeader(version, srv.ListenAddr, cfg)
	if err := srv.Start(); err != nil {
		logrus.Fatal(err)
	}
}
