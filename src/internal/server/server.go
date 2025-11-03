package server

import (
	"fmt"
	"net"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/http"
	"github.com/sunbk201/ua3f/internal/server/redirect"
	"github.com/sunbk201/ua3f/internal/server/socks5"
	"github.com/sunbk201/ua3f/internal/server/tproxy"
)

type Server interface {
	Start() error
	HandleClient(net.Conn)
	ForwardTCP(client, target net.Conn, destAddr string)
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
	default:
		return nil, fmt.Errorf("unknown server mode: %s", cfg.ServerMode)
	}
}
