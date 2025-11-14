//go:build !linux

package redirect

import (
	"errors"
	"net"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/utils"
)

type Server struct {
	cfg *config.Config
	rw  *rewrite.Rewriter
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	return &Server{
		cfg: cfg,
		rw:  rw,
	}
}

func (s *Server) Start() error {
	return errors.New("redirect server is only supported on linux")
}

func (s *Server) Close() error {
	return nil
}

func (s *Server) HandleClient(client net.Conn) {
	defer func() {
		_ = client.Close()
	}()
}

func (s *Server) ForwardTCP(client, target net.Conn, _ string) {
	go utils.CopyHalf(client, target)
	go utils.CopyHalf(target, client)
}
