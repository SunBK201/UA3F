package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.yaml.in/yaml/v3"
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
	RewriteModeRule   RewriteMode = "RULE"
)

type Config struct {
	ServerMode  ServerMode `yaml:"server-mode" validate:"required,oneof=HTTP SOCKS5 TPROXY REDIRECT NFQUEUE"`
	BindAddress string     `yaml:"bind-address" validate:"ip"`
	Port        int        `yaml:"port" default:"1080" validate:"required,min=1,max=65535"`

	LogLevel string `yaml:"log-level" default:"info" validate:"required,oneof=debug info warn error"`

	RewriteMode RewriteMode `yaml:"rewrite-mode" default:"GLOBAL" validate:"required,oneof=GLOBAL DIRECT RULE"`

	UserAgent               string `yaml:"user-agent" default:"FFF"`
	UserAgentRegex          string `yaml:"user-agent-regex"`
	UserAgentPartialReplace bool   `yaml:"user-agent-partial-replace"`

	TTL              bool `yaml:"ttl"`
	IPID             bool `yaml:"ipid"`
	TCPTimeStamp     bool `yaml:"tcp_timestamp"`
	TCPInitialWindow bool `yaml:"tcp_initial_window"`

	Desync DesyncConfig `yaml:"desync"`

	Rules     []Rule `yaml:"rules" validate:"dive"`
	RulesJson string `yaml:"rules_json,omitempty"`
}

type DesyncConfig struct {
	Reorder        bool   `yaml:"reorder"`
	ReorderBytes   uint32 `yaml:"reorder-bytes" default:"8" validate:"min=0"`
	ReorderPackets uint32 `yaml:"reorder-packets" default:"1500" validate:"min=0"`
	Inject         bool   `yaml:"inject"`
	InjectTTL      uint8  `yaml:"inject-ttl" default:"3" validate:"min=0"`
}

type Rule struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	Type string `json:"type" yaml:"type" validate:"required,oneof=HEADER-KEYWORD HEADER-REGEX DEST-PORT IP-CIDR SRC-IP DOMAIN-SUFFIX DOMAIN-KEYWORD DOMAIN URL-REGEX FINAL"`

	MatchHeader string `json:"match_header,omitempty" yaml:"match-header,omitempty" validate:"required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX"`
	MatchValue  string `json:"match_value,omitempty" yaml:"match-value,omitempty" validate:"required_if=Type DEST-PORT,required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX,required_if=Type IP-CIDR,required_if=Type SRC-IP,required_if=Type DOMAIN-SUFFIX,required_if=Type DOMAIN-KEYWORD,required_if=Type DOMAIN,required_if=Type URL-REGEX"`

	Action string `json:"action" yaml:"action" validate:"required,oneof=DIRECT REPLACE REPLACE-REGEX DELETE DROP ADD"`

	RewriteHeader string `json:"rewrite_header,omitempty" yaml:"rewrite-header,omitempty" validate:"required_if=Action REPLACE,required_if=Action REPLACE-REGEX,required_if=Action DELETE,required_if=Action ADD"`
	RewriteValue  string `json:"rewrite_value,omitempty" yaml:"rewrite-value,omitempty" validate:"required_if=Action REPLACE,required_if=Action REPLACE-REGEX,required_if=Action ADD"`

	RewriteRegex string `json:"rewrite_regex,omitempty" yaml:"rewrite-regex,omitempty" validate:"required_if=Action REPLACE-REGEX"`

	Continue bool `json:"continue,omitempty" yaml:"continue,omitempty"`
}

