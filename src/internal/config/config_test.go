package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// resetViper resets viper global state and sets the required defaults
// to mirror what initConfig() in cmd/root.go does.
func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	viper.SetDefault("server-mode", "SOCKS5")
	viper.SetDefault("bind-address", "127.0.0.1")
	viper.SetDefault("port", 1080)
	viper.SetDefault("log-level", "info")
	viper.SetDefault("user-agent", "FFF")
	viper.SetDefault("rewrite-mode", "GLOBAL")
	viper.SetDefault("desync.reorder-bytes", 8)
	viper.SetDefault("desync.reorder-packets", 1500)
	viper.SetDefault("desync.inject-ttl", 3)
}

// writeConfigFile writes YAML content to a temp file and configures viper to read it.
func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

// loadConfigFile merges a YAML config file into viper.
func loadConfigFile(t *testing.T, path string) {
	t.Helper()
	viper.SetConfigFile(path)
	if err := viper.MergeInConfig(); err != nil {
		t.Fatalf("failed to merge config file: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	resetViper(t)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"ServerMode", cfg.ServerMode, ServerModeSocks5},
		{"BindAddress", cfg.BindAddress, "127.0.0.1"},
		{"Port", cfg.Port, 1080},
		{"LogLevel", cfg.LogLevel, "info"},
		{"RewriteMode", cfg.RewriteMode, RewriteModeGlobal},
		{"UserAgent", cfg.UserAgent, "FFF"},
		{"UserAgentRegex", cfg.UserAgentRegex, ""},
		{"UserAgentPartialReplace", cfg.UserAgentPartialReplace, false},
		{"TTL", cfg.TTL, false},
		{"IPID", cfg.IPID, false},
		{"TCPTimeStamp", cfg.TCPTimeStamp, false},
		{"TCPInitialWindow", cfg.TCPInitialWindow, false},
		{"Desync.Reorder", cfg.Desync.Reorder, false},
		{"Desync.ReorderBytes", cfg.Desync.ReorderBytes, uint32(8)},
		{"Desync.ReorderPackets", cfg.Desync.ReorderPackets, uint32(1500)},
		{"Desync.Inject", cfg.Desync.Inject, false},
		{"Desync.InjectTTL", cfg.Desync.InjectTTL, uint8(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestConfigFromFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: HTTP
bind-address: 0.0.0.0
port: 8080
log-level: debug
rewrite-mode: RULE
user-agent: "TestAgent"
user-agent-regex: "(Android|iOS)"
user-agent-partial-replace: true
ttl: true
ipid: true
tcp_timestamp: true
tcp_initial_window: true
desync:
  reorder: true
  reorder-bytes: 16
  reorder-packets: 3000
  inject: true
  inject-ttl: 5
  desync-ports: "80,443"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerMode != ServerModeHTTP {
		t.Errorf("ServerMode = %v, want %v", cfg.ServerMode, ServerModeHTTP)
	}
	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %v, want 0.0.0.0", cfg.BindAddress)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.RewriteMode != RewriteModeRule {
		t.Errorf("RewriteMode = %v, want %v", cfg.RewriteMode, RewriteModeRule)
	}
	if cfg.UserAgent != "TestAgent" {
		t.Errorf("UserAgent = %v, want TestAgent", cfg.UserAgent)
	}
	if cfg.UserAgentRegex != "(Android|iOS)" {
		t.Errorf("UserAgentRegex = %v, want (Android|iOS)", cfg.UserAgentRegex)
	}
	if !cfg.UserAgentPartialReplace {
		t.Error("UserAgentPartialReplace should be true")
	}
	if !cfg.TTL {
		t.Error("TTL should be true")
	}
	if !cfg.IPID {
		t.Error("IPID should be true")
	}
	if !cfg.TCPTimeStamp {
		t.Error("TCPTimeStamp should be true")
	}
	if !cfg.TCPInitialWindow {
		t.Error("TCPInitialWindow should be true")
	}
	if !cfg.Desync.Reorder {
		t.Error("Desync.Reorder should be true")
	}
	if cfg.Desync.ReorderBytes != 16 {
		t.Errorf("Desync.ReorderBytes = %v, want 16", cfg.Desync.ReorderBytes)
	}
	if cfg.Desync.ReorderPackets != 3000 {
		t.Errorf("Desync.ReorderPackets = %v, want 3000", cfg.Desync.ReorderPackets)
	}
	if !cfg.Desync.Inject {
		t.Error("Desync.Inject should be true")
	}
	if cfg.Desync.InjectTTL != 5 {
		t.Errorf("Desync.InjectTTL = %v, want 5", cfg.Desync.InjectTTL)
	}
	if cfg.Desync.DesyncPorts != "80,443" {
		t.Errorf("Desync.DesyncPorts = %v, want 80,443", cfg.Desync.DesyncPorts)
	}
}

func TestHeaderRulesFromFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "MicroMessenger Client"
    action: DIRECT
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 3 {
		t.Fatalf("HeaderRules count = %d, want 3", len(cfg.HeaderRules))
	}

	r0 := cfg.HeaderRules[0]
	if r0.Type != "DEST-PORT" || r0.MatchValue != "22" || r0.Action != "DIRECT" {
		t.Errorf("HeaderRule[0] = %+v", r0)
	}

	r1 := cfg.HeaderRules[1]
	if r1.Type != "HEADER-KEYWORD" || r1.MatchHeader != "User-Agent" || r1.MatchValue != "MicroMessenger Client" || r1.Action != "DIRECT" {
		t.Errorf("HeaderRule[1] = %+v", r1)
	}

	r2 := cfg.HeaderRules[2]
	if r2.Type != "FINAL" || r2.Action != "REPLACE" || r2.RewriteHeader != "User-Agent" || r2.RewriteValue != "FFF" {
		t.Errorf("HeaderRule[2] = %+v", r2)
	}
}

func TestBodyRulesFromFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

body-rewrite:
  - type: URL-REGEX
    match-value: "^http://example.com"
    action: REPLACE-REGEX
    rewrite-direction: RESPONSE
    rewrite-regex: "OldValue"
    rewrite-value: "NewValue"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.BodyRules) != 1 {
		t.Fatalf("BodyRules count = %d, want 1", len(cfg.BodyRules))
	}

	r := cfg.BodyRules[0]
	if r.Type != "URL-REGEX" {
		t.Errorf("Type = %v, want URL-REGEX", r.Type)
	}
	if r.MatchValue != "^http://example.com" {
		t.Errorf("MatchValue = %v", r.MatchValue)
	}
	if r.Action != "REPLACE-REGEX" {
		t.Errorf("Action = %v, want REPLACE-REGEX", r.Action)
	}
	if r.RewriteDirection != "RESPONSE" {
		t.Errorf("RewriteDirection = %v, want RESPONSE", r.RewriteDirection)
	}
	if r.RewriteRegex != "OldValue" {
		t.Errorf("RewriteRegex = %v, want OldValue", r.RewriteRegex)
	}
	if r.RewriteValue != "NewValue" {
		t.Errorf("RewriteValue = %v, want NewValue", r.RewriteValue)
	}
}

func TestURLRedirectRulesFromFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com/old"
    action: REDIRECT-302
    rewrite-regex: "^http://example.com/old(.*)"
    rewrite-value: "http://example.com/new$1"
  - type: URL-REGEX
    match-value: "^http://test.com/"
    action: REDIRECT-307
    rewrite-regex: "^http://test.com/(.*)"
    rewrite-value: "http://test.com:8080/$1"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.URLRedirectRules) != 2 {
		t.Fatalf("URLRedirectRules count = %d, want 2", len(cfg.URLRedirectRules))
	}

	r0 := cfg.URLRedirectRules[0]
	if r0.Action != "REDIRECT-302" {
		t.Errorf("URLRedirectRules[0].Action = %v, want REDIRECT-302", r0.Action)
	}

	r1 := cfg.URLRedirectRules[1]
	if r1.Action != "REDIRECT-307" {
		t.Errorf("URLRedirectRules[1].Action = %v, want REDIRECT-307", r1.Action)
	}
}

func TestCaseNormalization(t *testing.T) {
	resetViper(t)

	// Use lowercase values — BuildConfigFromViper should normalize them
	viper.Set("server-mode", "http")
	viper.Set("log-level", "DEBUG")
	viper.Set("rewrite-mode", "rule")

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerMode != ServerModeHTTP {
		t.Errorf("ServerMode = %v, want HTTP", cfg.ServerMode)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.RewriteMode != RewriteModeRule {
		t.Errorf("RewriteMode = %v, want RULE", cfg.RewriteMode)
	}
}

func TestBackwardCompatRULES(t *testing.T) {
	resetViper(t)

	// The deprecated "RULES" value should map to "RULE"
	viper.Set("rewrite-mode", "RULES")

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RewriteMode != RewriteModeRule {
		t.Errorf("RewriteMode = %v, want RULE (from deprecated RULES)", cfg.RewriteMode)
	}
}

func TestAllServerModes(t *testing.T) {
	modes := []struct {
		input string
		want  ServerMode
	}{
		{"HTTP", ServerModeHTTP},
		{"SOCKS5", ServerModeSocks5},
		{"TPROXY", ServerModeTProxy},
		{"REDIRECT", ServerModeRedirect},
		{"NFQUEUE", ServerModeNFQueue},
		{"http", ServerModeHTTP},
		{"socks5", ServerModeSocks5},
		{"tproxy", ServerModeTProxy},
		{"redirect", ServerModeRedirect},
		{"nfqueue", ServerModeNFQueue},
	}

	for _, tt := range modes {
		t.Run(tt.input, func(t *testing.T) {
			resetViper(t)
			viper.Set("server-mode", tt.input)

			cfg, err := BuildConfigFromViper()
			if err != nil {
				t.Fatalf("unexpected error for mode %q: %v", tt.input, err)
			}
			if cfg.ServerMode != tt.want {
				t.Errorf("ServerMode = %v, want %v", cfg.ServerMode, tt.want)
			}
		})
	}
}

func TestAllRewriteModes(t *testing.T) {
	modes := []struct {
		input string
		want  RewriteMode
	}{
		{"GLOBAL", RewriteModeGlobal},
		{"DIRECT", RewriteModeDirect},
		{"RULE", RewriteModeRule},
		{"global", RewriteModeGlobal},
		{"direct", RewriteModeDirect},
		{"rule", RewriteModeRule},
		{"RULES", RewriteModeRule}, // backward compat
		{"rules", RewriteModeRule}, // backward compat + case
	}

	for _, tt := range modes {
		t.Run(tt.input, func(t *testing.T) {
			resetViper(t)
			viper.Set("rewrite-mode", tt.input)

			cfg, err := BuildConfigFromViper()
			if err != nil {
				t.Fatalf("unexpected error for rewrite-mode %q: %v", tt.input, err)
			}
			if cfg.RewriteMode != tt.want {
				t.Errorf("RewriteMode = %v, want %v", cfg.RewriteMode, tt.want)
			}
		})
	}
}

func TestValidation_InvalidServerMode(t *testing.T) {
	resetViper(t)
	viper.Set("server-mode", "INVALID")

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid server mode, got nil")
	}
}

