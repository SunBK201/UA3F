//go:build !linux

package bpf

import (
	"io"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
)

type BPF struct{}

func NewBPF(cfg *config.Config) (*BPF, error) {
	return nil, nil
}

func (b *BPF) Start() {}

func (b *BPF) Close() {}

func (b *BPF) TryOffload(_ *common.ConnLink, _ io.Reader) bool {
	return false
}

func (b *BPF) DeleteOffload(_ *common.ConnLink) {}
