//go:build !linux

package base

import (
	"io"

	"github.com/sunbk201/ua3f/internal/common"
)

func (s *Server) TryOffload(_ *common.ConnLink, _ io.Reader) bool {
	return false
}

func (s *Server) DeleteOffload(_ *common.ConnLink) {}
