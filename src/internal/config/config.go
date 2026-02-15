package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
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

	APIServer       string `yaml:"api-server"`
	APIServerSecret string `yaml:"api-server-secret"`

	LogLevel string `yaml:"log-level" default:"info" validate:"required,oneof=debug info warn error"`

	RewriteMode RewriteMode `yaml:"rewrite-mode" default:"GLOBAL" validate:"required,oneof=GLOBAL DIRECT RULE"`

	UserAgent               string `yaml:"user-agent" default:"FFF"`
	UserAgentRegex          string `yaml:"user-agent-regex"`
	UserAgentPartialReplace bool   `yaml:"user-agent-partial-replace"`

	TTL              bool `yaml:"ttl"`
	IPID             bool `yaml:"ipid"`
	TCPTimeStamp     bool `yaml:"tcp_timestamp"`
	TCPInitialWindow bool `yaml:"tcp_initial_window"`

	MitM MitMConfig `yaml:"mitm"`

	Desync DesyncConfig `yaml:"desync"`

	HeaderRules     []Rule `yaml:"header-rewrite" validate:"dive"`
	HeaderRulesJson string `yaml:"header-rewrite-json,omitempty"`

	BodyRules     []Rule `yaml:"body-rewrite" validate:"dive"`
	BodyRulesJson string `yaml:"body-rewrite-json,omitempty"`

	URLRedirectRules []Rule `yaml:"url-redirect" validate:"dive"`
	URLRedirectJson  string `yaml:"url-redirect-json,omitempty"`
}

type MitMConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Hostname           string `yaml:"hostname"`
	CAP12              string `yaml:"ca-p12"`
	CAP12Base64        string `yaml:"ca-p12-base64"`
	CAPassphrase       string `yaml:"ca-passphrase"`
	InsecureSkipVerify bool   `yaml:"insecure-skip-verify"`
}

type DesyncConfig struct {
	Reorder        bool   `yaml:"reorder"`
	ReorderBytes   uint32 `yaml:"reorder-bytes" default:"8" validate:"min=0"`
	ReorderPackets uint32 `yaml:"reorder-packets" default:"1500" validate:"min=0"`
	Inject         bool   `yaml:"inject"`
	InjectTTL      uint8  `yaml:"inject-ttl" default:"3" validate:"min=0"`

	DesyncPorts string `yaml:"desync-ports,omitempty"`
}

type Rule struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	Type string `json:"type" yaml:"type" validate:"required,oneof=HEADER-KEYWORD HEADER-REGEX DEST-PORT IP-CIDR SRC-IP DOMAIN-SUFFIX DOMAIN-KEYWORD DOMAIN DOMAIN-SET URL-REGEX FINAL"`

	MatchHeader string `json:"match_header,omitempty" yaml:"match-header,omitempty" validate:"required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX"`
	MatchValue  string `json:"match_value,omitempty" yaml:"match-value,omitempty" validate:"required_if=Type DEST-PORT,required_if=Type HEADER-KEYWORD,required_if=Type HEADER-REGEX,required_if=Type IP-CIDR,required_if=Type SRC-IP,required_if=Type DOMAIN-SUFFIX,required_if=Type DOMAIN-KEYWORD,required_if=Type DOMAIN,required_if=Type DOMAIN-SET,required_if=Type URL-REGEX"`

	Action string `json:"action" yaml:"action" validate:"required,oneof=DIRECT REPLACE REPLACE-REGEX DELETE DROP ADD REDIRECT-302 REDIRECT-307 REDIRECT-HEADER REJECT"`

	RewriteHeader    string `json:"rewrite_header,omitempty" yaml:"rewrite-header,omitempty"` // validate:"required_if=Action REPLACE,required_if=Action REPLACE-REGEX,required_if=Action DELETE,required_if=Action ADD"
	RewriteValue     string `json:"rewrite_value,omitempty" yaml:"rewrite-value,omitempty" validate:"required_if=Action REPLACE,required_if=Action REPLACE-REGEX,required_if=Action ADD"`
	RewriteDirection string `json:"rewrite_direction,omitempty" yaml:"rewrite-direction,omitempty" validate:"omitempty,oneof=REQUEST RESPONSE"`

	RewriteRegex string `json:"rewrite_regex,omitempty" yaml:"rewrite-regex,omitempty" validate:"required_if=Action REPLACE-REGEX"`

	Continue bool `json:"continue,omitempty" yaml:"continue,omitempty"`
}

// ReloadFromFile re-reads the config file (if one was set) and builds a new Config.
// This is used by the /restart API to pick up configuration changes without
// restarting the entire process.
func ReloadFromFile() (*Config, error) {
	if viper.ConfigFileUsed() != "" {
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to re-read config file: %w", err)
		}
	}
	return BuildConfigFromViper()
}

// BuildConfigFromViper constructs a Config from viper settings.
// Viper merges values from defaults, config file, env vars and CLI flags
// with the correct priority: defaults < config file < env vars < CLI flags.
func BuildConfigFromViper() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Normalize case
	cfg.ServerMode = ServerMode(strings.ToUpper(string(cfg.ServerMode)))
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.RewriteMode = RewriteMode(strings.ToUpper(string(cfg.RewriteMode)))

	// Backwards compatibility: convert deprecated "RULES" value to "RULE".
	if cfg.RewriteMode == "RULES" {
		cfg.RewriteMode = RewriteModeRule
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
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
		slog.Attr{
			Key: "MitM", Value: slog.GroupValue(
				slog.Bool("Enabled", c.MitM.Enabled),
				slog.String("Hostname", c.MitM.Hostname),
				slog.Bool("Insecure Skip Verify", c.MitM.InsecureSkipVerify),
			),
		},
	)
}
