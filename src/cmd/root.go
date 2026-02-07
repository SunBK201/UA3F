package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/daemon"
	"github.com/sunbk201/ua3f/internal/log"
	"github.com/sunbk201/ua3f/internal/server"
	"github.com/sunbk201/ua3f/internal/server/desync"
	"github.com/sunbk201/ua3f/internal/server/netlink"
)

var (
	AppVersion    = "Development"
	shutdownChain []func() error
)

var rootCmd = &cobra.Command{
	Use:   "ua3f",
	Short: "UA3F is an HTTP rewriting tool",
	Long:  "UA3F is an HTTP rewriting tool that efficiently and transparently rewrites HTTP traffic (e.g., User-Agent) as an HTTP, SOCKS5, TPROXY, REDIRECT, or NFQUEUE service.",
	RunE:  runRoot,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Short flags
	rootCmd.Flags().StringP("config", "c", "", "Config file path")
	rootCmd.Flags().StringP("mode", "m", "", "Server mode: HTTP, SOCKS5, TPROXY, REDIRECT, NFQUEUE")
	rootCmd.Flags().StringP("bind", "b", "", "Bind address")
	rootCmd.Flags().IntP("port", "p", 0, "Port")
	rootCmd.Flags().StringP("log-level", "l", "", "Log level")
	rootCmd.Flags().StringP("ua", "f", "", "User-Agent")
	rootCmd.Flags().StringP("ua-regex", "r", "", "User-Agent regex")
	rootCmd.Flags().BoolP("partial", "s", false, "Enable regex partial replace")
	rootCmd.Flags().StringP("rewrite-mode", "x", "", "Rewrite mode: GLOBAL, DIRECT, RULE")
	rootCmd.Flags().BoolP("version", "v", false, "Show version")
	rootCmd.Flags().BoolP("generate-config", "g", false, "Generate template config file")

	// Long flags
	rootCmd.Flags().String("header-rewrite", "", "Header rewrite json rules")
	rootCmd.Flags().String("body-rewrite", "", "Body rewrite json rules")
	rootCmd.Flags().String("url-redirect", "", "URL redirect json rules")
	rootCmd.Flags().Bool("ttl", false, "Set TTL")
	rootCmd.Flags().Bool("ipid", false, "Set IP ID")
	rootCmd.Flags().Bool("tcpts", false, "Delete TCP Timestamp")
	rootCmd.Flags().Bool("tcpwin", false, "Set TCP Initial Window")
	rootCmd.Flags().Bool("desync-reorder", false, "Enable desync reorder")
	rootCmd.Flags().Uint("desync-reorder-bytes", 0, "Desync reorder bytes")
	rootCmd.Flags().Uint("desync-reorder-packets", 0, "Desync reorder packets")
	rootCmd.Flags().Bool("desync-inject", false, "Enable desync inject")
	rootCmd.Flags().Uint("desync-inject-ttl", 0, "Desync inject TTL")
	rootCmd.Flags().String("desync-ports", "", "Desync ports")

	// Bind all flags to viper using consistent key names
	_ = viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))
	_ = viper.BindPFlag("server-mode", rootCmd.Flags().Lookup("mode"))
	_ = viper.BindPFlag("bind-address", rootCmd.Flags().Lookup("bind"))
	_ = viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("log-level", rootCmd.Flags().Lookup("log-level"))
	_ = viper.BindPFlag("user-agent", rootCmd.Flags().Lookup("ua"))
	_ = viper.BindPFlag("user-agent-regex", rootCmd.Flags().Lookup("ua-regex"))
	_ = viper.BindPFlag("user-agent-partial-replace", rootCmd.Flags().Lookup("partial"))
	_ = viper.BindPFlag("rewrite-mode", rootCmd.Flags().Lookup("rewrite-mode"))
	_ = viper.BindPFlag("header-rewrite-json", rootCmd.Flags().Lookup("header-rewrite"))
	_ = viper.BindPFlag("body-rewrite-json", rootCmd.Flags().Lookup("body-rewrite"))
	_ = viper.BindPFlag("url-redirect-json", rootCmd.Flags().Lookup("url-redirect"))
	_ = viper.BindPFlag("ttl", rootCmd.Flags().Lookup("ttl"))
	_ = viper.BindPFlag("ipid", rootCmd.Flags().Lookup("ipid"))
	_ = viper.BindPFlag("tcp_timestamp", rootCmd.Flags().Lookup("tcpts"))
	_ = viper.BindPFlag("tcp_initial_window", rootCmd.Flags().Lookup("tcpwin"))
	_ = viper.BindPFlag("desync.reorder", rootCmd.Flags().Lookup("desync-reorder"))
	_ = viper.BindPFlag("desync.reorder-bytes", rootCmd.Flags().Lookup("desync-reorder-bytes"))
	_ = viper.BindPFlag("desync.reorder-packets", rootCmd.Flags().Lookup("desync-reorder-packets"))
	_ = viper.BindPFlag("desync.inject", rootCmd.Flags().Lookup("desync-inject"))
	_ = viper.BindPFlag("desync.inject-ttl", rootCmd.Flags().Lookup("desync-inject-ttl"))
	_ = viper.BindPFlag("desync.desync-ports", rootCmd.Flags().Lookup("desync-ports"))

	// Bind environment variables
	viper.SetEnvPrefix("UA3F")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Map specific env vars to viper keys for backward compatibility
	_ = viper.BindEnv("server-mode", "UA3F_SERVER_MODE")
	_ = viper.BindEnv("bind-address", "UA3F_BIND_ADDRESS")
	_ = viper.BindEnv("port", "UA3F_PORT")
	_ = viper.BindEnv("log-level", "UA3F_LOG_LEVEL")
	_ = viper.BindEnv("rewrite-mode", "UA3F_REWRITE_MODE")
	_ = viper.BindEnv("user-agent", "UA3F_PAYLOAD_UA")
	_ = viper.BindEnv("user-agent-regex", "UA3F_UA_REGEX")
	_ = viper.BindEnv("user-agent-partial-replace", "UA3F_PARTIAL_REPLACE")
	_ = viper.BindEnv("ttl", "UA3F_TTL")
	_ = viper.BindEnv("ipid", "UA3F_IPID")
	_ = viper.BindEnv("tcp_timestamp", "UA3F_TCPTS")
	_ = viper.BindEnv("tcp_initial_window", "UA3F_TCP_INIT_WINDOW")
	_ = viper.BindEnv("desync.reorder", "UA3F_DESYNC_REORDER")
	_ = viper.BindEnv("desync.reorder-bytes", "UA3F_DESYNC_REORDER_BYTES")
	_ = viper.BindEnv("desync.reorder-packets", "UA3F_DESYNC_REORDER_PACKETS")
	_ = viper.BindEnv("desync.inject", "UA3F_DESYNC_INJECT")
	_ = viper.BindEnv("desync.inject-ttl", "UA3F_DESYNC_INJECT_TTL")
	_ = viper.BindEnv("desync.desync-ports", "UA3F_DESYNC_PORTS")
	_ = viper.BindEnv("header-rewrite-json", "UA3F_HEADER_REWRITE")
	_ = viper.BindEnv("body-rewrite-json", "UA3F_BODY_REWRITE")
	_ = viper.BindEnv("url-redirect-json", "UA3F_URL_REDIRECT")
}

