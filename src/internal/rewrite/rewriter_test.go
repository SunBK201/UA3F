package rewrite

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sunbk201/ua3f/internal/config"
)

type mockConn struct {
	io.Reader
	io.Writer
}

func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func newTestRewriter(t *testing.T) *Rewriter {
	cfg := &config.Config{
		UAPattern:            "TestUA",
		PayloadUA:            "MockUA/1.0",
		EnablePartialReplace: false,
	}
	rewriter, err := New(cfg)
	assert.NoError(t, err)
	return rewriter
}

func TestNewRewriter(t *testing.T) {
	cfg := &config.Config{
		UAPattern:            "TestUA",
		PayloadUA:            "FFF0",
		EnablePartialReplace: false,
	}
	rewriter, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.PayloadUA, rewriter.payloadUA)
	assert.Equal(t, cfg.UAPattern, rewriter.pattern)
	assert.Equal(t, cfg.EnablePartialReplace, rewriter.enablePartialReplace)
	assert.NotNil(t, rewriter.uaRegex)
	assert.NotNil(t, rewriter.cache)
}

func TestIsHTTP(t *testing.T) {
	r := newTestRewriter(t)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"HTTP Get", "GET / HTTP/1.1\r\n", true},
		{"HTTP Post", "POST /test HTTP/1.1\r\n", true},
		{"HTTP Connect", "CONNECT example.com:443 HTTP/1.1\r\n", true},
		{"Not HTTP", "HELLO WORLD\r\n", false},
		{"Not HTTP", "SSH-2.0-OpenSSH_8.4\r\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			isHTTP, _ := r.isHTTP(reader)
			assert.Equal(t, tt.expected, isHTTP)
		})
	}
}

func TestProxyHTTPOrRaw_HTTPRewrite(t *testing.T) {
	r := newTestRewriter(t)

	reqStr := "GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: MyTestUA\r\n\r\n"
	src := &mockConn{Reader: strings.NewReader(reqStr), Writer: &bytes.Buffer{}}
	dstBuf := &bytes.Buffer{}
	dst := &mockConn{Reader: nil, Writer: dstBuf}

	r.ProxyHTTPOrRaw(dst, src, "example.com:80")

	out := dstBuf.String()
	assert.Contains(t, out, "User-Agent: MockUA/1.0")
}