func Parse() (*Config, bool, error) {
	var (
		configFile       string
		serverMode       string
		bindAddr         string
		port             int
		loglevel         string
		payloadUA        string
		uaRegx           string
		partial          bool
		rewriteMode      string
		rulesJson        string
		showVer          bool
		genConfig        bool
		ttl              bool
		ipid             bool
		tcpTimestamp     bool
		tcpInitialWindow bool
		desyncReorder    bool
		reorderBytes     uint
		reorderPackets   uint
		desyncInject     bool
		injectTTL        uint
	)

	flag.StringVar(&configFile, "c", "", "Config file path")
	flag.StringVar(&serverMode, "m", "", "Server mode: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE")
	flag.StringVar(&bindAddr, "b", "", "Bind address")
	flag.IntVar(&port, "p", 0, "Port")
	flag.StringVar(&loglevel, "l", "", "Log level")
	flag.StringVar(&payloadUA, "f", "", "User-Agent")
	flag.StringVar(&uaRegx, "r", "", "User-Agent regex")
	flag.BoolVar(&partial, "s", false, "Enable regex partial replace")
	flag.StringVar(&rewriteMode, "x", "", "Rewrite mode: GLOBAL, DIRECT, RULE")
	flag.StringVar(&rulesJson, "z", "", "Rules JSON string")
	flag.BoolVar(&showVer, "v", false, "Show version")
	flag.BoolVar(&genConfig, "g", false, "Generate template config file")
	flag.BoolVar(&ttl, "ttl", false, "Set TTL")
	flag.BoolVar(&ipid, "ipid", false, "Set IP ID")
	flag.BoolVar(&tcpTimestamp, "tcpts", false, "Delete TCP Timestamp")
	flag.BoolVar(&tcpInitialWindow, "tcpwin", false, "Set TCP Initial Window")
	flag.BoolVar(&desyncReorder, "desync-reorder", false, "Enable desync reorder")
	flag.UintVar(&reorderBytes, "desync-reorder-bytes", 0, "Desync reorder bytes")
	flag.UintVar(&reorderPackets, "desync-reorder-packets", 0, "Desync reorder packets")
	flag.BoolVar(&desyncInject, "desync-inject", false, "Enable desync inject")
	flag.UintVar(&injectTTL, "desync-inject-ttl", 0, "Desync inject TTL")
	flag.Parse()

	if genConfig {
		_, err := GenerateTemplateConfig(true)
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate template config: %w", err)
		}
		fmt.Println("Template config file 'config.yaml' generated successfully.")
		return nil, false, nil
	}

	if showVer {
		return nil, true, nil
	}

	// Track which CLI flags were explicitly set
	cliSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		cliSet[f.Name] = true
	})

	// 1. Start with default values (lowest priority)
	cfg := &Config{
		ServerMode:  ServerModeSocks5,
		BindAddress: "127.0.0.1",
		Port:        1080,
		LogLevel:    "info",
		UserAgent:   "FFF",
		RewriteMode: RewriteModeGlobal,
		Desync: DesyncConfig{
			ReorderBytes:   8,
			ReorderPackets: 1500,
			InjectTTL:      3,
		},
	}

	// 2. Apply config file values (if provided)
	if configFile != "" {
		fileCfg, err := LoadConfig(configFile)
		if err != nil {
			return nil, false, err
		}
		cfg = fileCfg
	}

	// 3. Apply environment variables (overrides config file)
	applyEnvConfig(cfg)

	// 4. Apply CLI arguments (highest priority, only if explicitly set)
	if cliSet["m"] {
		cfg.ServerMode = ServerMode(strings.ToUpper(serverMode))
	}
	if cliSet["b"] {
		cfg.BindAddress = bindAddr
	}
	if cliSet["p"] {
		cfg.Port = port
	}
	if cliSet["l"] {
		cfg.LogLevel = loglevel
	}
	if cliSet["f"] {
		cfg.UserAgent = payloadUA
	}
	if cliSet["r"] {
		cfg.UserAgentRegex = uaRegx
	}
	if cliSet["s"] {
		cfg.UserAgentPartialReplace = partial
	}
	if cliSet["x"] {
		cfg.RewriteMode = RewriteMode(strings.ToUpper(rewriteMode))
	}
	if cliSet["z"] {
		cfg.RulesJson = rulesJson
	}
	if cliSet["ttl"] {
		cfg.TTL = ttl
	}
	if cliSet["ipid"] {
		cfg.IPID = ipid
	}
	if cliSet["tcpts"] {
		cfg.TCPTimeStamp = tcpTimestamp
	}
	if cliSet["tcpwin"] {
		cfg.TCPInitialWindow = tcpInitialWindow
	}
	if cliSet["desync-reorder"] {
		cfg.Desync.Reorder = desyncReorder
	}
	if cliSet["desync-reorder-bytes"] {
		cfg.Desync.ReorderBytes = uint32(reorderBytes)
	}
	if cliSet["desync-reorder-packets"] {
		cfg.Desync.ReorderPackets = uint32(reorderPackets)
	}
	if cliSet["desync-inject"] {
		cfg.Desync.Inject = desyncInject
	}
	if cliSet["desync-inject-ttl"] {
		cfg.Desync.InjectTTL = uint8(injectTTL)
	}

	// Backwards compatibility: convert deprecated "RULES" value to "RULE".
	if cfg.RewriteMode == "RULES" {
		cfg.RewriteMode = RewriteModeRule
	}

	return cfg, showVer, nil
}

