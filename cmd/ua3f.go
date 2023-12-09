package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"ua3f/http"
	"ua3f/log"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
)

var version = "0.1.1"
var payloadByte []byte
var cache *expirable.LRU[string, string]

func main() {

	var payload string
	var addr string
	var port int
	var loglevel string

	flag.StringVar(&addr, "b", "127.0.0.1", "bind address (default: 127.0.0.1)")
	flag.IntVar(&port, "p", 1080, "port")
	flag.StringVar(&payload, "f", "FFF", "User-Agent")
	flag.StringVar(&loglevel, "l", "info", "Log level (default: info)")
	flag.Parse()

	log.SetLogConf(loglevel)

	logrus.Info("UA3F v" + version)
	logrus.Info(fmt.Sprintf("Port: %d", port))
	logrus.Info(fmt.Sprintf("User-Agent: %s", payload))
	logrus.Info(fmt.Sprintf("Log level: %s", loglevel))

	cache = expirable.NewLRU[string, string](100, nil, time.Second*600)

	payloadByte = []byte(payload)

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		logrus.Fatal("Listen failed: ", err)
		return
	}
	logrus.Info(fmt.Sprintf("Listen on %s:%d", addr, port))
	for {
		client, err := server.Accept()
		if err != nil {
			logrus.Error("Accept failed: ", err)
			continue
		}
		logrus.Debug(fmt.Sprintf("Accept %s", client.RemoteAddr().String()))
		go process(client)
	}
}

func process(client net.Conn) {
	if err := Socks5Auth(client); err != nil {
		logrus.Error("Auth failed: ", err)
		client.Close()
		return
	}
	target, err := Socks5Connect(client)
	if err != nil {
		logrus.Error("Connect failed: ", err)
		client.Close()
		return
	}
	Socks5Forward(client, target)
}

func Socks5Auth(client net.Conn) (err error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(client, buf[:2])
	if n != 2 {
		return errors.New("reading header:" + err.Error())
	}
	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("invalid version")
	}
	n, err = io.ReadFull(client, buf[:nMethods])
	if n != nMethods {
		return errors.New("reading methods:" + err.Error())
	}
	n, err = client.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write rsp:" + err.Error())
	}
	return nil
}

func Socks5Connect(client net.Conn) (net.Conn, error) {
	buf := make([]byte, 256)
	n, err := io.ReadFull(client, buf[:4])
	if n != 4 {
		return nil, errors.New("read header:" + err.Error())
	}
	ver, cmd, _, atyp := buf[0], buf[1], buf[2], buf[3]
	if ver != 5 || cmd != 1 {
		return nil, errors.New("invalid ver/cmd")
	}
	addr := ""
	switch atyp {
	case 1:
		n, err = io.ReadFull(client, buf[:4])
		if n != 4 {
			return nil, errors.New("invalid IPv4:" + err.Error())
		}
		addr = fmt.Sprintf("%d.%d.%d.%d", buf[0], buf[1], buf[2], buf[3])
	case 3:
		n, err = io.ReadFull(client, buf[:1])
		if n != 1 {
			return nil, errors.New("invalid hostname:" + err.Error())
		}
		addrLen := int(buf[0])
		n, err = io.ReadFull(client, buf[:addrLen])
		if n != addrLen {
			return nil, errors.New("invalid hostname:" + err.Error())
		}
		addr = string(buf[:addrLen])
	case 4:
		return nil, errors.New("IPv6: no supported yet")
	default:
		return nil, errors.New("invalid atyp")
	}
	n, err = io.ReadFull(client, buf[:2])
	if n != 2 {
		return nil, errors.New("read port:" + err.Error())
	}
	port := binary.BigEndian.Uint16(buf[:2])
	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	logrus.Debug(fmt.Sprintf("Connecting %s", destAddrPort))
	dest, err := net.Dial("tcp", destAddrPort)
	if err != nil {
		return nil, errors.New("dial dst:" + err.Error())
	}
	logrus.Debug(fmt.Sprintf("Connected %s", destAddrPort))
	_, err = client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		dest.Close()
		return nil, errors.New("write rsp:" + err.Error())
	}
	return dest, nil
}

