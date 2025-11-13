//go:build !linux

package nfqueue

import (
	"errors"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
)

type Server struct {
	cfg *config.Config
	rw  *rewrite.Rewriter
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		cfg: cfg,
		rw:  rw,
	}
	return s
}

func (s *Server) Setup() (err error) {
	return nil
}

func (s *Server) Start() (err error) {
	return errors.New("nfqueue server is only supported on linux")
}

func (s *Server) Close() (err error) {
	return nil
}
