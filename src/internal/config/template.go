package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

func GenerateTemplateConfig(writeToFile bool) (Config, error) {
	cfg := Config{
		ServerMode:  "SOCKS5",
		BindAddress: "127.0.0.1",
		Port:        1080,

		LogLevel: "info",

		RewriteMode: "GLOBAL",

		UserAgent:               "FFF",
		UserAgentRegex:          "",
		UserAgentPartialReplace: false,

		TTL:              false,
		IPID:             false,
		TCPTimeStamp:     false,
		TCPInitialWindow: false,

		Desync: DesyncConfig{
			Reorder:        false,
			ReorderBytes:   8,
			ReorderPackets: 1500,
			Inject:         false,
			InjectTTL:      3,
		},

		HeaderRules: []Rule{
			{
				Type:          "FINAL",
				Action:        "REPLACE",
				RewriteHeader: "User-Agent",
				RewriteValue:  "FFF",
			},
		},
	}

	if writeToFile {
		data, err := yaml.Marshal(&cfg)
		if err != nil {
			return Config{}, fmt.Errorf("failed to marshal template config to YAML: %w", err)
		}
		if err := os.WriteFile("config.yaml", data, 0644); err != nil {
			return Config{}, fmt.Errorf("failed to write template config to file: %w", err)
		}
	}
	return cfg, nil
}
