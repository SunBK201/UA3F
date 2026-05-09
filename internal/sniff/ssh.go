package sniff

import "bufio"

func SniffSSH(reader *bufio.Reader) (bool, error) {
	header, err := reader.Peek(4)
	if err != nil {
		return false, err
	}
	if string(header) == "SSH-" {
		return true, nil
	}
	return false, nil
}
