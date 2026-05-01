package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// maxConcurrentConns limits the number of simultaneous IPC connections.
	maxConcurrentConns = 64
	// maxRequestSize limits the size of a single IPC JSON request.
	maxRequestSize = 1 << 20 // 1 MiB
)

// InvokeRequest is the JSON message sent from client to server.
type InvokeRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	Format string          `json:"format,omitempty"`
}

// InvokeResponse is the JSON message sent from server to client.
type InvokeResponse struct {
	OK     bool   `json:"ok"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Handler processes invoke requests over IPC.
type Handler func(ctx context.Context, req InvokeRequest) InvokeResponse

// Server listens on a Unix domain socket and dispatches invoke requests
// to a Handler.
type Server struct {
	socketPath string
	handler    Handler
	listener   net.Listener
	done       chan struct{}
	closeOnce  sync.Once
	ctx        context.Context
	cancel     context.CancelFunc
	sem        chan struct{} // limits concurrent connections
}

// NewServer creates a new IPC server at the given socket path.
func NewServer(socketPath string, handler Handler) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		done:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
		sem:        make(chan struct{}, maxConcurrentConns),
	}
}

// Start begins accepting connections.
func (s *Server) Start() error {
	os.Remove(s.socketPath)

	if dir := filepath.Dir(s.socketPath); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create socket dir: %w", err)
		}
	}

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		l.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.listener = l

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		s.sem <- struct{}{}
		go func() {
			defer func() { <-s.sem }()
			s.handleConn(conn)
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	dec := json.NewDecoder(io.LimitReader(conn, maxRequestSize))
	enc := json.NewEncoder(writerOnly{conn})

	for {
		var req InvokeRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return
			}
			enc.Encode(InvokeResponse{OK: false, Error: fmt.Sprintf("decode: %v", err)})
			return
		}

		ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
		resp := s.handler(ctx, req)
		cancel()

		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

// writerOnly wraps an io.Writer to prevent json.NewEncoder from
// attempting to use other interfaces on the underlying conn.
type writerOnly struct{ io.Writer }

// Close stops the server and removes the socket file.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		s.cancel()
		close(s.done)
		if s.listener != nil {
			s.listener.Close()
		}
		os.Remove(s.socketPath)
	})
	return nil
}

// SocketPath returns the path the server is listening on.
func (s *Server) SocketPath() string {
	return s.socketPath
}

// Client connects to a running tgdev server.
type Client struct {
	socketPath string
}

// NewClient creates a client that connects to the given socket path.
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

// Invoke sends a request to the server and returns the response.
func (c *Client) Invoke(ctx context.Context, req InvokeRequest) (InvokeResponse, error) {
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return InvokeResponse{}, fmt.Errorf("connect to %s (is tgdev listen running?): %w", c.socketPath, err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(60 * time.Second))
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return InvokeResponse{}, fmt.Errorf("send: %w", err)
	}

	var resp InvokeResponse
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return InvokeResponse{}, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

// Ping checks if a server is listening at the socket path.
func Ping(socketPath string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
