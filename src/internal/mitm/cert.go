package mitm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"
)

// CertManager dynamically generates and caches TLS certificates
// signed by the root CA for each intercepted hostname.
type CertManager struct {
	ca    *CA
	cache sync.Map // map[string]*tls.Certificate
}

// NewCertManager creates a new certificate manager backed by the given CA.
func NewCertManager(ca *CA) *CertManager {
	return &CertManager{
		ca: ca,
	}
}

// GetCertificate returns a TLS certificate for the given hostname,
// generating one on-the-fly if it's not already cached.
// This satisfies tls.Config.GetCertificate.
func (cm *CertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "localhost"
	}
	return cm.GetCertificateForHost(host)
}

// GetCertificateForHost returns a TLS certificate for the given hostname.
func (cm *CertManager) GetCertificateForHost(host string) (*tls.Certificate, error) {
	if cached, ok := cm.cache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}

	cert, err := cm.generateCert(host)
	if err != nil {
		return nil, err
	}

	cm.cache.Store(host, cert)
	return cert, nil
}

// generateCert creates a new leaf certificate for the given host, signed by the root CA.
func (cm *CertManager) generateCert(host string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate leaf key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"UA3F MitM"},
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	// Set SAN
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		template,
		cm.ca.Certificate, // parent
		&key.PublicKey,
		cm.ca.PrivateKey, // signer (crypto.Signer)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaf certificate: %w", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER, cm.ca.Certificate.Raw},
		PrivateKey:  key,
	}

	return tlsCert, nil
}
