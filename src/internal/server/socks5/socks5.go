package socks5

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/luyuhuang/subsocks/socks"
	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/server/base"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type Server struct {
	base.Server
	listener net.Listener
	so_mark  int
}

func New(cfg *config.Config, rw *rewrite.Rewriter, rc *statistics.Recorder) *Server {
	return &Server{
		Server: base.Server{
			Cfg:      cfg,
			Rewriter: rw,
			Recorder: rc,
			Cache:    expirable.NewLRU[string, struct{}](512, nil, 30*time.Minute),
			BufioReaderPool: sync.Pool{
				New: func() interface{} {
					return bufio.NewReaderSize(nil, 16*1024)
				},
			},
		},
		so_mark: base.SO_MARK,
	}
}

func (s *Server) Close() (err error) {
	if s.listener != nil {
		err = s.listener.Close()
	}
	return
}

func (s *Server) Start() (err error) {
	listenAddr := fmt.Sprintf("%s:%d", s.Cfg.BindAddress, s.Cfg.Port)
	if s.listener, err = net.Listen("tcp", listenAddr); err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	s.Recorder.Start()

	go func() {
		var client net.Conn
		for {
			if client, err = s.listener.Accept(); err != nil {
				if errors.Is(err, syscall.EMFILE) {
					time.Sleep(time.Second)
				} else if errors.Is(err, net.ErrClosed) {
					return
				}
				slog.Error("s.listener.Accept", slog.Any("error", err))
				continue
			}
			go s.HandleClient(client)
		}
	}()
	return nil
}