func TestValidation_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too_large", 70000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("port", tt.port)

			_, err := BuildConfigFromViper()
			if err == nil {
				t.Fatalf("expected validation error for port %d, got nil", tt.port)
			}
		})
	}
}

func TestValidation_InvalidLogLevel(t *testing.T) {
	resetViper(t)
	viper.Set("log-level", "TRACE")

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid log-level, got nil")
	}
}

func TestValidation_InvalidRewriteMode(t *testing.T) {
	resetViper(t)
	viper.Set("rewrite-mode", "INVALID")

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid rewrite-mode, got nil")
	}
}

func TestValidation_InvalidBindAddress(t *testing.T) {
	resetViper(t)
	viper.Set("bind-address", "not-an-ip")

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid bind-address, got nil")
	}
}

func TestValidation_InvalidRuleType(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: INVALID-TYPE
    match-value: "test"
    action: DIRECT
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid rule type, got nil")
	}
}

func TestValidation_InvalidRuleAction(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: FINAL
    action: INVALID-ACTION
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid rule action, got nil")
	}
}

func TestValidation_MissingMatchValueForDestPort(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: DEST-PORT
    action: DIRECT
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for missing match-value on DEST-PORT, got nil")
	}
}

func TestValidation_MissingMatchHeaderForHeaderKeyword(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: HEADER-KEYWORD
    match-value: "test"
    action: DIRECT
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for missing match-header on HEADER-KEYWORD, got nil")
	}
}

