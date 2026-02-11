package common

import (
	"github.com/sunbk201/ua3f/internal/config"
)

type Server interface {
	Start() error
	Close() error
	Restart(cfg *config.Config) (Server, error)
	GetRewriter() Rewriter
}
