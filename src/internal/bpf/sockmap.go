//go:build linux

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -tags linux sockmap sockmap.c

package bpf

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

type Sockmap struct {
	Objs        *sockmapObjects
	ParserLink  link.Link
	VerdictLink link.Link
	rawAttach   bool // true when using BPF_PROG_ATTACH fallback
}

func NewSockmap() (*Sockmap, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("remove memlock: %w", err)
	}

	var objs sockmapObjects
	if err := loadSockmapObjects(&objs, nil); err != nil {
		return nil, fmt.Errorf("load objs: %w", err)
	}

	// Try BPF_LINK_CREATE first (kernel 6.10+), fall back to BPF_PROG_ATTACH.
	sm, err := attachSockmapLink(&objs)
	if err != nil {
		sm, err = attachSockmapProgAttach(&objs)
	}
	if err != nil {
		objs.Close()
		return nil, err
	}

	return sm, nil
}

func (s *Sockmap) Close() error {
	var errs []error

	if s.rawAttach {
		if s.Objs != nil {
			if err := link.RawDetachProgram(link.RawDetachProgramOptions{
				Target:  s.Objs.Sockhash.FD(),
				Program: s.Objs.StreamVerdict,
				Attach:  ebpf.AttachSkSKBStreamVerdict,
			}); err != nil {
				errs = append(errs, fmt.Errorf("detach verdict prog: %w", err))
			}
			if err := link.RawDetachProgram(link.RawDetachProgramOptions{
				Target:  s.Objs.Sockhash.FD(),
				Program: s.Objs.StreamParser,
				Attach:  ebpf.AttachSkSKBStreamParser,
			}); err != nil {
				errs = append(errs, fmt.Errorf("detach parser prog: %w", err))
			}
		}
	} else {
		if s.VerdictLink != nil {
			if err := s.VerdictLink.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close verdict link: %w", err))
			}
		}
		if s.ParserLink != nil {
			if err := s.ParserLink.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close parser link: %w", err))
			}
		}
	}

	if s.Objs != nil {
		if err := s.Objs.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close objs: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %w", errors.Join(errs...))
	}
	return nil
}

func (s *Sockmap) Add(lfd, rfd int, lc, rc uint64) (err error) {
	// 1) sockhash: cookie -> socket(fd)
	if err := s.Objs.Sockhash.Update(lc, uint32(lfd), ebpf.UpdateAny); err != nil {
		return fmt.Errorf("sockhash put l: %w", err)
	}
	if err := s.Objs.Sockhash.Update(rc, uint32(rfd), ebpf.UpdateAny); err != nil {
		_ = s.Objs.Sockhash.Delete(lc)
		return fmt.Errorf("sockhash put r: %w", err)
	}

	// 2) peer: cookie -> peer_cookie
	if err := s.Objs.Peer.Update(rc, lc, ebpf.UpdateAny); err != nil {
		_ = s.Objs.Sockhash.Delete(lc)
		_ = s.Objs.Sockhash.Delete(rc)
		return fmt.Errorf("peer put r: %w", err)
	}
	if err := s.Objs.Peer.Update(lc, rc, ebpf.UpdateAny); err != nil {
		_ = s.Objs.Peer.Delete(lc)
		_ = s.Objs.Sockhash.Delete(lc)
		_ = s.Objs.Sockhash.Delete(rc)
		return fmt.Errorf("peer put l: %w", err)
	}

	return nil
}

func (s *Sockmap) Delete(lc, rc uint64) {
	_ = s.Objs.Peer.Delete(lc)
	_ = s.Objs.Peer.Delete(rc)
	_ = s.Objs.Sockhash.Delete(lc)
	_ = s.Objs.Sockhash.Delete(rc)
}

// attachSockmapLink uses BPF_LINK_CREATE (kernel 6.10+).
func attachSockmapLink(objs *sockmapObjects) (*Sockmap, error) {
	parserLink, err := link.AttachRawLink(link.RawLinkOptions{
		Target:  objs.Sockhash.FD(),
		Program: objs.StreamParser,
		Attach:  ebpf.AttachSkSKBStreamParser,
	})
	if err != nil {
		return nil, fmt.Errorf("attach parser: %w", err)
	}

	verdictLink, err := link.AttachRawLink(link.RawLinkOptions{
		Target:  objs.Sockhash.FD(),
		Program: objs.StreamVerdict,
		Attach:  ebpf.AttachSkSKBStreamVerdict,
	})
	if err != nil {
		parserLink.Close()
		return nil, fmt.Errorf("attach verdict: %w", err)
	}

	return &Sockmap{
		Objs:        objs,
		ParserLink:  parserLink,
		VerdictLink: verdictLink,
	}, nil
}

// attachSockmapProgAttach uses BPF_PROG_ATTACH (works on older kernels).
func attachSockmapProgAttach(objs *sockmapObjects) (*Sockmap, error) {
	if err := link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  objs.Sockhash.FD(),
		Program: objs.StreamParser,
		Attach:  ebpf.AttachSkSKBStreamParser,
	}); err != nil {
		return nil, fmt.Errorf("prog_attach parser: %w", err)
	}

	if err := link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  objs.Sockhash.FD(),
		Program: objs.StreamVerdict,
		Attach:  ebpf.AttachSkSKBStreamVerdict,
	}); err != nil {
		// Best-effort detach parser on failure.
		_ = link.RawDetachProgram(link.RawDetachProgramOptions{
			Target:  objs.Sockhash.FD(),
			Program: objs.StreamParser,
			Attach:  ebpf.AttachSkSKBStreamParser,
		})
		return nil, fmt.Errorf("prog_attach verdict: %w", err)
	}

	return &Sockmap{
		Objs:      objs,
		rawAttach: true,
	}, nil
}
