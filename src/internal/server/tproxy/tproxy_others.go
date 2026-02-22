//go:build !linux

package tproxy

import (
	"errors"
	"net"

	"github.com/sunbk201/ua3f/internal/bpf"
	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/mitm"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	base.Server
}

func New(cfg *config.Config, rw common.Rewriter, rc *statistics.Recorder, middleMan *mitm.MiddleMan, bpf *bpf.BPF) *Server {
	return &Server{
		Server: base.Server{Cfg: cfg, Rewriter: rw, Recorder: rc, MiddleMan: middleMan, BPF: bpf},
	}
}

func (s *Server) Start() error {
	return errors.New("tproxy server is only supported on linux")
}

func (s *Server) Close() error {
	return nil
}

func (s *Server) Restart(cfg *config.Config) (common.Server, error) {
	if err := s.Close(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *Server) HandleClient(client net.Conn) {
	defer func() {
		_ = client.Close()
	}()
}
