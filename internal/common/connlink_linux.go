//go:build linux

package common

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux UAPI: include/uapi/asm-generic/socket.h
const soCookie = 57

func (c *ConnLink) LSOCookie() (uint64, error) {
	if c.lcookie != 0 {
		return c.lcookie, nil
	}
	return socookie(c.LConn)
}

func (c *ConnLink) RSOCookie() (uint64, error) {
	if c.rcookie != 0 {
		return c.rcookie, nil
	}
	return socookie(c.RConn)
}

// socookie returns Linux SO_COOKIE (u64) for the underlying socket of conn.
func socookie(conn net.Conn) (uint64, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return 0, fmt.Errorf("conn type %T does not implement syscall.Conn", conn)
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return 0, err
	}

	var (
		cookie  uint64
		sockErr error
	)

	if err := rc.Control(func(fd uintptr) {
		cookie, sockErr = getsockoptU64(int(fd), unix.SOL_SOCKET, soCookie)
	}); err != nil {
		return 0, err
	}
	if sockErr != nil {
		return 0, sockErr
	}
	return cookie, nil
}

func getsockoptU64(fd, level, opt int) (uint64, error) {
	var v uint64
	l := uint32(unsafe.Sizeof(v)) // socklen_t 32-bit

	_, _, errno := unix.Syscall6(
		unix.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(level),
		uintptr(opt),
		uintptr(unsafe.Pointer(&v)),
		uintptr(unsafe.Pointer(&l)),
		0,
	)
	if errno != 0 {
		return 0, errno
	}
	if l != uint32(unsafe.Sizeof(v)) {
		return 0, unix.EINVAL
	}
	return v, nil
}
