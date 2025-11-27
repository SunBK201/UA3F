//go:build !linux

package usergroup

import (
	"github.com/sunbk201/ua3f/internal/config"
)

func SetUserGroup(cfg *config.Config) error {
	return nil
}
