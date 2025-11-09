package server

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/http"
	"github.com/sunbk201/ua3f/internal/server/nfqueue"
	"github.com/sunbk201/ua3f/internal/server/redirect"
	"github.com/sunbk201/ua3f/internal/server/socks5"
	"github.com/sunbk201/ua3f/internal/server/tproxy"
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
}

func NewServer(cfg *config.Config, rw *rewrite.Rewriter) (Server, error) {
	switch cfg.ServerMode {
	case config.ServerModeHTTP:
		return http.New(cfg, rw), nil
	case config.ServerModeSocks5:
		return socks5.New(cfg, rw), nil
	case config.ServerModeTProxy:
		return tproxy.New(cfg, rw), nil
	case config.ServerModeRedirect:
		return redirect.New(cfg, rw), nil
	case config.ServerModeNFQueue:
		return nfqueue.New(cfg, rw), nil
	default:
		return nil, fmt.Errorf("NewServer unknown server mode: %s", cfg.ServerMode)
	}
}
