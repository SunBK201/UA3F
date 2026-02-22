package server

import (
	"fmt"
	"log/slog"

	"github.com/sunbk201/ua3f/internal/bpf"
	"github.com/sunbk201/ua3f/internal/common"
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

func NewServer(cfg *config.Config) (common.Server, error) {
	rc := statistics.New()

	rw, err := rewrite.New(cfg, rc)
	if err != nil {
		slog.Error("rewrite.New", slog.Any("error", err))
		return nil, err
	}

	middleMan, err := mitm.NewMiddleMan(cfg)
	if err != nil {
		slog.Error("mitm.NewMiddleMan", slog.Any("error", err))
		return nil, err
	}

	bpf, err := bpf.NewBPF(cfg)
	if err != nil {
		slog.Error("bpf.NewBPF", slog.Any("error", err))
		return nil, err
	}

	switch cfg.ServerMode {
	case config.ServerModeHTTP:
		return http.New(cfg, rw, rc, middleMan, bpf), nil
	case config.ServerModeSocks5:
		return socks5.New(cfg, rw, rc, middleMan, bpf), nil
	case config.ServerModeTProxy:
		return tproxy.New(cfg, rw, rc, middleMan, bpf), nil
	case config.ServerModeRedirect:
		return redirect.New(cfg, rw, rc, middleMan, bpf), nil
	case config.ServerModeNFQueue:
		return nfqueue.New(cfg, rw, rc), nil
	default:
		return nil, fmt.Errorf("NewServer unknown server mode: %s", cfg.ServerMode)
	}
}
