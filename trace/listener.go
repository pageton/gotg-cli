package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/gotd/td/tg"
)

// UpdateListener handles incoming Telegram updates and logs them
// with correlation IDs that can be cross-referenced with API calls.
type UpdateListener struct {
	tracer *Tracer
}

// NewUpdateListener creates an UpdateListener that logs updates
// through the given Tracer.
func NewUpdateListener(tracer *Tracer) *UpdateListener {
	return &UpdateListener{tracer: tracer}
}

// HandleUpdate processes a single update and logs it.
// This can be used as a gotg dispatcher handler.
func (l *UpdateListener) HandleUpdate(ctx context.Context, update tg.UpdateClass) error {
	cid := l.tracer.NextID()
	l.tracer.Printf(colorUpdate, "[%d] UPDATE %s", cid, formatUpdateType(update))
	return nil
}

// HandleRawUpdate processes a raw update box.
func (l *UpdateListener) HandleRawUpdate(ctx context.Context, updates tg.UpdatesClass) error {
	cid := l.tracer.NextID()
	l.tracer.Printf(colorUpdate, "[%d] UPDATES %s", cid, formatUpdatesType(updates))
	return nil
}

// formatUpdateType returns a human-readable type name for an update.
func formatUpdateType(u tg.UpdateClass) string {
	if u == nil {
		return "nil"
	}
	// Try TypeInfo for rich output.
	if obj, ok := u.(interface{ TypeName() string }); ok {
		return obj.TypeName()
	}
	return fmt.Sprintf("%T", u)
}

// formatUpdatesType returns a human-readable type name for an updates class.
func formatUpdatesType(u tg.UpdatesClass) string {
	if u == nil {
		return "nil"
	}
	if obj, ok := u.(interface{ TypeName() string }); ok {
		return obj.TypeName()
	}
	return fmt.Sprintf("%T", u)
}

// StartUpdateLogger starts a goroutine that logs update timing.
// Returns a stop function.
func (l *UpdateListener) StartUpdateLogger(ctx context.Context) (stop func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cid := l.tracer.NextID()
				l.tracer.Printf(colorTiming, "[%d] HEARTBEAT client alive", cid)
			}
		}
	}()
	return func() { <-done }
}

var (
	colorUpdate = color.New(color.FgMagenta, color.Bold)
	colorTiming = color.New(color.FgHiBlack)
)
