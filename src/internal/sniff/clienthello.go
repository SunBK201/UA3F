package sniff

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// TLSInfo holds information extracted from a TLS ClientHello message.
type TLSInfo struct {
	ServerName string // SNI from the ClientHello
}

// SniffTLSClientHello peeks at the buffered reader to detect a TLS ClientHello
// and extract the SNI (Server Name Indication). The data is NOT consumed.
// Returns (nil, nil) if the data is not a TLS ClientHello.
func SniffTLSClientHello(reader *bufio.Reader) (*TLSInfo, error) {
	// We need at least 5 bytes for TLS record header
	header, err := reader.Peek(5)
	if err != nil {
		return nil, err
	}

	// TLS record type 0x16 = Handshake
	if header[0] != 0x16 {
		return nil, nil
	}
	// TLS version major
	if header[1] != 0x03 {
		return nil, nil
	}
	// TLS version minor 0x01-0x04
	if header[2] < 0x01 || header[2] > 0x04 {
		return nil, nil
	}

	// Record length (bytes 3-4)
	recordLen := int(binary.BigEndian.Uint16(header[3:5]))
	if recordLen <= 0 || recordLen > 16384 {
		return nil, nil
	}

	// Peek the entire TLS record
	totalLen := 5 + recordLen
	if totalLen > 16389 { // 5 + 16384
		totalLen = 16389
	}

	data, err := reader.Peek(totalLen)
	if err != nil {
		// If we can't peek the full record, try to parse what we have
		if err != io.EOF {
			return nil, err
		}
		data, _ = reader.Peek(reader.Buffered())
		if len(data) < 44 {
			return nil, nil
		}
	}

	sni := extractSNI(data[5:])
	if sni == "" {
		return &TLSInfo{}, nil
	}
	return &TLSInfo{ServerName: sni}, nil
}

// extractSNI parses the handshake message to find SNI extension.
func extractSNI(data []byte) string {
	if len(data) < 39 {
		return ""
	}

	// Handshake type: ClientHello = 0x01
	if data[0] != 0x01 {
		return ""
	}

	// Handshake length (3 bytes)
	hsLen := int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if hsLen > len(data)-4 {
		hsLen = len(data) - 4
	}
	data = data[4 : 4+hsLen]

	// Client version (2 bytes) + Random (32 bytes) = 34 bytes
	if len(data) < 34 {
		return ""
	}
	pos := 34

	// Session ID
	if pos >= len(data) {
		return ""
	}
	sessionIDLen := int(data[pos])
	pos += 1 + sessionIDLen
	if pos >= len(data) {
		return ""
	}

	// Cipher suites
	if pos+2 > len(data) {
		return ""
	}
	cipherSuitesLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2 + cipherSuitesLen
	if pos >= len(data) {
		return ""
	}

	// Compression methods
	if pos >= len(data) {
		return ""
	}
	compressionLen := int(data[pos])
	pos += 1 + compressionLen
	if pos >= len(data) {
		return ""
	}

	// Extensions
	if pos+2 > len(data) {
		return ""
	}
	extensionsLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2
	end := pos + extensionsLen
	if end > len(data) {
		end = len(data)
	}

	for pos+4 <= end {
		extType := binary.BigEndian.Uint16(data[pos : pos+2])
		extLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		pos += 4

		if pos+extLen > end {
			break
		}

		// SNI extension type = 0x0000
		if extType == 0x0000 {
			return parseSNIExtension(data[pos : pos+extLen])
		}

		pos += extLen
	}

	return ""
}

// parseSNIExtension parses the SNI extension data to extract the hostname.
func parseSNIExtension(data []byte) string {
	if len(data) < 5 {
		return ""
	}

	// Server name list length (2 bytes)
	listLen := int(binary.BigEndian.Uint16(data[0:2]))
	if listLen+2 > len(data) {
		listLen = len(data) - 2
	}

	pos := 2
	end := 2 + listLen

	for pos+3 <= end {
		nameType := data[pos]
		nameLen := int(binary.BigEndian.Uint16(data[pos+1 : pos+3]))
		pos += 3

		if pos+nameLen > end {
			break
		}

		// Host name type = 0
		if nameType == 0 {
			name := string(data[pos : pos+nameLen])
			if isValidHostname(name) {
				return name
			}
		}

		pos += nameLen
	}

	return ""
}

// isValidHostname performs basic hostname validation.
func isValidHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, c := range host {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// FormatTLSInfo formats TLS info for logging.
func (info *TLSInfo) String() string {
	if info == nil {
		return "TLS(nil)"
	}
	return fmt.Sprintf("TLS(SNI=%s)", info.ServerName)
}
