package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunbk201/ua3f/internal/config"
	"github.com/sunbk201/ua3f/internal/rewrite"
	"github.com/sunbk201/ua3f/internal/statistics"
)

type echoServer struct {
	listener net.Listener
	server   *http.Server
	addr     string
}

func NewEchoServer(t *testing.T) *echoServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create echo server listener: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/echo-ua", func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ua))
	})

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for name, values := range r.Header {
			for _, v := range values {
				_, _ = fmt.Fprintf(w, "%s: %s\n", name, v)
			}
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{Handler: mux}

	es := &echoServer{
		listener: listener,
		server:   server,
		addr:     listener.Addr().String(),
	}

	go func() {
		_ = server.Serve(listener)
	}()

	return es
}

func (es *echoServer) close() {
	_ = es.server.Close()
	_ = es.listener.Close()
}

func (es *echoServer) URL(path string) string {
	return fmt.Sprintf("http://%s%s", es.addr, path)
}

// mockRecorder creates a minimal recorder for testing
func mockRecorder() *statistics.Recorder {
	return &statistics.Recorder{
		RewriteRecordList:     statistics.NewRewriteRecordList("/dev/null"),
		PassThroughRecordList: statistics.NewPassThroughRecordList("/dev/null"),
		ConnectionRecordList:  statistics.NewConnectionRecordList("/dev/null"),
	}
}

func TestHTTPProxyUserAgentRewrite(t *testing.T) {
	// Start echo server
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	tests := []struct {
		name          string
		rewriteMode   config.RewriteMode
		targetUA      string
		originalUA    string
		expectedUA    string
		expectRewrite bool
	}{
		{
			name:          "Global mode rewrites UA",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "UA3F-Test-Agent",
			originalUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedUA:    "UA3F-Test-Agent",
			expectRewrite: true,
		},
		{
			name:          "Global mode rewrites custom UA",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "CustomUA",
			originalUA:    "MyCustomBrowser/1.0",
			expectedUA:    "CustomUA",
			expectRewrite: true,
		},
		{
			name:          "Direct mode passes UA through",
			rewriteMode:   config.RewriteModeDirect,
			targetUA:      "ShouldNotAppear",
			originalUA:    "OriginalUserAgent/1.0",
			expectedUA:    "OriginalUserAgent/1.0",
			expectRewrite: false,
		},
		{
			name:          "Whitelist UA Go-http-client is not rewritten",
			rewriteMode:   config.RewriteModeGlobal,
			targetUA:      "ShouldNotAppear",
			originalUA:    "Go-http-client/1.1",
			expectedUA:    "Go-http-client/1.1",
			expectRewrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ServerMode:  config.ServerModeHTTP,
				BindAddress: "127.0.0.1",
				Port:        0, // Will be assigned dynamically
				LogLevel:    "error",
				RewriteMode: tt.rewriteMode,
				UserAgent:   tt.targetUA,
			}

			recorder := mockRecorder()
			rw, err := rewrite.New(cfg, recorder)
			if err != nil {
				t.Fatalf("failed to create rewriter: %v", err)
			}

			// Find an available port
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("failed to find available port: %v", err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()

			cfg.Port = port
			server := New(cfg, rw, recorder, nil)

			if err := server.Start(); err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer func() { _ = server.Close() }()

			// Wait for server to be ready
			time.Sleep(100 * time.Millisecond)

			proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
			client := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 5 * time.Second,
			}

			req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if tt.originalUA != "" {
				req.Header.Set("User-Agent", tt.originalUA)
			} else {
				req.Header.Del("User-Agent")
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to send request through proxy: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			receivedUA := string(body)
			if receivedUA != tt.expectedUA {
				t.Errorf("User-Agent mismatch: got %q, want %q", receivedUA, tt.expectedUA)
			}
		})
	}
}