func Socks5Forward(client, target net.Conn) {
	forward := func(src, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		io.Copy(src, dest)
	}

	gforward := func(dst, src net.Conn) {
		defer dst.Close()
		defer src.Close()
		CopyPileline(dst, src)
	}

	go forward(client, target)
	if cache.Contains(string(target.RemoteAddr().String())) {
		go forward(target, client)
		return
	}
	go gforward(target, client)
}

func CopyPileline(dst io.Writer, src io.Reader) {
	buf := make([]byte, 1024*8)
	nr, err := src.Read(buf)
	if err != nil && err != io.EOF {
		logrus.Error("read error: ", err)
		return
	}
	hint := string(buf[0:7])
	HTTP_METHOD := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE", "CONNECT"}
	is_http := false
	for _, v := range HTTP_METHOD {
		if strings.HasPrefix(hint, v) {
			is_http = true
			break
		}
	}
	if !is_http {
		dst.Write(buf[0:nr])
		io.Copy(dst, src)
		cache.Add(string(dst.(*net.TCPConn).RemoteAddr().String()), string(dst.(*net.TCPConn).RemoteAddr().String()))
		return
	}
	for {
		parser := http.NewHTTPParser()
		httpBodyOffset, err := parser.Parse(buf[0:nr])
		for err == http.ErrMissingData {
			var m int
			m, err = src.Read(buf[nr:])
			if err != nil {
				logrus.Error("read error in http accumulation: ", err)
				break
			}
			nr += m
			httpBodyOffset, err = parser.Parse(buf[:nr])
		}
		value, start, end := parser.FindHeader([]byte("User-Agent"))
		if value != nil && end > start {
			logrus.Debug(fmt.Sprintf("[%s] Hit User-Agent: %s", string(parser.Host()), string(value)))
			for i := start; i < end; i++ {
				buf[i] = 32
			}
			for i := range payloadByte {
				if start+i >= end {
					break
				}
				buf[start+i] = payloadByte[i]
			}
		} else {
			logrus.Debug(fmt.Sprintf("[%s] Not found User-Agent", string(parser.Host())))
			dst.Write(buf[0:nr])
			io.Copy(dst, src)
			cache.Add(string(dst.(*net.TCPConn).RemoteAddr().String()), string(dst.(*net.TCPConn).RemoteAddr().String()))
			return
		}
		bodyLen := int(parser.ContentLength())
		if bodyLen == -1 {
			bodyLen = 0
		}

		_, ew := dst.Write(buf[0:min(httpBodyOffset+bodyLen, nr)])
		if ew != nil {
			logrus.Error("write error: ", ew)
			break
		}
		if httpBodyOffset+bodyLen > nr {
			left := httpBodyOffset + bodyLen - nr
			for left > 0 {
				m, err := src.Read(buf[0:left])
				if err != nil {
					logrus.Error("read error in large body: ", err)
					break
				}
				_, ew := dst.Write(buf[0:m])
				if ew != nil {
					logrus.Error("write error in large body: ", ew)
					break
				}
				left -= m
			}
			nr = 0
		} else if httpBodyOffset+bodyLen < nr {
			copy(buf[0:], buf[httpBodyOffset+bodyLen:])
			nr = nr - httpBodyOffset - bodyLen
		} else {
			nr = 0
		}

		m, err := src.Read(buf[nr:])
		nr += m
		if err != nil {
			if err == io.EOF {
				logrus.Debug("read EOF in next phase")
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				logrus.Debug("read closed in next phase: ", err)
			} else {
				logrus.Error("read error in next phase: ", err)
			}
			break
		}
	}
}
