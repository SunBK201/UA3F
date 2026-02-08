package server

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/http"
	"github.com/sunbk201/ua3f/internal/server/nfqueue"
	"github.com/sunbk201/ua3f/internal/server/redirect"
	"github.com/sunbk201/ua3f/internal/server/socks5"
	"github.com/sunbk201/ua3f/internal/server/tproxy"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type ServerMode string

const (
	ServerModeHTTP     ServerMode = "HTTP"
	ServerModeSocks5   ServerMode = "SOCKS5"
	ServerModeTProxy   ServerMode = "TPROXY"
	ServerModeRedirect ServerMode = "REDIRECT"
	ServerModeNFQueue  ServerMode = "NFQUEUE"
)

type Server interface {
	Start() error
	Close() error
	GetRewriter() rewrite.Rewriter
}

func NewServer(cfg *config.Config) (Server, error) {
	rc := statistics.New()

	rw, err := rewrite.New(cfg, rc)
	if err != nil {
		slog.Error("rewrite.New", slog.Any("error", err))
		return nil, err
	}

	var middleMan *mitm.MiddleMan
	if cfg.MitM.Enabled {
		ca, err := mitm.LoadCA(cfg.MitM.CAP12Base64, cfg.MitM.CAPassphrase)
		if err != nil {
			return nil, fmt.Errorf("MitM CA init failed: %w", err)
		}
		slog.Info("MitM enabled, CA certificate loaded")
		certManager := mitm.NewCertManager(ca)
		hostnameFilter, err := mitm.NewHostnameFilter(cfg.MitM.Hostname)
		if err != nil {
			return nil, fmt.Errorf("MitM hostname filter init failed: %w", err)
		}
		middleMan = mitm.NewMiddleMan(certManager, hostnameFilter, cfg.MitM.InsecureSkipVerify)
	}

	switch cfg.ServerMode {
	case config.ServerModeHTTP:
		return http.New(cfg, rw, rc, middleMan), nil
	case config.ServerModeSocks5:
		return socks5.New(cfg, rw, rc, middleMan), nil
	case config.ServerModeTProxy:
		return tproxy.New(cfg, rw, rc, middleMan), nil
	case config.ServerModeRedirect:
		return redirect.New(cfg, rw, rc, middleMan), nil
	case config.ServerModeNFQueue:
		return nfqueue.New(cfg, rw, rc), nil
	default:
		return nil, fmt.Errorf("NewServer unknown server mode: %s", cfg.ServerMode)
	}
}
