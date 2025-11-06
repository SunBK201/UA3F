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
	ServerMode     string
	BindAddr       string
	Port           int
	ListenAddr     string
	LogLevel       string
	PayloadUA      string
	UARegex        string
	PartialReplace bool
	DirectForward  bool
}

func Parse() (*Config, bool) {
	var (
		serverMode    string
		bindAddr      string
		port          int
		loglevel      string
		payloadUA     string
		uaRegx        string
		partial       bool
		directForward bool
		showVer       bool
	)

	flag.StringVar(&serverMode, "m", ServerModeSocks5, "Server mode: HTTP, SOCKS5, TPROXY, REDIRECT")
	flag.StringVar(&bindAddr, "b", "127.0.0.1", "Bind address")
	flag.IntVar(&port, "p", 1080, "Port")
	flag.StringVar(&loglevel, "l", "info", "Log level")
	flag.StringVar(&payloadUA, "f", "FFF", "User-Agent")
	flag.StringVar(&uaRegx, "r", "", "User-Agent regex")
	flag.BoolVar(&partial, "s", false, "Enable regex partial replace")
	flag.BoolVar(&directForward, "d", false, "Pure Forwarding (no User-Agent rewriting)")
	flag.BoolVar(&showVer, "v", false, "Show version")
	flag.Parse()

	cfg := &Config{
		ServerMode:     strings.ToUpper(serverMode),
		BindAddr:       bindAddr,
		Port:           port,
		ListenAddr:     fmt.Sprintf("%s:%d", bindAddr, port),
		LogLevel:       loglevel,
		PayloadUA:      payloadUA,
		UARegex:        uaRegx,
		DirectForward:  directForward,
		PartialReplace: partial,
	}
	if serverMode == ServerModeRedirect {
		cfg.BindAddr = "0.0.0.0"
		cfg.ListenAddr = fmt.Sprintf("0.0.0.0:%d", port)
	}
	return cfg, showVer
}
