//go:build !linux

package sockmap

import "github.com/sunbk201/ua3f/internal/config"

type Sockmap struct{}

func NewSockmap(_ *config.Config) (*Sockmap, error) {
	return nil, nil
}

func (s *Sockmap) Close() error {
	return nil
}
