package mitm

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	"github.com/sunbk201/ua3f/internal/common"
	"github.com/sunbk201/ua3f/internal/config"
)

// MiddleMan performs HTTPS MitM by terminating client TLS, decrypting traffic,
// then handing the cleartext streams back to the standard processing pipeline.
type MiddleMan struct {
	CertManager        *CertManager
	HostnameFilter     *HostnameFilter
	InsecureSkipVerify bool
}

func NewMiddleMan(cfg *config.Config) (*MiddleMan, error) {
	if !cfg.MitM.Enabled {
		return nil, nil
	}

	ca, err := LoadCA(cfg.MitM.CAP12Base64, cfg.MitM.CAPassphrase)
	if err != nil {
		return nil, fmt.Errorf("MitM CA init failed: %w", err)
	}
	slog.Info("MitM enabled, CA certificate loaded")

	hostnameFilter, err := NewHostnameFilter(cfg.MitM.Hostname)
	if err != nil {
		return nil, fmt.Errorf("MitM hostname filter init failed: %w", err)
	}

	return &MiddleMan{
		CertManager:        NewCertManager(ca),
		HostnameFilter:     hostnameFilter,
		InsecureSkipVerify: cfg.MitM.InsecureSkipVerify,
	}, nil
}

// HandleTLS intercepts a TLS connection given the original ConnLink.
// clientReader is a *bufio.Reader that has already peeked the ClientHello.
// serverName is the extracted SNI hostname.
// Returns (true, nil) if MitM was performed, (false, nil) if skipped, or (false, error) on failure.
func (h *MiddleMan) HandleTLS(c *common.ConnLink, clientReader *bufio.Reader, serverName string) (bool, error) {
	destPort := c.RPort()

	// Check if this hostname:port should be MitM'd
	if !h.HostnameFilter.Allow(serverName, destPort) {
		c.LogInfof("MitM: skipping %s:%s (not in hostname list)", serverName, destPort)
		return false, nil
	}

	c.LogInfof("MitM: intercepting HTTPS to %s (SNI=%s, port=%s)", c.RAddr, serverName, destPort)

	// Generate a certificate for this host
	cert, err := h.CertManager.GetCertificateForHost(serverName)
	if err != nil {
		return false, fmt.Errorf("MitM: failed to get cert for %s: %w", serverName, err)
	}

	// Wrap the client connection with TLS (server-side handshake with client)
	// We need to use the buffered reader data since we've already peeked bytes
	clientTLS := tls.Server(newBufferedConn(c.LConn, clientReader), &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	if err := clientTLS.Handshake(); err != nil {
		return false, fmt.Errorf("MitM: client TLS handshake failed: %w", err)
	}

	c.LogInfof("MitM: client TLS handshake completed for %s", serverName)

	// Connect to the real upstream server with TLS
	serverTLS := tls.Client(c.RConn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: h.InsecureSkipVerify,
	})
	if err := serverTLS.Handshake(); err != nil {
		_ = clientTLS.Close()
		return false, fmt.Errorf("MitM: server TLS handshake failed for %s: %w", serverName, err)
	}

	c.LogInfof("MitM: server TLS handshake completed for %s", serverName)

	// Replace the ConnLink's connections in-place with the decrypted streams.
	c.LConn = clientTLS
	c.RConn = serverTLS

	return true, nil
}

// bufferedConn wraps a net.Conn with a bufio.Reader so that bytes
// already peeked (but not consumed) from the reader are included.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func newBufferedConn(conn net.Conn, reader *bufio.Reader) *bufferedConn {
	return &bufferedConn{
		Conn:   conn,
		reader: reader,
	}
}

func (bc *bufferedConn) Read(b []byte) (int, error) {
	return bc.reader.Read(b)
}
