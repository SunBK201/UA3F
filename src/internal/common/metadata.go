package common

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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

func (m *Metadata) RequestBody(decode bool) []byte {
	if m.Request == nil || m.Request.Body == nil || m.Request.Body == http.NoBody {
		return nil
	}

	body, err := io.ReadAll(m.Request.Body)
	if err != nil {
		slog.Error("RequestBody io.ReadAll", "error", err)
		return nil
	}

	m.Request.Body = io.NopCloser(bytes.NewReader(body))
	m.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	b := bytes.Clone(body)

	if decode {
		encoding := m.Request.Header.Get("Content-Encoding")
		decodedBody, err := decodeBody(b, encoding)
		if err != nil {
			slog.Warn("RequestBody decodeBody", "error", err)
			return body
		}
		b = decodedBody
	}

	return b
}

func (m *Metadata) UpdateRequestBody(newBody []byte, encode bool) {
	if m.Request == nil {
		return
	}

	r := m.Request

	if encode {
		encoding := r.Header.Get("Content-Encoding")
		encodedBody, err := encodeBody(newBody, encoding)
		if err != nil {
			slog.Warn("UpdateRequestBody encodeBody", "error", err)
		} else {
			newBody = encodedBody
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(newBody))
	r.ContentLength = int64(len(newBody))

	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(newBody)), nil
	}

	r.Header.Del("Transfer-Encoding")
	r.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
}

func (m *Metadata) ResponseBody(decode bool) []byte {
	if m.Response == nil || m.Response.Body == nil || m.Response.Body == http.NoBody {
		return nil
	}

	body, err := io.ReadAll(m.Response.Body)
	if err != nil {
		slog.Error("ResponseBody io.ReadAll", "error", err)
		return nil
	}

	m.Response.Body = io.NopCloser(bytes.NewReader(body))
	m.Response.ContentLength = int64(len(body))

	b := bytes.Clone(body)

	if decode {
		encoding := m.Response.Header.Get("Content-Encoding")
		decodedBody, err := decodeBody(b, encoding)
		if err != nil {
			slog.Warn("ResponseBody decodeBody", "error", err)
			return body
		}
		b = decodedBody
	}

	return b
}

func (m *Metadata) UpdateResponseBody(newBody []byte, encode bool) {
	if m.Response == nil {
		return
	}

	r := m.Response

	if encode {
		encoding := r.Header.Get("Content-Encoding")
		encodedBody, err := encodeBody(newBody, encoding)
		if err != nil {
			slog.Warn("UpdateResponseBody encodeBody", "error", err)
		} else {
			newBody = encodedBody
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(newBody))
	r.ContentLength = int64(len(newBody))

	r.Header.Del("Transfer-Encoding")
	r.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
}

func (m *Metadata) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("src_addr", m.SrcAddr()),
		slog.String("dest_addr", m.DestAddr()),
		slog.String("host", m.Host()),
		slog.String("user_agent", m.UserAgent()),
	)
}

func decodeBody(body []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = r.Close()
		}()
		return io.ReadAll(r)

	case "", "identity":
		return body, nil
	default:
		slog.Warn("unknown encoding", "encoding", encoding)
		return body, nil
	}
}

func encodeBody(body []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "gzip":
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil

	case "", "identity":
		return body, nil
	default:
		slog.Warn("unknown encoding", "encoding", encoding)
		return body, nil
	}
}
