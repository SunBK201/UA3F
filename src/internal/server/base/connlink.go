package base

import (
	"fmt"
	"io"
	"log/slog"
	"net"
)

var one = make([]byte, 1)

type ConnLink struct {
	LConn net.Conn
	RConn net.Conn
	LAddr string
	RAddr string
}

func (c *ConnLink) CopyLR() {
	defer func() {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.LConn.Close()
		}
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.RConn.Close()
		}
	}()
	_, _ = io.CopyBuffer(c.RConn, c.LConn, one)
}

func (c *ConnLink) CopyRL() {
	defer func() {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.RConn.Close()
		}
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.LConn.Close()
		}
	}()
	_, _ = io.CopyBuffer(c.LConn, c.RConn, one)
}

func (c *ConnLink) CloseLR() error {
	if c.LConn != nil {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.LConn.Close()
		}
	}
	if c.RConn != nil {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.RConn.Close()
		}
	}
	return nil
}

func (c *ConnLink) CloseRL() error {
	if c.RConn != nil {
		if tc, ok := c.RConn.(*net.TCPConn); ok {
			_ = tc.CloseRead()
		} else {
			_ = c.RConn.Close()
		}
	}
	if c.LConn != nil {
		if tc, ok := c.LConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		} else {
			_ = c.LConn.Close()
		}
	}
	return nil
}

func (c *ConnLink) Close() error {
	if c.LConn != nil {
		_ = c.LConn.Close()
	}
	if c.RConn != nil {
		_ = c.RConn.Close()
	}
	return nil
}

func (c *ConnLink) LogDebug(msg string) {
	slog.Debug(fmt.Sprintf("[%s -> %s] %s", c.LAddr, c.RAddr, msg))
}

func (c *ConnLink) LogInfo(msg string) {
	slog.Info(fmt.Sprintf("[%s -> %s] %s", c.LAddr, c.RAddr, msg))
}

func (c *ConnLink) LogWarn(msg string) {
	slog.Warn(fmt.Sprintf("[%s -> %s] %s", c.LAddr, c.RAddr, msg))
}

func (c *ConnLink) LogError(msg string) {
	slog.Error(fmt.Sprintf("[%s -> %s] %s", c.LAddr, c.RAddr, msg))
}

func (c *ConnLink) LogDebugf(format string, args ...interface{}) {
	c.LogDebug(fmt.Sprintf(format, args...))
}

func (c *ConnLink) LogInfof(format string, args ...interface{}) {
	c.LogInfo(fmt.Sprintf(format, args...))
}

func (c *ConnLink) LogWarnf(format string, args ...interface{}) {
	c.LogWarn(fmt.Sprintf(format, args...))
}

func (c *ConnLink) LogErrorf(format string, args ...interface{}) {
	c.LogError(fmt.Sprintf(format, args...))
}
