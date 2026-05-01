package invoke

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tdp"
	"github.com/gotd/td/tg"
)

// Invoke calls the given TL method by name with JSON-encoded parameters.
// It resolves the method name to the correct gotd request type, unmarshals
// the JSON params, invokes via the raw RPC invoker, and returns the response
// formatted via tdp.Format.
//
// Uses the raw invoker (client.Invoker()) rather than calling typed methods
// on tg.Client, since typed methods have varying signatures that don't work
// with a generic reflection approach.
func Invoke(ctx context.Context, client *tg.Client, method string, params json.RawMessage) (string, error) {
	reg := GlobalRegistry()

	req, err := reg.NewRequest(method)
	if err != nil {
		return "", fmt.Errorf("resolve method %q: %w", method, err)
	}

	if len(params) > 0 {
		if err := unmarshalRequest(params, req); err != nil {
			return "", fmt.Errorf("unmarshal params for %q: %w", method, err)
		}
	}

	// Capture raw response bytes.
	var resp captureDecoder
	if err := client.Invoker().Invoke(ctx, req, &resp); err != nil {
		return "", fmt.Errorf("invoke %q: %w", method, err)
	}

	if len(resp.data) == 0 {
		return "(empty response)", nil
	}

	// Try to decode the response using the type constructor registry.
	result, decodeErr := decodeResponse(resp.data)
	if result != nil {
		switch v := result.(type) {
		case tdp.Object:
			return tdp.Format(v), nil
		case *genericVector:
			return v.String(), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	}
	if decodeErr != nil {
		return "", fmt.Errorf("decode response: %w", decodeErr)
	}
	return fmt.Sprintf("(%d bytes)", len(resp.data)), nil
}

// captureDecoder implements bin.Decoder by capturing the raw response buffer.
type captureDecoder struct {
	data []byte
}

func (c *captureDecoder) Decode(b *bin.Buffer) error {
	c.data = b.Copy()
	return nil
}

func decodeResponse(data []byte) (interface{}, error) {
	if len(data) < 4 {
		return nil, nil
	}

	buf := &bin.Buffer{Buf: data}

	id, err := buf.Uint32()
	if err != nil {
		return nil, fmt.Errorf("read constructor id: %w", err)
	}

	// Handle bare vectors: decode elements by their constructor IDs.
	if id == bin.TypeVector {
		return decodeVectorResponse(data)
	}

	ctors := GlobalRegistry().ctors
	ctor, ok := ctors[id]
	if !ok {
		return nil, nil // unrecognized constructor
	}

	// Reset buffer to start — obj.Decode() expects to read the constructor ID.
	buf.ResetTo(data)

	obj := ctor()
	if err := obj.Decode(buf); err != nil {
		return nil, fmt.Errorf("decode %T: %w", obj, err)
	}

	return obj, nil
}

// genericVector represents a decoded bare vector of TL objects.
type genericVector struct {
	Elems []interface{}
}

func (v *genericVector) String() string {
	var b strings.Builder
	b.WriteString("vector")
	for i, elem := range v.Elems {
		if i > 0 {
			b.WriteByte('\n')
		}
		if o, ok := elem.(tdp.Object); ok {
			b.WriteString(tdp.Format(o))
		} else {
			fmt.Fprintf(&b, "%v", elem)
		}
	}
	return b.String()
}

func decodeVectorResponse(data []byte) (interface{}, error) {
	buf := &bin.Buffer{Buf: data}

	// VectorHeader reads and validates the vector type ID, then reads count.
	count, err := buf.VectorHeader()
	if err != nil {
		return nil, fmt.Errorf("read vector header: %w", err)
	}

	// Cap pre-allocation to prevent OOM from crafted count values.
	const maxVectorElements = 100000
	if count > maxVectorElements {
		return nil, fmt.Errorf("vector count %d exceeds safety limit %d", count, maxVectorElements)
	}

	ctors := GlobalRegistry().ctors
	vec := &genericVector{Elems: make([]interface{}, 0, count)}

	var firstErr error
	for i := 0; i < count; i++ {
		elemID, err := buf.Uint32()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("element %d: read id: %w", i, err)
			}
			break
		}
		ctor, ok := ctors[elemID]
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("element %d: unknown constructor 0x%08x", i, elemID)
			}
			break
		}

		remaining := buf.Buf
		idBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(idBytes, elemID)
		elemData := append(idBytes, remaining...)

		elemBuf := &bin.Buffer{Buf: elemData}
		obj := ctor()
		if err := obj.Decode(elemBuf); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("element %d: decode %T: %w", i, obj, err)
			}
			break
		}

		consumed := len(remaining) - len(elemBuf.Buf)
		buf.Buf = remaining[consumed:]

		vec.Elems = append(vec.Elems, obj)
	}

	// Return partial results even if some elements failed.
	if len(vec.Elems) > 0 {
		return vec, firstErr
	}
	return nil, firstErr
}