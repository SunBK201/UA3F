//go:build linux

package utils

import (
	"net"
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const SO_MARK = 0xc9

// ConnectWithMark dials the target address with SO_MARK set and returns the connection.
func ConnectWithMark(addr string, mark int) (target net.Conn, err error) {
	logrus.Debugf("Connecting %s with SO_MARK=%d", addr, mark)

	dialer := net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, mark)
			})
		},
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Connected %s with SO_MARK=%d", addr, mark)
	return conn, nil
}
