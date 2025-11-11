package config

import (
	"flag"
	"fmt"
	"strings"
)

type ServerMode string

const (
	ServerModeHTTP     ServerMode = "HTTP"
	ServerModeSocks5   ServerMode = "SOCKS5"
	ServerModeTProxy   ServerMode = "TPROXY"
	ServerModeRedirect ServerMode = "REDIRECT"
	ServerModeNFQueue  ServerMode = "NFQUEUE"
)

type RewriteMode string

const (
	RewriteModeGlobal RewriteMode = "GLOBAL"
	RewriteModeDirect RewriteMode = "DIRECT"
	RewriteModeRules  RewriteMode = "RULES"
)

type Config struct {
	ServerMode      ServerMode
	BindAddr        string
	Port            int
	ListenAddr      string
	LogLevel        string
	RewriteMode     RewriteMode
	Rules           string
	PayloadUA       string
	UARegex         string
	PartialReplace  bool
	SetTTL          bool
	SetIPID         bool
	DelTCPTimestamp bool
}

func Parse() (*Config, bool) {
	var (
		serverMode  string
		bindAddr    string
		port        int
		loglevel    string
		payloadUA   string
		uaRegx      string
		partial     bool
		rewriteMode string
		rules       string
		others      string
		showVer     bool
	)

	flag.StringVar(&serverMode, "m", string(ServerModeSocks5), "Server mode: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE")
	flag.StringVar(&bindAddr, "b", "127.0.0.1", "Bind address")
	flag.IntVar(&port, "p", 1080, "Port")
	flag.StringVar(&loglevel, "l", "info", "Log level")
	flag.StringVar(&payloadUA, "f", "FFF", "User-Agent")
	flag.StringVar(&uaRegx, "r", "", "User-Agent regex")
	flag.BoolVar(&partial, "s", false, "Enable regex partial replace")
	flag.StringVar(&rewriteMode, "x", string(RewriteModeGlobal), "Rewrite mode: GLOBAL, DIRECT, RULES")
	flag.StringVar(&rules, "z", "", "Rules JSON string")
	flag.StringVar(&others, "o", "", "Other options (tcpts, ttl, ipid)")
	flag.BoolVar(&showVer, "v", false, "Show version")
	flag.Parse()

	cfg := &Config{
		ServerMode:     ServerMode(strings.ToUpper(serverMode)),
		BindAddr:       bindAddr,
		Port:           port,
		ListenAddr:     fmt.Sprintf("%s:%d", bindAddr, port),
		LogLevel:       loglevel,
		PayloadUA:      payloadUA,
		UARegex:        uaRegx,
		PartialReplace: partial,
		RewriteMode:    RewriteMode(strings.ToUpper(rewriteMode)),
		Rules:          rules,
	}
	if cfg.ServerMode == ServerModeRedirect || cfg.ServerMode == ServerModeTProxy {
		cfg.BindAddr = "0.0.0.0"
		cfg.ListenAddr = fmt.Sprintf("0.0.0.0:%d", port)
	}

	// Parse other options
	opts := strings.Split(others, ",")
	for _, opt := range opts {
		switch strings.ToLower(strings.TrimSpace(opt)) {
		case "tcpts":
			cfg.DelTCPTimestamp = true
		case "ttl":
			cfg.SetTTL = true
		case "ipid":
			cfg.SetIPID = true
		}
	}

	return cfg, showVer
}