func initConfig() {
	configFile := viper.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.MergeInConfig(); err != nil {
			slog.Error("Failed to read config file", slog.Any("error", err))
			os.Exit(1)
		}
	}

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

func runRoot(cmd *cobra.Command, args []string) error {
	// Handle -v / --version
	showVer, _ := cmd.Flags().GetBool("version")
	if showVer {
		fmt.Printf("UA3F version %s\n", AppVersion)
		return nil
	}

	// Handle -g / --generate-config
	genConfig, _ := cmd.Flags().GetBool("generate-config")
	if genConfig {
		_, err := config.GenerateTemplateConfig(true)
		if err != nil {
			return fmt.Errorf("failed to generate template config: %w", err)
		}
		fmt.Println("Template config file 'config.yaml' generated successfully.")
		return nil
	}

	cfg, err := config.BuildConfigFromViper()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	log.SetLogConf(cfg.LogLevel)
	log.LogHeader(AppVersion, cfg)

	if err := daemon.DaemonSetup(cfg); err != nil {
		slog.Error("daemon.DaemonSetup", slog.Any("error", err))
		return err
	}

	helper := netlink.New(cfg)
	addShutdown("helper.Close", helper.Close)
	if err := helper.Start(); err != nil {
		slog.Error("helper.Start", slog.Any("error", err))
		shutdown()
		return err
	}

	if cfg.Desync.Reorder || cfg.Desync.Inject {
		d := desync.New(cfg)
		addShutdown("desync.Close", d.Close)
		if err := d.Start(); err != nil {
			slog.Error("desync.Start", slog.Any("error", err))
			shutdown()
			return err
		}
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		slog.Error("server.NewServer", slog.Any("error", err))
		shutdown()
		return err
	}
	addShutdown("srv.Close", srv.Close)
	if err := srv.Start(); err != nil {
		slog.Error("srv.Start", slog.Any("error", err))
		shutdown()
		return err
	}

	cleanup := make(chan os.Signal, 1)
	signal.Notify(cleanup, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	for {
		s := <-cleanup
		slog.Info("Received signal", slog.String("signal", s.String()))
		switch s {
		case syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM:
			shutdown()
			return nil
		case syscall.SIGHUP:
		default:
			return nil
		}
	}
}

func addShutdown(name string, fn func() error) {
	shutdownChain = append(shutdownChain, func() error {
		if err := fn(); err != nil {
			slog.Error(name, slog.Any("error", err))
			return err
		}
		return nil
	})
}

func shutdown() {
	for i := len(shutdownChain) - 1; i >= 0; i-- {
		_ = shutdownChain[i]()
	}
	slog.Info("UA3F exit")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
