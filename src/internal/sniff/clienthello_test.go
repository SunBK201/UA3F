package sniff

import (
	"bufio"
	"bytes"
	"testing"
)

// buildClientHello builds a minimal TLS ClientHello with the given SNI.
func buildClientHello(sni string) []byte {
	// Build SNI extension
	sniBytes := []byte(sni)
	// Server name list entry: type(1) + length(2) + name
	sniEntry := make([]byte, 0, 3+len(sniBytes))
	sniEntry = append(sniEntry, 0x00)                                        // host_name type
	sniEntry = append(sniEntry, byte(len(sniBytes)>>8), byte(len(sniBytes))) // name length
	sniEntry = append(sniEntry, sniBytes...)

	// Server name list: length(2) + entries
	sniList := make([]byte, 0, 2+len(sniEntry))
	sniList = append(sniList, byte(len(sniEntry)>>8), byte(len(sniEntry)))
	sniList = append(sniList, sniEntry...)

	// SNI extension: type(2) + length(2) + data
	sniExt := make([]byte, 0, 4+len(sniList))
	sniExt = append(sniExt, 0x00, 0x00) // SNI extension type
	sniExt = append(sniExt, byte(len(sniList)>>8), byte(len(sniList)))
	sniExt = append(sniExt, sniList...)

	// Extensions: length(2) + extensions
	extensions := make([]byte, 0, 2+len(sniExt))
	extensions = append(extensions, byte(len(sniExt)>>8), byte(len(sniExt)))
	extensions = append(extensions, sniExt...)

	// ClientHello body:
	// Version(2) + Random(32) + SessionID length(1) + CipherSuites length(2) + CipherSuite(2) + Compression length(1) + Compression(1) + extensions
	body := make([]byte, 0)
	body = append(body, 0x03, 0x03)          // TLS 1.2
	body = append(body, make([]byte, 32)...) // Random
	body = append(body, 0x00)                // Session ID length
	body = append(body, 0x00, 0x02)          // Cipher suites length
	body = append(body, 0x00, 0x2f)          // TLS_RSA_WITH_AES_128_CBC_SHA
	body = append(body, 0x01)                // Compression methods length
	body = append(body, 0x00)                // null compression
	body = append(body, extensions...)

	// Handshake: type(1) + length(3) + body
	handshake := make([]byte, 0, 4+len(body))
	handshake = append(handshake, 0x01) // ClientHello
	hsLen := len(body)
	handshake = append(handshake, byte(hsLen>>16), byte(hsLen>>8), byte(hsLen))
	handshake = append(handshake, body...)

	// TLS record: type(1) + version(2) + length(2) + handshake
	record := make([]byte, 0, 5+len(handshake))
	record = append(record, 0x16)       // Handshake
	record = append(record, 0x03, 0x01) // TLS 1.0 record version
	recLen := len(handshake)
	record = append(record, byte(recLen>>8), byte(recLen))
	record = append(record, handshake...)

	return record
}

func TestSniffTLSClientHello_WithSNI(t *testing.T) {
	data := buildClientHello("example.com")
	reader := bufio.NewReaderSize(bytes.NewReader(data), len(data))

	info, err := SniffTLSClientHello(reader)
	if err != nil {
		t.Fatalf("SniffTLSClientHello failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected TLSInfo, got nil")
	}
	if info.ServerName != "example.com" {
		t.Fatalf("expected SNI 'example.com', got '%s'", info.ServerName)
	}
}

func TestSniffTLSClientHello_LongSNI(t *testing.T) {
	data := buildClientHello("very-long-subdomain.deeply.nested.example.org")
	reader := bufio.NewReaderSize(bytes.NewReader(data), len(data))

	info, err := SniffTLSClientHello(reader)
	if err != nil {
		t.Fatalf("SniffTLSClientHello failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected TLSInfo, got nil")
	}
	if info.ServerName != "very-long-subdomain.deeply.nested.example.org" {
		t.Fatalf("unexpected SNI: %s", info.ServerName)
	}
}

func TestSniffTLSClientHello_NonTLS(t *testing.T) {
	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	reader := bufio.NewReaderSize(bytes.NewReader(data), len(data))

	info, err := SniffTLSClientHello(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Fatalf("expected nil for non-TLS data, got: %v", info)
	}
}

func TestSniffTLSClientHello_NoSNI(t *testing.T) {
	// Build a TLS record without SNI extension
	// ClientHello with no extensions
	body := make([]byte, 0)
	body = append(body, 0x03, 0x03)          // TLS 1.2
	body = append(body, make([]byte, 32)...) // Random
	body = append(body, 0x00)                // Session ID length
	body = append(body, 0x00, 0x02)          // Cipher suites length
	body = append(body, 0x00, 0x2f)          // TLS_RSA_WITH_AES_128_CBC_SHA
	body = append(body, 0x01)                // Compression methods length
	body = append(body, 0x00)                // null compression
	// No extensions

	handshake := make([]byte, 0, 4+len(body))
	handshake = append(handshake, 0x01)
	hsLen := len(body)
	handshake = append(handshake, byte(hsLen>>16), byte(hsLen>>8), byte(hsLen))
	handshake = append(handshake, body...)

	record := make([]byte, 0, 5+len(handshake))
	record = append(record, 0x16, 0x03, 0x01)
	recLen := len(handshake)
	record = append(record, byte(recLen>>8), byte(recLen))
	record = append(record, handshake...)

	reader := bufio.NewReaderSize(bytes.NewReader(record), len(record))

	info, err := SniffTLSClientHello(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected TLSInfo (even without SNI), got nil")
	}
	if info.ServerName != "" {
		t.Fatalf("expected empty SNI, got '%s'", info.ServerName)
	}
}
