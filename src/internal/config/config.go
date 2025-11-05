package config

import (
	"flag"
	"fmt"
	"strings"
)

const (
	ServerModeHTTP     = "HTTP"
	ServerModeSocks5   = "SOCKS5"
	ServerModeTProxy   = "TPROXY"
	ServerModeRedirect = "REDIRECT"
)

type Config struct {
	ServerMode           string
	BindAddr             string
	Port                 int
	ListenAddr           string
	LogLevel             string
	PayloadUA            string
	UARegex              string
	EnablePartialReplace bool
}

func Parse() (*Config, bool) {
	var (
		serverMode string
		bindAddr   string
		port       int
		loglevel   string
		payloadUA  string
		uaRegx     string
		partial    bool
		showVer    bool
	)

	flag.StringVar(&serverMode, "m", ServerModeSocks5, "server mode: HTTP, SOCKS5, TPROXY, REDIRECT (default: SOCKS5)")
	flag.StringVar(&bindAddr, "b", "127.0.0.1", "bind address (default: 127.0.0.1)")
	flag.IntVar(&port, "p", 1080, "port")
	flag.StringVar(&payloadUA, "f", "FFF", "User-Agent")
	flag.StringVar(&uaRegx, "r", "", "UA-Pattern")
	flag.BoolVar(&partial, "s", false, "Enable Regex Partial Replace")
	flag.StringVar(&loglevel, "l", "info", "Log level (default: info)")
	flag.BoolVar(&showVer, "v", false, "show version")
	flag.Parse()

	cfg := &Config{
		ServerMode:           strings.ToUpper(serverMode),
		BindAddr:             bindAddr,
		Port:                 port,
		ListenAddr:           fmt.Sprintf("%s:%d", bindAddr, port),
		LogLevel:             loglevel,
		PayloadUA:            payloadUA,
		UARegex:              uaRegx,
		EnablePartialReplace: partial,
	}
	if serverMode == ServerModeRedirect {
		cfg.BindAddr = "0.0.0.0"
		cfg.ListenAddr = fmt.Sprintf("0.0.0.0:%d", port)
	}
	return cfg, showVer
}
