package invoke

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tdp"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/telegram"
)

// Config controls DebugInvoker behavior.
type Config struct {
	// Output writer for log messages. Defaults to os.Stderr.
	Output io.Writer
	// Enabled controls whether logging is active.
	// When false, the invoker passes through with zero overhead.
	Enabled bool
	// LogReqs controls request payload logging.
	LogReqs bool
	// LogResps controls response payload logging.
	LogResps bool
	// UseColor enables ANSI color output.
	UseColor bool
}

// DebugInvoker is a telegram.Middleware that logs all outgoing MTProto
// invocations with method names, payloads, timing, and errors.
type DebugInvoker struct {
	cfg Config
}

// NewMiddleware creates a new telegram.Middleware from the given Config.
func NewMiddleware(cfg Config) telegram.Middleware {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
	return &DebugInvoker{cfg: cfg}
}

// Handle implements telegram.Middleware. It returns an InvokeFunc that
// wraps the next invoker with debug logging.
func (d *DebugInvoker) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		if !d.cfg.Enabled {
			return next.Invoke(ctx, input, output)
		}

		reg := GlobalRegistry()

		// Resolve method name from input type.
		methodName := resolveMethodName(input, reg)

		// Log request.
		if d.cfg.LogReqs {
			d.logRequest(methodName, input)
		}

		// Wrap the output decoder to capture the response.
	recorder := NewRecordingDecoder(output)

		// Perform the invocation with timing.
		start := time.Now()
		err := next.Invoke(ctx, input, recorder)
		elapsed := time.Since(start)

		// Log response or error.
		if err != nil {
			d.logError(methodName, err, elapsed)
		} else if d.cfg.LogResps {
		d.logResponse(methodName, output, recorder.Buf, elapsed)
		}

		return err
	}
}

// RecordingDecoder wraps a bin.Decoder and records the raw bytes
// that were written to the buffer before decoding.
type RecordingDecoder struct {
	target bin.Decoder
	Buf   []byte
}
// NewRecordingDecoder creates a new RecordingDecoder that wraps target.
func NewRecordingDecoder(target bin.Decoder) *RecordingDecoder {
	return &RecordingDecoder{target: target}
}

func (r *RecordingDecoder) Decode(b *bin.Buffer) error {
	// Capture the raw response bytes before passing to the real decoder.
	r.Buf = b.Copy()
	return r.target.Decode(b)
}

// resolveMethodName extracts the TL schema name from an input encoder.
func resolveMethodName(input bin.Encoder, reg *Registry) string {
	// Try TypeName() first (all gotd generated types have it).
	if tn, ok := input.(interface{ TypeName() string }); ok {
		return tn.TypeName()
	}

	// Fallback: try TypeID() + registry lookup.
	if tp, ok := input.(interface{ TypeID() uint32 }); ok {
		id := tp.TypeID()
		bare, _ := reg.NameByID(id)
		if bare != "" {
			return bare
		}
		return fmt.Sprintf("0x%08x", id)
	}

	return fmt.Sprintf("%T", input)
}

// logRequest writes request details to the configured output.
func (d *DebugInvoker) logRequest(method string, input bin.Encoder) {
	var body string
	if obj, ok := input.(tdp.Object); ok {
		body = tdp.Format(obj)
	} else {
		body = fmt.Sprintf("%T", input)
	}

	d.printf(ColorMethod, ">> %s", method)
	if d.cfg.LogReqs && body != "" {
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			d.printf(ColorBody, "   %s", line)
		}
	}
}

// logResponse writes response details to the configured output.
func (d *DebugInvoker) logResponse(method string, output bin.Decoder, rawResp []byte, elapsed time.Duration) {
	var body string
	if obj, ok := output.(tdp.Object); ok {
		body = tdp.Format(obj)
	} else if len(rawResp) > 0 {
		body = fmt.Sprintf("(%d bytes)", len(rawResp))
	}

	d.printf(ColorResponse, "<< %s [%s]", method, FormatDuration(elapsed))
	if d.cfg.LogResps && body != "" {
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			d.printf(ColorResponseBody, "   %s", line)
		}
	}
}

// logError writes error details to the configured output.
func (d *DebugInvoker) logError(method string, err error, elapsed time.Duration) {
	d.printf(ColorError, "!! %s [%s] error: %v", method, FormatDuration(elapsed), err)
}

// printf writes a formatted line to the output. If color is enabled,
// it applies the given color function.
func (d *DebugInvoker) printf(c *color.Color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if d.cfg.UseColor {
		msg = c.Sprint(msg)
	}
	fmt.Fprintln(d.cfg.Output, msg)
}

// FormatDuration renders a duration for human consumption.
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	}
	return d.Round(time.Millisecond).String()
}

// Color definitions for cross-package use.
var (
	ColorMethod       = color.New(color.FgCyan, color.Bold)
	ColorBody         = color.New(color.FgYellow)
	ColorResponse     = color.New(color.FgGreen, color.Bold)
	ColorResponseBody = color.New(color.FgHiGreen)
	ColorError        = color.New(color.FgRed, color.Bold)
	ColorTiming       = color.New(color.FgHiBlack)
)

// FormatHexID formats a uint32 type ID as hex string.
func FormatHexID(id uint32) string {
	return "0x" + strconv.FormatUint(uint64(id), 16)
}

// DefaultConfig returns a Config with sensible defaults for terminal use.
func DefaultConfig() Config {
	return Config{
		Output:   os.Stderr,
		Enabled:  true,
		LogReqs:  true,
		LogResps: true,
		UseColor: true,
	}
}
