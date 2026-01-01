package common

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type Metadata struct {
	ConnLink *ConnLink
	Request  *http.Request
	Response *http.Response

	Packet *Packet // NFQUEUE

	srcAddr  string
	destAddr string
}

func (m *Metadata) UpdateRequest(req *http.Request) {
	m.Request = req
	m.destAddr = ""
}

func (m *Metadata) UpdateResponse(resp *http.Response) {
	m.Response = resp
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

func (m *Metadata) DestPort() string {
	if m.ConnLink != nil {
		return m.ConnLink.RPort()
	}
	if m.Request != nil {
		port := m.Request.URL.Port()
		if port == "" {
			if m.Request.URL.Scheme == "https" {
				return "443"
			}
			return "80"
		}
	}
	return ""
}

func (m *Metadata) DestAddr() string {
	if m.destAddr != "" {
		return m.destAddr
	}
	if m.Request != nil {
		m.destAddr = m.Request.Host
		if len(m.destAddr) == 0 && m.ConnLink != nil {
			m.destAddr = m.ConnLink.RAddr
		}
		if strings.IndexByte(m.destAddr, ':') == -1 {
			m.destAddr = net.JoinHostPort(m.destAddr, m.DestPort())
		}
	}
	if m.Packet != nil {
		return m.Packet.DstAddr
	}
	return m.destAddr
}

func (m *Metadata) Host() string {
	if m.Request == nil {
		return ""
	}
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
	if m.Request == nil {
		return ""
	}
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
