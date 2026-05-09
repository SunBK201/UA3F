//go:build !linux

package daemon

import (
	"github.com/sunbk201/ua3f/internal/config"
)

func SetUserGroup(cfg *config.Config) error {
	return nil
}

func SetOOMScoreAdj(value int) error {
	return nil
}
