package rewrite

import (
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
		UARegex:        "TestUA",
		PayloadUA:      "MockUA/1.0",
		PartialReplace: false,
	}
	rewriter, err := New(cfg)
	assert.NoError(t, err)
	return rewriter
}

func TestNewRewriter(t *testing.T) {
	cfg := &config.Config{
		UARegex:        "TestUA",
		PayloadUA:      "FFF0",
		PartialReplace: false,
	}
	rewriter, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.PayloadUA, rewriter.payloadUA)
	assert.Equal(t, cfg.UARegex, rewriter.pattern)
	assert.Equal(t, cfg.PartialReplace, rewriter.partialReplace)
	assert.NotNil(t, rewriter.uaRegex)
	assert.NotNil(t, rewriter.Cache)
}

func TestProxyHTTPOrRaw_HTTPRewrite(t *testing.T) {
	r := newTestRewriter(t)

	reqStr := "GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: MyTestUA\r\n\r\n"
	src := &mockConn{Reader: strings.NewReader(reqStr), Writer: &bytes.Buffer{}}
	dstBuf := &bytes.Buffer{}
	dst := &mockConn{Reader: nil, Writer: dstBuf}

	err := r.Process(dst, src, "example.com:80", "srcAddr")
	assert.NoError(t, err)

	out := dstBuf.String()
	assert.Contains(t, out, "User-Agent: MockUA/1.0")
}
