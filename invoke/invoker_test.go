package invoke

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/telegram"
)

// mockInvoker records calls and optionally returns an error.
type mockInvoker struct {
	lastInput bin.Encoder
	calls     int
	err       error
}

func (m *mockInvoker) Invoke(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
	m.calls++
	m.lastInput = input
	return m.err
}

func TestDebugInvoker_Disabled(t *testing.T) {
	var buf bytes.Buffer
	middleware := NewMiddleware(Config{
		Output:  &buf,
		Enabled: false,
	})

	mock := &mockInvoker{}
	invoker := middleware.Handle(mock)

	req := &tg.MessagesSendMessageRequest{
		Peer:    &tg.InputPeerUser{},
		Message: "test",
	}

	// Use a real decoder type since *bin.Buffer doesn't implement bin.Decoder.
	var result tg.UpdatesBox
	err := invoker.Invoke(context.Background(), req, &result)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}

	if buf.Len() > 0 {
		t.Errorf("disabled middleware should produce no output, got %q", buf.String())
	}
}

func TestDebugInvoker_Enabled(t *testing.T) {
	var buf bytes.Buffer
	middleware := NewMiddleware(Config{
		Output:   &buf,
		Enabled:  true,
		LogReqs:  true,
		LogResps: true,
		UseColor: false,
	})

	mock := &mockInvoker{}
	invoker := middleware.Handle(mock)

	req := &tg.MessagesSendMessageRequest{
		Message: "hello",
	}

	var result tg.UpdatesBox
	err := invoker.Invoke(context.Background(), req, &result)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "messages.sendMessage") {
		t.Errorf("output should contain method name, got: %q", output)
	}

	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestDebugInvoker_Error(t *testing.T) {
	var buf bytes.Buffer
	middleware := NewMiddleware(Config{
		Output:   &buf,
		Enabled:  true,
		LogReqs:  true,
		LogResps: true,
		UseColor: false,
	})

	mock := &mockInvoker{err: context.DeadlineExceeded}
	invoker := middleware.Handle(mock)

	req := &tg.MessagesSendMessageRequest{}

	var result tg.UpdatesBox
	err := invoker.Invoke(context.Background(), req, &result)
	if err == nil {
		t.Fatal("expected error from invoker")
	}

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Errorf("output should contain error, got: %q", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{100 * time.Nanosecond, "100ns"},
		{1500 * time.Nanosecond, "1.5µs"},
		{2500 * time.Microsecond, "2.5ms"},
		{1500 * time.Millisecond, "1.5s"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v): got %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("DefaultConfig().Enabled should be true")
	}
	if !cfg.LogReqs {
		t.Error("DefaultConfig().LogReqs should be true")
	}
	if !cfg.LogResps {
		t.Error("DefaultConfig().LogResps should be true")
	}
	if !cfg.UseColor {
		t.Error("DefaultConfig().UseColor should be true")
	}
}

func TestMiddlewareImplementsInterface(t *testing.T) {
	var _ telegram.Middleware = &DebugInvoker{}
	var _ telegram.Middleware = NewMiddleware(DefaultConfig())
}