func TestHTTPProxyCONNECT(t *testing.T) {
	// Start echo server
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeHTTP,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test CONNECT method (used for HTTPS tunneling)
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("failed to connect to proxy: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send CONNECT request
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", echoSrv.addr, echoSrv.addr)
	_, err = conn.Write([]byte(connectReq))
	if err != nil {
		t.Fatalf("failed to send CONNECT request: %v", err)
	}

	// Read response
	buf := make([]byte, 1024)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read CONNECT response: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "200 Connection Established") {
		t.Errorf("expected '200 Connection Established', got: %s", response)
	}

	// Now send HTTP request through the tunnel
	httpReq := "GET / HTTP/1.1\r\nHost: " + echoSrv.addr + "\r\n\r\n"
	_, err = conn.Write([]byte(httpReq))
	if err != nil {
		t.Fatalf("failed to send HTTP request through tunnel: %v", err)
	}

	// Read HTTP response
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read HTTP response: %v", err)
	}

	httpResponse := string(buf[:n])
	if !strings.Contains(httpResponse, "200 OK") && !strings.Contains(httpResponse, "HTTP/1.1 200") {
		t.Errorf("expected HTTP 200 response, got: %s", httpResponse)
	}
}

func TestHTTPProxyConcurrentRequests(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeHTTP,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "ConcurrentTestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			MaxIdleConnsPerHost: 10,
		},
		Timeout: 10 * time.Second,
	}

	const numRequests = 20
	var wg sync.WaitGroup
	results := make(chan string, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to create request: %v", id, err)
				return
			}
			req.Header.Set("User-Agent", fmt.Sprintf("OriginalUA-%d", id))

			resp, err := client.Do(req)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to send request: %v", id, err)
				return
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- fmt.Errorf("request %d: failed to read response: %v", id, err)
				return
			}

			results <- string(body)
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all responses have the rewritten UA
	for result := range results {
		if result != "ConcurrentTestUA" {
			t.Errorf("expected 'ConcurrentTestUA', got %q", result)
		}
	}
}

func TestHTTPProxyDifferentMethods(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:  config.ServerModeHTTP,
		BindAddress: "127.0.0.1",
		Port:        0,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "MethodTestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var body io.Reader
			if method == "POST" || method == "PUT" || method == "PATCH" {
				body = strings.NewReader("test body")
			}

			req, err := http.NewRequest(method, echoSrv.URL("/echo-ua"), body)
			if err != nil {
				t.Fatalf("failed to create %s request: %v", method, err)
			}
			req.Header.Set("User-Agent", "OriginalUA")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to send %s request: %v", method, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if method == "HEAD" {
				// HEAD request doesn't have a body
				return
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			receivedUA := string(respBody)
			if receivedUA != "MethodTestUA" {
				t.Errorf("%s: expected 'MethodTestUA', got %q", method, receivedUA)
			}
		})
	}
}

func TestHTTPProxyPartialRewrite(t *testing.T) {
	echoSrv := NewEchoServer(t)
	defer echoSrv.close()

	cfg := &config.Config{
		ServerMode:              config.ServerModeHTTP,
		BindAddress:             "127.0.0.1",
		Port:                    0,
		LogLevel:                "error",
		RewriteMode:             config.RewriteModeGlobal,
		UserAgent:               "ReplacedPart",
		UserAgentRegex:          `Chrome/\d+`,
		UserAgentPartialReplace: true,
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg.Port = port
	server := New(cfg, rw, recorder, nil)

	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Close() }()

	time.Sleep(100 * time.Millisecond)

	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", echoSrv.URL("/echo-ua"), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	// Original UA contains "Chrome/120" which should be replaced
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	receivedUA := string(body)
	expectedUA := "Mozilla/5.0 ReplacedPart Safari/537.36"
	if receivedUA != expectedUA {
		t.Errorf("partial rewrite mismatch: got %q, want %q", receivedUA, expectedUA)
	}
}

func TestServerNew(t *testing.T) {
	cfg := &config.Config{
		ServerMode:  config.ServerModeHTTP,
		BindAddress: "127.0.0.1",
		Port:        8080,
		LogLevel:    "error",
		RewriteMode: config.RewriteModeGlobal,
		UserAgent:   "TestUA",
	}

	recorder := mockRecorder()
	rw, err := rewrite.New(cfg, recorder)
	if err != nil {
		t.Fatalf("failed to create rewriter: %v", err)
	}

	server := New(cfg, rw, recorder, nil)

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.Cfg != cfg {
		t.Error("config not set correctly")
	}
	if server.Rewriter != rw {
		t.Error("rewriter not set correctly")
	}
	if server.Recorder != recorder {
		t.Error("recorder not set correctly")
	}
	if server.Cache == nil {
		t.Error("cache not initialized")
	}
}
