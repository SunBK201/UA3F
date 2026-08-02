package cmd

import (
	"testing"

	"github.com/sunbk201/ua3f/internal/config"
)

func TestTTLValueConfigurationSources(t *testing.T) {
	initConfig()

	t.Run("environment", func(t *testing.T) {
		t.Setenv("UA3F_L3_REWRITE_TTL_VALUE", "128")

		cfg, err := config.BuildConfigFromViper()
		if err != nil {
			t.Fatalf("BuildConfigFromViper() error = %v", err)
		}
		if cfg.L3Rewrite.TTLValue != 128 {
			t.Fatalf("TTLValue = %d, want 128", cfg.L3Rewrite.TTLValue)
		}
	})

	t.Run("command line overrides environment", func(t *testing.T) {
		flag := rootCmd.Flags().Lookup("l3-rewrite-ttl-value")
		if flag == nil {
			t.Fatal("l3-rewrite-ttl-value flag is not registered")
		}
		originalValue, originalChanged := flag.Value.String(), flag.Changed
		t.Cleanup(func() {
			_ = flag.Value.Set(originalValue)
			flag.Changed = originalChanged
		})

		t.Setenv("UA3F_L3_REWRITE_TTL_VALUE", "130")
		if err := rootCmd.Flags().Set("l3-rewrite-ttl-value", "131"); err != nil {
			t.Fatalf("set l3-rewrite-ttl-value flag: %v", err)
		}

		cfg, err := config.BuildConfigFromViper()
		if err != nil {
			t.Fatalf("BuildConfigFromViper() error = %v", err)
		}
		if cfg.L3Rewrite.TTLValue != 131 {
			t.Fatalf("TTLValue = %d, want 131", cfg.L3Rewrite.TTLValue)
		}
	})
}
