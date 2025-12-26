package common

import (
	"net/http"
	"strings"
)

type Metadata struct {
	Request  *http.Request
	SrcAddr  string
	DestAddr string
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

func (m *Metadata) DestPort() string {
	port := m.Request.URL.Port()
	if port == "" {
		if strings.HasPrefix(m.Request.URL.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}
	return port
}
