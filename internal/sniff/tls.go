package sniff

import "bufio"

func SniffTLS(reader *bufio.Reader) (bool, error) {
	header, err := reader.Peek(3)
	if err != nil {
		return false, err
	}
	// TLS record type 0x16 = Handshake
	if header[0] != 0x16 {
		return false, nil
	}
	// TLS version
	versionMajor := header[1]
	versionMinor := header[2]
	if versionMajor != 0x03 {
		return false, nil
	}
	if versionMinor < 0x01 || versionMinor > 0x04 {
		return false, nil
	}

	return true, nil
}
