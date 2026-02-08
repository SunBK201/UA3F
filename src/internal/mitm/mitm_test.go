package mitm

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestGenerateCA(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}
	if ca.Certificate == nil {
		t.Fatal("CA certificate is nil")
	}
	if ca.PrivateKey == nil {
		t.Fatal("CA private key is nil")
	}
	if !ca.Certificate.IsCA {
		t.Fatal("CA certificate IsCA should be true")
	}
	if ca.Certificate.Subject.CommonName != "UA3F Generated Root CA" {
		t.Fatalf("unexpected CA CN: %s", ca.Certificate.Subject.CommonName)
	}
}

func TestCertManager_GetCertificateForHost(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	cm := NewCertManager(ca)

	// Generate a certificate for example.com
	cert, err := cm.GetCertificateForHost("example.com")
	if err != nil {
		t.Fatalf("GetCertificateForHost failed: %v", err)
	}
	if cert == nil {
		t.Fatal("certificate is nil")
	}
	if len(cert.Certificate) != 2 {
		t.Fatalf("expected 2 certs in chain (leaf + CA), got %d", len(cert.Certificate))
	}

	// Parse the leaf certificate
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate failed: %v", err)
	}
	if leaf.Subject.CommonName != "example.com" {
		t.Fatalf("unexpected leaf CN: %s", leaf.Subject.CommonName)
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "example.com" {
		t.Fatalf("unexpected leaf DNSNames: %v", leaf.DNSNames)
	}

	// Verify the leaf is signed by our CA
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Fatalf("leaf certificate verification failed: %v", err)
	}

	// Second call should return the same cached certificate
	cert2, err := cm.GetCertificateForHost("example.com")
	if err != nil {
		t.Fatalf("GetCertificateForHost (cached) failed: %v", err)
	}
	if cert != cert2 {
		t.Fatal("expected cached certificate to be the same pointer")
	}
}

func TestCertManager_GetCertificate_TLSClientHello(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	cm := NewCertManager(ca)

	cert, err := cm.GetCertificate(&tls.ClientHelloInfo{
		ServerName: "test.example.org",
	})
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate failed: %v", err)
	}
	if leaf.Subject.CommonName != "test.example.org" {
		t.Fatalf("unexpected CN: %s", leaf.Subject.CommonName)
	}
}

func TestCertManager_IPAddress(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	cm := NewCertManager(ca)

	cert, err := cm.GetCertificateForHost("192.168.1.1")
	if err != nil {
		t.Fatalf("GetCertificateForHost (IP) failed: %v", err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate failed: %v", err)
	}
	if len(leaf.IPAddresses) != 1 || leaf.IPAddresses[0].String() != "192.168.1.1" {
		t.Fatalf("unexpected IP SANs: %v", leaf.IPAddresses)
	}
}

func TestCACertPEM(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	pem := ca.CertPEM()
	if len(pem) == 0 {
		t.Fatal("CertPEM returned empty bytes")
	}
	if string(pem[:27]) != "-----BEGIN CERTIFICATE-----" {
		t.Fatalf("unexpected PEM header: %s", string(pem[:27]))
	}
}

func TestEncodeAndDecodeP12(t *testing.T) {
	ca, err := GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	passphrase := "test-passphrase"

	p12Base64, err := ca.EncodeP12(passphrase)
	if err != nil {
		t.Fatalf("EncodeP12 failed: %v", err)
	}
	if p12Base64 == "" {
		t.Fatal("EncodeP12 returned empty string")
	}

	loaded, err := DecodeP12(p12Base64, passphrase)
	if err != nil {
		t.Fatalf("DecodeP12 failed: %v", err)
	}

	if loaded.Certificate.Subject.CommonName != ca.Certificate.Subject.CommonName {
		t.Fatal("loaded CA CN mismatch")
	}
}
