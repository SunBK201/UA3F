package common

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type Metadata struct {
	Request  *http.Request
	ConnLink *ConnLink
	Packet   *Packet // NFQUEUE

	srcAddr  string
	destAddr string
}

func (m *Metadata) UpdateRequest(req *http.Request) {
	m.Request = req
	m.destAddr = ""
}

func (m *Metadata) SrcAddr() string {
	if m.ConnLink != nil {
		return m.ConnLink.LAddr
	}
	if m.Request != nil {
		return m.Request.RemoteAddr
	}
	if m.Packet != nil {
		return m.Packet.SrcAddr
	}
	return m.srcAddr
}

func (m *Metadata) DestAddr() string {
	if m.destAddr != "" {
		return m.destAddr
	}
	if m.Request != nil {
		m.destAddr = m.Request.Host
		if len(m.destAddr) == 0 {
			m.destAddr = m.ConnLink.RAddr
		}
		if strings.IndexByte(m.destAddr, ':') == -1 {
			m.destAddr = net.JoinHostPort(m.destAddr, m.ConnLink.RPort())
		}
	}
	if m.Packet != nil {
		return m.Packet.DstAddr
	}
	return m.destAddr
}

func (m *Metadata) Host() string {
	host := m.Request.Host
	for i := 0; i < len(host); i++ {
		if host[i] == ':' {
			host = host[:i]
			break
		}
	}
	return host
}

func (m *Metadata) UserAgent() string {
	ua := m.Request.UserAgent()
	if ua == "" {
		m.Request.Header.Set("User-Agent", "")
	}
	return ua
}

func (m *Metadata) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("src_addr", m.SrcAddr()),
		slog.String("dest_addr", m.DestAddr()),
		slog.String("host", m.Host()),
		slog.String("user_agent", m.UserAgent()),
	)
}