func TestValidation_InvalidRewriteDirection(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
    rewrite-direction: INVALID
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for invalid rewrite-direction, got nil")
	}
}

func TestValidPortBoundaries(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"min", 1},
		{"max", 65535},
		{"typical", 8080},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("port", tt.port)

			cfg, err := BuildConfigFromViper()
			if err != nil {
				t.Fatalf("unexpected error for port %d: %v", tt.port, err)
			}
			if cfg.Port != tt.port {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.port)
			}
		})
	}
}

func TestConfigFileOverridesDefaults(t *testing.T) {
	resetViper(t)

	yaml := `
port: 9090
user-agent: "CustomUA"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden values
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.UserAgent != "CustomUA" {
		t.Errorf("UserAgent = %v, want CustomUA", cfg.UserAgent)
	}

	// Defaults should still be in effect
	if cfg.ServerMode != ServerModeSocks5 {
		t.Errorf("ServerMode = %v, want %v (default)", cfg.ServerMode, ServerModeSocks5)
	}
	if cfg.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %v, want 127.0.0.1 (default)", cfg.BindAddress)
	}
}

func TestViperSetOverridesConfigFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: HTTP
port: 9090
user-agent: "FileUA"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	// Simulate CLI flag override via viper.Set (highest priority)
	viper.Set("port", 7070)
	viper.Set("user-agent", "CLIUA")

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CLI override should win
	if cfg.Port != 7070 {
		t.Errorf("Port = %d, want 7070 (CLI override)", cfg.Port)
	}
	if cfg.UserAgent != "CLIUA" {
		t.Errorf("UserAgent = %v, want CLIUA (CLI override)", cfg.UserAgent)
	}
	// Config file value should still apply for un-overridden keys
	if cfg.ServerMode != ServerModeHTTP {
		t.Errorf("ServerMode = %v, want HTTP (from file)", cfg.ServerMode)
	}
}

func TestEnvVarOverridesDefault(t *testing.T) {
	resetViper(t)

	_ = viper.BindEnv("user-agent", "UA3F_PAYLOAD_UA")
	_ = viper.BindEnv("port", "UA3F_PORT")

	t.Setenv("UA3F_PAYLOAD_UA", "EnvUA")
	t.Setenv("UA3F_PORT", "3333")

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.UserAgent != "EnvUA" {
		t.Errorf("UserAgent = %v, want EnvUA (from env)", cfg.UserAgent)
	}
	if cfg.Port != 3333 {
		t.Errorf("Port = %d, want 3333 (from env)", cfg.Port)
	}
}

func TestDesyncConfigFromFile(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: GLOBAL
user-agent: FFF

desync:
  reorder: true
  reorder-bytes: 32
  reorder-packets: 500
  inject: true
  inject-ttl: 10
  desync-ports: "80,443,8080"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Desync.Reorder {
		t.Error("Desync.Reorder should be true")
	}
	if cfg.Desync.ReorderBytes != 32 {
		t.Errorf("Desync.ReorderBytes = %d, want 32", cfg.Desync.ReorderBytes)
	}
	if cfg.Desync.ReorderPackets != 500 {
		t.Errorf("Desync.ReorderPackets = %d, want 500", cfg.Desync.ReorderPackets)
	}
	if !cfg.Desync.Inject {
		t.Error("Desync.Inject should be true")
	}
	if cfg.Desync.InjectTTL != 10 {
		t.Errorf("Desync.InjectTTL = %d, want 10", cfg.Desync.InjectTTL)
	}
	if cfg.Desync.DesyncPorts != "80,443,8080" {
		t.Errorf("Desync.DesyncPorts = %v, want 80,443,8080", cfg.Desync.DesyncPorts)
	}
}

func TestRuleContinueFlag(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: HEADER-KEYWORD
    match-header: "User-Agent"
    match-value: "Test"
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "Replaced"
    continue: true
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 2 {
		t.Fatalf("HeaderRules count = %d, want 2", len(cfg.HeaderRules))
	}
	if !cfg.HeaderRules[0].Continue {
		t.Error("HeaderRules[0].Continue should be true")
	}
	if cfg.HeaderRules[1].Continue {
		t.Error("HeaderRules[1].Continue should be false")
	}
}

func TestRuleEnabledFlag(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
    enabled: true
  - type: DEST-PORT
    match-value: "22"
    action: DIRECT
    enabled: false
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 2 {
		t.Fatalf("HeaderRules count = %d, want 2", len(cfg.HeaderRules))
	}
	if !cfg.HeaderRules[0].Enabled {
		t.Error("HeaderRules[0].Enabled should be true")
	}
	if cfg.HeaderRules[1].Enabled {
		t.Error("HeaderRules[1].Enabled should be false")
	}
}

func TestHeaderRegexRule(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Apple|Android)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-regex: "(Apple|Android)"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 1 {
		t.Fatalf("HeaderRules count = %d, want 1", len(cfg.HeaderRules))
	}

	r := cfg.HeaderRules[0]
	if r.Type != "HEADER-REGEX" {
		t.Errorf("Type = %v, want HEADER-REGEX", r.Type)
	}
	if r.MatchHeader != "User-Agent" {
		t.Errorf("MatchHeader = %v", r.MatchHeader)
	}
	if r.RewriteRegex != "(Apple|Android)" {
		t.Errorf("RewriteRegex = %v", r.RewriteRegex)
	}
}

func TestDocsConfigFile(t *testing.T) {
	resetViper(t)

	// Test that the example docs/config.yaml parses without error
	docsConfig := filepath.Join("..", "..", "..", "docs", "config.yaml")
	if _, err := os.Stat(docsConfig); os.IsNotExist(err) {
		t.Skip("docs/config.yaml not found, skipping")
	}
	loadConfigFile(t, docsConfig)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("docs/config.yaml failed to parse: %v", err)
	}

	if cfg.ServerMode != ServerModeSocks5 {
		t.Errorf("ServerMode = %v, want SOCKS5", cfg.ServerMode)
	}
	if cfg.RewriteMode != RewriteModeRule {
		t.Errorf("RewriteMode = %v, want RULE", cfg.RewriteMode)
	}
	if len(cfg.HeaderRules) == 0 {
		t.Error("expected header-rewrite rules from docs config")
	}
	if len(cfg.BodyRules) == 0 {
		t.Error("expected body-rewrite rules from docs config")
	}
	if len(cfg.URLRedirectRules) == 0 {
		t.Error("expected url-redirect rules from docs config")
	}
}

func TestLogValue(t *testing.T) {
	resetViper(t)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LogValue should not panic
	val := cfg.LogValue()
	if val.Kind() != slog.KindGroup {
		t.Errorf("LogValue().Kind() = %v, want Group", val.Kind())
	}
}

func TestUnmarshalDirectly(t *testing.T) {
	// Test that our Config struct tags match what viper produces
	resetViper(t)

	viper.Set("server-mode", "NFQUEUE")
	viper.Set("bind-address", "192.168.1.1")
	viper.Set("port", 2222)
	viper.Set("log-level", "warn")
	viper.Set("rewrite-mode", "DIRECT")
	viper.Set("user-agent", "CustomAgent")
	viper.Set("ttl", true)
	viper.Set("ipid", true)

	var cfg Config
	err := viper.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if string(cfg.ServerMode) != "NFQUEUE" {
		t.Errorf("ServerMode = %v, want NFQUEUE", cfg.ServerMode)
	}
	if cfg.BindAddress != "192.168.1.1" {
		t.Errorf("BindAddress = %v, want 192.168.1.1", cfg.BindAddress)
	}
	if cfg.Port != 2222 {
		t.Errorf("Port = %d, want 2222", cfg.Port)
	}
	if cfg.UserAgent != "CustomAgent" {
		t.Errorf("UserAgent = %v, want CustomAgent", cfg.UserAgent)
	}
	if !cfg.TTL {
		t.Error("TTL should be true")
	}
	if !cfg.IPID {
		t.Error("IPID should be true")
	}
}

func TestMultipleRuleTypes(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: DOMAIN-SUFFIX
    match-value: "example.com"
    action: ADD
    rewrite-direction: REQUEST
    rewrite-header: "X-Custom"
    rewrite-value: "test"
  - type: DOMAIN-KEYWORD
    match-value: "example"
    action: DIRECT
  - type: DOMAIN
    match-value: "exact.example.com"
    action: DIRECT
  - type: IP-CIDR
    match-value: "192.168.0.0/16"
    action: DIRECT
  - type: SRC-IP
    match-value: "10.0.0.1"
    action: DROP
  - type: URL-REGEX
    match-value: "^http://test"
    action: DIRECT
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 7 {
		t.Fatalf("HeaderRules count = %d, want 7", len(cfg.HeaderRules))
	}

	expectedTypes := []string{
		"DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "DOMAIN",
		"IP-CIDR", "SRC-IP", "URL-REGEX", "FINAL",
	}
	for i, expected := range expectedTypes {
		if cfg.HeaderRules[i].Type != expected {
			t.Errorf("HeaderRules[%d].Type = %v, want %v", i, cfg.HeaderRules[i].Type, expected)
		}
	}
}

func TestValidation_MissingRewriteRegexForReplaceRegex(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: HEADER-REGEX
    match-header: "User-Agent"
    match-value: "(Test)"
    action: REPLACE-REGEX
    rewrite-header: "User-Agent"
    rewrite-value: "FFF"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for missing rewrite-regex on REPLACE-REGEX action, got nil")
	}
}

func TestValidation_MissingRewriteValueForReplace(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: FINAL
    action: REPLACE
    rewrite-header: "User-Agent"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	_, err := BuildConfigFromViper()
	if err == nil {
		t.Fatal("expected validation error for missing rewrite-value on REPLACE action, got nil")
	}
}

func TestEmptyRules(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: GLOBAL
user-agent: FFF
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 0 {
		t.Errorf("HeaderRules count = %d, want 0", len(cfg.HeaderRules))
	}
	if len(cfg.BodyRules) != 0 {
		t.Errorf("BodyRules count = %d, want 0", len(cfg.BodyRules))
	}
	if len(cfg.URLRedirectRules) != 0 {
		t.Errorf("URLRedirectRules count = %d, want 0", len(cfg.URLRedirectRules))
	}
}

func TestDeleteAction(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

header-rewrite:
  - type: FINAL
    action: DELETE
    rewrite-header: "X-Unwanted"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.HeaderRules) != 1 {
		t.Fatalf("HeaderRules count = %d, want 1", len(cfg.HeaderRules))
	}
	if cfg.HeaderRules[0].Action != "DELETE" {
		t.Errorf("Action = %v, want DELETE", cfg.HeaderRules[0].Action)
	}
}

func TestRedirectHeaderAction(t *testing.T) {
	resetViper(t)

	yaml := `
server-mode: SOCKS5
bind-address: 127.0.0.1
port: 1080
log-level: info
rewrite-mode: RULE
user-agent: FFF

url-redirect:
  - type: URL-REGEX
    match-value: "^http://example.com"
    action: REDIRECT-HEADER
    rewrite-regex: "^http://example.com(.*)"
    rewrite-value: "http://new.example.com$1"
`
	path := writeConfigFile(t, yaml)
	loadConfigFile(t, path)

	cfg, err := BuildConfigFromViper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.URLRedirectRules) != 1 {
		t.Fatalf("URLRedirectRules count = %d, want 1", len(cfg.URLRedirectRules))
	}
	if cfg.URLRedirectRules[0].Action != "REDIRECT-HEADER" {
		t.Errorf("Action = %v, want REDIRECT-HEADER", cfg.URLRedirectRules[0].Action)
	}
}
