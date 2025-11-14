package base

import (
	"errors"
	"fmt"
	"net"
)

func Connect(addr string) (target net.Conn, err error) {
	if target, err = net.Dial("tcp", addr); err != nil {
		return nil, fmt.Errorf("net.Dial: %v", err)
	}
	return target, nil
}

func GetConnFD(conn net.Conn) (fd int, err error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, errors.New("GetConnFD connection is not *net.TCPConn")
	}
	file, err := tcpConn.File()
	if err != nil {
		return 0, fmt.Errorf("tcpConn.File: %v", err)
	}

	return int(file.Fd()), nil
}
