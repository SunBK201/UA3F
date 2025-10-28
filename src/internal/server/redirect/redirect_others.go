//go:build !linux

package redirect

import (
	"fmt"
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
	return fmt.Errorf("REDIRECT Mode is only supported on Linux")
}

func (s *Server) HandleClient(client net.Conn) {
	defer client.Close()
}

func (s *Server) ForwardTCP(client, target net.Conn, _ string) {
	go utils.CopyHalf(client, target)
	go utils.CopyHalf(target, client)
}
