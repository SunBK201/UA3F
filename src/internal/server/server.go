package server

import (
	"fmt"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/socks5"
)

type Server interface {
	Start() error
}

func NewServer(cfg *config.Config, rw *rewrite.Rewriter) (Server, error) {
	switch cfg.ServerMode {
	case config.ServerModeSocks5:
		return socks5.New(cfg, rw), nil
	default:
		return nil, fmt.Errorf("unknown server mode: %s", cfg.ServerMode)
	}
}
