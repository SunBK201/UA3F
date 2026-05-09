package mitm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

type CA struct {
	Certificate *x509.Certificate
	PrivateKey  crypto.Signer
}

func LoadCA(p12Base64, passphrase string) (*CA, error) {
	if p12Base64 == "" {
		return nil, fmt.Errorf("no PKCS#12 provided")
	}
	return DecodeP12(p12Base64, passphrase)
}

func GenerateCA() (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "UA3F Generated Root CA",
			Organization: []string{"UA3F"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return &CA{
		Certificate: cert,
		PrivateKey:  key,
	}, nil
}

// DecodeP12 decodes base64-encoded PKCS#12 data and extracts the CA certificate and private key.
func DecodeP12(p12Base64, passphrase string) (*CA, error) {
	p12Data, err := base64.StdEncoding.DecodeString(p12Base64)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode PKCS#12: %w", err)
	}

	privateKey, cert, err := pkcs12.Decode(p12Data, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PKCS#12: %w", err)
	}

	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("PKCS#12 private key does not implement crypto.Signer")
	}

	return &CA{
		Certificate: cert,
		PrivateKey:  signer,
	}, nil
}

// EncodeP12 encodes the CA certificate and private key into base64-encoded PKCS#12.
func (ca *CA) EncodeP12(passphrase string) (string, error) {
	p12Data, err := pkcs12.Modern.Encode(ca.PrivateKey, ca.Certificate, nil, passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to encode PKCS#12: %w", err)
	}
	return base64.StdEncoding.EncodeToString(p12Data), nil
}

// CertPEM returns the PEM-encoded CA certificate.
func (ca *CA) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.Certificate.Raw,
	})
}
