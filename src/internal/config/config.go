package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
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
	ServerMode          ServerMode
	BindAddr            string
	Port                int
	ListenAddr          string
	LogLevel            string
	RewriteMode         RewriteMode
	Rules               string
	PayloadUA           string
	UARegex             string
	PartialReplace      bool
	SetTTL              bool
	SetIPID             bool
	DelTCPTimestamp     bool
	SetTCPInitialWindow bool
	TCPDesync           TCPDesyncConfig
}

type TCPDesyncConfig struct {
	Enabled bool
	Bytes   uint32
	Packets uint32
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

	if os.Getenv("UA3F_SERVER_MODE") != "" {
		cfg.ServerMode = ServerMode(strings.ToUpper(os.Getenv("UA3F_SERVER_MODE")))
	}

	if os.Getenv("UA3F_PORT") != "" {
		var p int
		_, err := fmt.Sscanf(os.Getenv("UA3F_PORT"), "%d", &p)
		if err == nil {
			cfg.Port = p
			cfg.ListenAddr = fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)
		}
	}

	if os.Getenv("UA3F_REWRITE_MODE") != "" {
		cfg.RewriteMode = RewriteMode(strings.ToUpper(os.Getenv("UA3F_REWRITE_MODE")))
	}

	if os.Getenv("UA3F_PAYLOAD_UA") != "" {
		cfg.PayloadUA = os.Getenv("UA3F_PAYLOAD_UA")
	}

	if os.Getenv("UA3F_UA_REGEX") != "" {
		cfg.UARegex = os.Getenv("UA3F_UA_REGEX")
	}

	if os.Getenv("UA3F_PARTIAL_REPLACE") == "1" {
		cfg.PartialReplace = true
	}

	if os.Getenv("UA3F_TCPTS") == "1" {
		cfg.DelTCPTimestamp = true
	}
	if os.Getenv("UA3F_TTL") == "1" {
		cfg.SetTTL = true
	}
	if os.Getenv("UA3F_IPID") == "1" {
		cfg.SetIPID = true
	}
	if os.Getenv("UA3F_TCP_INIT_WINDOW") == "1" {
		cfg.SetTCPInitialWindow = true
	}

	if os.Getenv("UA3F_DESYNC") == "1" {
		cfg.TCPDesync.Enabled = true
		if val := os.Getenv("UA3F_DESYNC_BYTES"); val != "" {
			var bytes uint32
			_, err := fmt.Sscanf(val, "%d", &bytes)
			if err == nil {
				cfg.TCPDesync.Bytes = bytes
			}
		}
		if val := os.Getenv("UA3F_DESYNC_PACKETS"); val != "" {
			var packets uint32
			_, err := fmt.Sscanf(val, "%d", &packets)
			if err == nil {
				cfg.TCPDesync.Packets = packets
			}
		}
	}

	// Parse other options from -o flag
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

func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("Log Level", c.LogLevel),
		slog.String("Server Mode", string(c.ServerMode)),
		slog.String("Listen Address", c.ListenAddr),
		slog.String("Rewrite Mode", string(c.RewriteMode)),
		slog.String("User-Agent", c.PayloadUA),
		slog.String("User-Agent Regex", c.UARegex),
		slog.Bool("Partial Replace", c.PartialReplace),
		slog.Bool("Set TTL", c.SetTTL),
		slog.Bool("Set IP ID", c.SetIPID),
		slog.Bool("Delete TCP Timestamp", c.DelTCPTimestamp),
	)
}
