package sniff

import "bufio"

func SniffWebSocket(reader *bufio.Reader) (bool, error) {
	header, err := reader.Peek(2)
	if err != nil {
		return false, err
	}

	b0 := header[0]
	b1 := header[1]

	rsv := b0 & 0x70    // RSV1-3
	opcode := b0 & 0x0F // opcode
	mask := b1 & 0x80   // MASK

	// requested frames from client to server must be masked
	if mask == 0 {
		return false, nil
	}
	// Control frames must have FIN set
	if rsv != 0 {
		return false, nil
	}
	// opcode must be in valid range
	if opcode > 0xA {
		return false, nil
	}
	// payload length
	payloadLen := b1 & 0x7F
	if payloadLen > 0 && payloadLen <= 125 {
		return true, nil
	}

	return true, nil
}
