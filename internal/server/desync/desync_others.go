//go:build !linux

package desync

import (
	"github.com/sunbk201/ua3f/internal/config"
)

type Server struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
	}
	return s
}

func (s *Server) Setup() (err error) {
	return nil
}

func (s *Server) Start() (err error) {
	return nil
}

func (s *Server) Close() (err error) {
	return nil
}

func (s *Server) Restart(cfg *config.Config) (*Server, error) {
	if err := s.Close(); err != nil {
		return nil, err
	}
	return nil, nil
}