// applyEnvConfig applies environment variables to the config
func applyEnvConfig(cfg *Config) {
	if os.Getenv("UA3F_SERVER_MODE") != "" {
		cfg.ServerMode = ServerMode(strings.ToUpper(os.Getenv("UA3F_SERVER_MODE")))
	}

	if os.Getenv("UA3F_BIND_ADDRESS") != "" {
		cfg.BindAddress = os.Getenv("UA3F_BIND_ADDRESS")
	}

	if os.Getenv("UA3F_PORT") != "" {
		var p int
		_, err := fmt.Sscanf(os.Getenv("UA3F_PORT"), "%d", &p)
		if err == nil {
			cfg.Port = p
		}
	}

	if os.Getenv("UA3F_LOG_LEVEL") != "" {
		cfg.LogLevel = strings.ToLower(os.Getenv("UA3F_LOG_LEVEL"))
	}

	if os.Getenv("UA3F_REWRITE_MODE") != "" {
		cfg.RewriteMode = RewriteMode(strings.ToUpper(os.Getenv("UA3F_REWRITE_MODE")))
	}

	if os.Getenv("UA3F_PAYLOAD_UA") != "" {
		cfg.UserAgent = os.Getenv("UA3F_PAYLOAD_UA")
	}

	if os.Getenv("UA3F_UA_REGEX") != "" {
		cfg.UserAgentRegex = os.Getenv("UA3F_UA_REGEX")
	}

	if os.Getenv("UA3F_PARTIAL_REPLACE") == "1" {
		cfg.UserAgentPartialReplace = true
	}

	if os.Getenv("UA3F_TCPTS") == "1" {
		cfg.TCPTimeStamp = true
	}
	if os.Getenv("UA3F_TTL") == "1" {
		cfg.TTL = true
	}
	if os.Getenv("UA3F_IPID") == "1" {
		cfg.IPID = true
	}
	if os.Getenv("UA3F_TCP_INIT_WINDOW") == "1" {
		cfg.TCPInitialWindow = true
	}

	if os.Getenv("UA3F_DESYNC_REORDER") == "1" {
		cfg.Desync.Reorder = true
	}
	if val := os.Getenv("UA3F_DESYNC_REORDER_BYTES"); val != "" {
		var bytes uint32
		_, err := fmt.Sscanf(val, "%d", &bytes)
		if err == nil {
			cfg.Desync.ReorderBytes = bytes
		}
	}
	if val := os.Getenv("UA3F_DESYNC_REORDER_PACKETS"); val != "" {
		var packets uint32
		_, err := fmt.Sscanf(val, "%d", &packets)
		if err == nil {
			cfg.Desync.ReorderPackets = packets
		}
	}

	if os.Getenv("UA3F_DESYNC_INJECT") == "1" {
		cfg.Desync.Inject = true
	}
	if val := os.Getenv("UA3F_DESYNC_INJECT_TTL"); val != "" {
		var ttl uint8
		_, err := fmt.Sscanf(val, "%d", &ttl)
		if err == nil {
			cfg.Desync.InjectTTL = ttl
		}
	}

	if os.Getenv("UA3F_RULES_JSON") != "" {
		cfg.RulesJson = os.Getenv("UA3F_RULES_JSON")
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("Log Level", c.LogLevel),
		slog.String("Server Mode", string(c.ServerMode)),
		slog.String("Bind Address", c.BindAddress),
		slog.String("Rewrite Mode", string(c.RewriteMode)),
		slog.String("User-Agent", c.UserAgent),
		slog.String("User-Agent Regex", c.UserAgentRegex),
		slog.Bool("User-Agent Partial Replace", c.UserAgentPartialReplace),
		slog.Bool("Set TTL", c.TTL),
		slog.Bool("Set IP ID", c.IPID),
		slog.Bool("Delete TCP Timestamp", c.TCPTimeStamp),
		slog.Bool("Set TCP Initial Window", c.TCPInitialWindow),
		slog.Attr{
			Key: "Desync", Value: slog.GroupValue(
				slog.Bool("Reorder", c.Desync.Reorder),
				slog.Uint64("Reorder Bytes", uint64(c.Desync.ReorderBytes)),
				slog.Uint64("Reorder Packets", uint64(c.Desync.ReorderPackets)),
				slog.Bool("Inject", c.Desync.Inject),
				slog.Uint64("Inject TTL", uint64(c.Desync.InjectTTL)),
			),
		},
	)
}
