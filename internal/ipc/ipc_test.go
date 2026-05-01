package ipc

import (
	"context"
	"testing"
	"time"
)

// echoHandler returns "echo:" prefixed with the request method.
func echoHandler(ctx context.Context, req InvokeRequest) InvokeResponse {
	return InvokeResponse{OK: true, Result: "echo:" + req.Method}
}

func socketPath(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/tgdev.sock"
}

func startServer(t *testing.T, socket string, handler Handler) *Server {
	t.Helper()
	s := NewServer(socket, handler)
	if err := s.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestServerClientRoundTrip(t *testing.T) {
	sock := socketPath(t)
	startServer(t, sock, echoHandler)

	client := NewClient(sock)
	resp, err := client.Invoke(context.Background(), InvokeRequest{
		Method: "hello",
		Params: nil,
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK=true, got error: %s", resp.Error)
	}
	if resp.Result != "echo:hello" {
		t.Fatalf("expected result %q, got %q", "echo:hello", resp.Result)
	}
}

func TestServerMultipleRequests(t *testing.T) {
	sock := socketPath(t)
	startServer(t, sock, echoHandler)

	client := NewClient(sock)
	methods := []string{"a", "b", "c", "d", "e"}
	for _, m := range methods {
		resp, err := client.Invoke(context.Background(), InvokeRequest{Method: m})
		if err != nil {
			t.Fatalf("invoke %q: %v", m, err)
		}
		want := "echo:" + m
		if resp.Result != want {
			t.Fatalf("method %q: expected %q, got %q", m, want, resp.Result)
		}
	}
}

func TestPing(t *testing.T) {
	t.Run("running server", func(t *testing.T) {
		sock := socketPath(t)
		startServer(t, sock, echoHandler)
		if !Ping(sock) {
			t.Fatal("expected Ping to return true for running server")
		}
	})

	t.Run("no server", func(t *testing.T) {
		sock := socketPath(t) + "/nonexistent.sock"
		if Ping(sock) {
			t.Fatal("expected Ping to return false for missing server")
		}
	})
}

func TestClientServerTimeout(t *testing.T) {
	sock := socketPath(t)
	slowHandler := func(ctx context.Context, req InvokeRequest) InvokeResponse {
		// Simulate work — but respect context cancellation.
		select {
		case <-ctx.Done():
			return InvokeResponse{OK: false, Error: ctx.Err().Error()}
		case <-time.After(5 * time.Second):
			return InvokeResponse{OK: true, Result: "slow:" + req.Method}
		}
	}
	startServer(t, sock, slowHandler)

	client := NewClient(sock)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := client.Invoke(ctx, InvokeRequest{Method: "slow"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestServerClose(t *testing.T) {
	sock := socketPath(t)
	s := NewServer(sock, echoHandler)
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Verify server is reachable before close.
	if !Ping(sock) {
		t.Fatal("server should be reachable before close")
	}

	// Close and verify unreachability.
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if Ping(sock) {
		t.Fatal("server should be unreachable after close")
	}
}
