package sniff

import "bufio"

func SniffTLSClientHello(reader *bufio.Reader) bool {
	header, err := reader.Peek(3)
	if err != nil {
		return false
	}
	// TLS record type 0x16 = Handshake
	if header[0] != 0x16 {
		return false
	}
	// TLS version
	versionMajor := header[1]
	versionMinor := header[2]
	if versionMajor != 0x03 {
		return false
	}
	if versionMinor < 0x01 || versionMinor > 0x04 {
		return false
	}

	return true
}