func (s *Server) HandleClient(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	srcAddr := conn.RemoteAddr().String()

	slog.Info("New socks5 connection", slog.String("srcAddr", srcAddr))

	if err := s.handShake(conn); err != nil {
		slog.Error("s.handShake", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		return
	}

	request, err := socks.ReadRequest(conn)
	if err != nil {
		slog.Error("socks.ReadRequest", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		return
	}

	switch request.Cmd {
	case socks.CmdConnect:
		err = s.handleConnect(conn, request)
		if err != nil {
			err = fmt.Errorf("s.handleConnect: %w", err)
		}
	case socks.CmdBind:
		err = s.handleBind(conn)
		if err != nil {
			err = fmt.Errorf("s.handleBind: %w", err)
		}
	case socks.CmdUDP:
		err = s.handleUDPAssociate(conn)
		if err != nil {
			err = fmt.Errorf("s.handleUDPAssociate: %w", err)
		}
	default:
		err = fmt.Errorf("socks5 unsupported command %d", request.Cmd)
	}
	if err != nil {
		slog.Error("HandleClient", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		return
	}
}

func (s *Server) handShake(conn net.Conn) error {
	methods, err := socks.ReadMethods(conn)
	if err != nil {
		return fmt.Errorf("socks.ReadMethods: %w", err)
	}
	method := socks.MethodNoAcceptable
	for _, m := range methods {
		if m == socks.MethodNoAuth {
			method = m
		}
	}
	if err := socks.WriteMethod(socks.MethodNoAuth, conn); err != nil || method == socks.MethodNoAcceptable {
		if err != nil {
			return fmt.Errorf("socks.WriteMethod: %w", err)
		} else {
			return fmt.Errorf("socks5 methods is not acceptable")
		}
	}
	return nil
}

func (s *Server) handleConnect(src net.Conn, req *socks.Request) error {
	srcAddr := src.RemoteAddr().String()
	destAddr := req.Addr.String()

	dest, err := base.Connect(destAddr, s.so_mark)
	if err != nil {
		if err := socks.NewReply(socks.HostUnreachable, nil).Write(src); err != nil {
			slog.Error("socks.NewReply.Write", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		}
		return fmt.Errorf("base.Connect: %w, dest: %s", err, destAddr)
	}

	if err := socks.NewReply(socks.Succeeded, nil).Write(src); err != nil {
		_ = dest.Close()
		return fmt.Errorf("socks.NewReply.Write: %w", err)
	}

	s.ServeConnLink(&base.ConnLink{
		LConn: src,
		RConn: dest,
		LAddr: srcAddr,
		RAddr: destAddr,
	})

	return nil
}

func (s *Server) handleBind(conn net.Conn) error {
	srcAddr := conn.RemoteAddr().String()
	listener, err := net.ListenTCP("tcp", nil)
	if err != nil {
		if err := socks.NewReply(socks.Failure, nil).Write(conn); err != nil {
			slog.Error("socks.NewReply.Write", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		}
		return fmt.Errorf("net.ListenTCP: %w", err)
	}

	addr, _ := socks.NewAddrFromAddr(listener.Addr(), conn.LocalAddr())
	if err := socks.NewReply(socks.Succeeded, addr).Write(conn); err != nil {
		_ = listener.Close()
		return fmt.Errorf("socks.NewReply.Write: %w", err)
	}

	newConn, err := listener.AcceptTCP()
	_ = listener.Close()
	if err != nil {
		if err := socks.NewReply(socks.Failure, nil).Write(conn); err != nil {
			slog.Error("socks.NewReply.Write", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		}
		return fmt.Errorf("listener.AcceptTCP: %w", err)
	}
	defer func() {
		_ = newConn.Close()
	}()

	raddr, _ := socks.NewAddr(newConn.RemoteAddr().String())
	if err := socks.NewReply(socks.Succeeded, raddr).Write(conn); err != nil {
		return fmt.Errorf("socks.NewReply.Write: %w", err)
	}

	s.ServeConnLink(&base.ConnLink{
		LConn: conn,
		RConn: newConn,
		LAddr: srcAddr,
		RAddr: newConn.RemoteAddr().String(),
	})
	return nil
}

func (s *Server) handleUDPAssociate(conn net.Conn) error {
	srcAddr := conn.RemoteAddr().String()

	udp, err := net.ListenUDP("udp", nil)
	if err != nil {
		if err := socks.NewReply(socks.Failure, nil).Write(conn); err != nil {
			slog.Error("socks.NewReply.Write", slog.String("srcAddr", srcAddr), slog.Any("error", err))
		}
		return fmt.Errorf("net.ListenUDP: %w", err)
	}

	addr, _ := socks.NewAddrFromAddr(udp.LocalAddr(), conn.LocalAddr())
	if err := socks.NewReply(socks.Succeeded, addr).Write(conn); err != nil {
		_ = udp.Close()
		return fmt.Errorf("socks.NewReply.Write: %w", err)
	}

	slog.Info("UDP associate established", slog.String("srcAddr", srcAddr), slog.String("udpAddr", udp.LocalAddr().String()))

	s.tunnelUDP(conn, udp)
	return nil
}

func (s *Server) tunnelUDP(conn net.Conn, udp *net.UDPConn) {
	srcAddr := conn.RemoteAddr().String()
	tcpRemote := conn.RemoteAddr().(*net.TCPAddr)

	var clientUDPAddr *net.UDPAddr

	done := make(chan struct{})

	go func() {
		defer func() {
			_ = udp.Close()
		}()

		b := make([]byte, 64*1024)

		for {
			select {
			case <-done:
				return
			default:
			}

			_ = udp.SetReadDeadline(time.Now().Add(time.Second * 30))
			n, addr, err := udp.ReadFrom(b)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				slog.Error("udp.ReadFrom", slog.String("srcAddr", srcAddr), slog.Any("error", err))
				continue
			}

			udpAddr, ok := addr.(*net.UDPAddr)
			if !ok {
				continue
			}

			isFromClient := udpAddr.IP.Equal(tcpRemote.IP)
			if isFromClient {
				clientUDPAddr = udpAddr

				dgram, err := socks.ReadUDPDatagram(bytes.NewReader(b[:n]))
				if err != nil {
					slog.Error("socks.ReadUDPDatagram error", slog.String("srcAddr", srcAddr), slog.Any("error", err))
					continue
				}

				destAddr, err := net.ResolveUDPAddr("udp", dgram.Header.Addr.String())
				if err != nil {
					slog.Error("net.ResolveUDPAddr error",
						slog.String("srcAddr", srcAddr),
						slog.String("destAddr", dgram.Header.Addr.String()),
						slog.Any("error", err))
					continue
				}

				if _, err := udp.WriteTo(dgram.Data, destAddr); err != nil {
					slog.Error("udp.WriteTo dest error",
						slog.String("srcAddr", srcAddr),
						slog.String("destAddr", destAddr.String()),
						slog.Any("error", err))
					continue
				}

				slog.Debug("UDP relay request",
					slog.String("from", addr.String()),
					slog.String("to", destAddr.String()),
					slog.Int("bytes", len(dgram.Data)))

			} else {
				if clientUDPAddr == nil {
					continue
				}

				saddr, _ := socks.NewAddr(addr.String())
				dgram := socks.NewUDPDatagram(
					socks.NewUDPHeader(0, 0, saddr), b[:n])

				var writer bytes.Buffer
				if err := dgram.Write(&writer); err != nil {
					slog.Debug("dgram.Write error", slog.String("srcAddr", srcAddr), slog.Any("error", err))
					continue
				}

				if _, err := udp.WriteTo(writer.Bytes(), clientUDPAddr); err != nil {
					slog.Debug("udp.WriteTo client error", slog.String("srcAddr", srcAddr), slog.Any("error", err))
					continue
				}

				slog.Debug("UDP relay response",
					slog.String("from", addr.String()),
					slog.String("to", clientUDPAddr.String()),
					slog.Int("bytes", n))
			}
		}
	}()

	// tcp connection monitor
	b := make([]byte, 1)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Minute))
		if _, err := conn.Read(b); err != nil {
			slog.Info("TCP connection closed, stopping UDP relay", slog.String("srcAddr", srcAddr), slog.String("udpAddr", udp.LocalAddr().String()))
			close(done)
			return
		}
	}
}
