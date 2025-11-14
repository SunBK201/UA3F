//go:build !linux

package nfqueue

import (
	"errors"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
)

type Server struct {
	base.Server
}

func New(cfg *config.Config, rw *rewrite.Rewriter) *Server {
	s := &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
		},
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
