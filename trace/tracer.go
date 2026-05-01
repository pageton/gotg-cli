package trace

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tdp"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/telegram"
	"github.com/pageton/gotg-cli/invoke"
)

// Tracer combines debug invocation logging with update listening to provide
// full lifecycle tracing. It correlates updates → handler invocations →
// API calls → responses.
type Tracer struct {
	output io.Writer
	color  bool

	// counter generates monotonically increasing correlation IDs.
	counter atomic.Uint64
}

// Config controls Tracer behavior.
type Config struct {
	Output   io.Writer
	UseColor bool
}

// NewTracer creates a new Tracer with the given config.
func NewTracer(cfg Config) *Tracer {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
	return &Tracer{
		output: cfg.Output,
		color:  cfg.UseColor,
	}
}

// NextID returns the next correlation ID.
func (t *Tracer) NextID() uint64 {
	return t.counter.Add(1)
}

// Middleware returns a telegram.Middleware that logs all invocations
// with correlation IDs and timing.
func (t *Tracer) Middleware() telegram.Middleware {
	return telegram.MiddlewareFunc(func(next tg.Invoker) telegram.InvokeFunc {
		return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
			cid := t.NextID()
			methodName := resolveMethodName(input)

			t.Printf(invoke.ColorMethod, "[%d] >> %s", cid, methodName)

			// Log request body.
			if obj, ok := input.(tdp.Object); ok {
				body := tdp.Format(obj)
				if body != "" {
					t.Printf(invoke.ColorBody, "[%d]    %s", cid, body)
				}
			}

			// Wrap output to capture response.
			recorder := invoke.NewRecordingDecoder(output)

			start := time.Now()
			err := next.Invoke(ctx, input, recorder)
			elapsed := time.Since(start)

			if err != nil {
				t.Printf(invoke.ColorError, "[%d] !! %s [%s] %v", cid, methodName, invoke.FormatDuration(elapsed), err)
			} else {
				t.Printf(invoke.ColorResponse, "[%d] << %s [%s]", cid, methodName, invoke.FormatDuration(elapsed))
				if obj, ok := output.(tdp.Object); ok {
					body := tdp.Format(obj)
					if body != "" {
						t.Printf(invoke.ColorResponseBody, "[%d]    %s", cid, body)
					}
				}
			}

			return err
		}
	})
}

func (t *Tracer) Printf(c *color.Color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if t.color {
		msg = c.Sprint(msg)
	}
	fmt.Fprintln(t.output, msg)
}

// resolveMethodName extracts the TL schema name from an input encoder.
func resolveMethodName(input bin.Encoder) string {
	if tn, ok := input.(interface{ TypeName() string }); ok {
		return tn.TypeName()
	}
	if tp, ok := input.(interface{ TypeID() uint32 }); ok {
		reg := invoke.GlobalRegistry()
		bare, _ := reg.NameByID(tp.TypeID())
		if bare != "" {
			return bare
		}
		return fmt.Sprintf("0x%08x", tp.TypeID())
	}
	return fmt.Sprintf("%T", input)
}

// FormatDuration is re-exported from invoke for convenience.
var FormatDuration = invoke.FormatDuration
