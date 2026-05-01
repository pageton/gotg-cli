package invoke

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gotd/td/tg"
)

// InvokeJSON is like Invoke but returns the response serialized as
// indented JSON instead of tdp.Format text.
func InvokeJSON(ctx context.Context, client *tg.Client, method string, params json.RawMessage) (string, error) {
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

	var resp captureDecoder
	if err := client.Invoker().Invoke(ctx, req, &resp); err != nil {
		return "", fmt.Errorf("invoke %q: %w", method, err)
	}

	if len(resp.data) == 0 {
		return "(empty response)", nil
	}

	result, decodeErr := decodeResponse(resp.data)
	if result != nil {
		return jsonMarshalResult(result)
	}
	if decodeErr != nil {
		return "", fmt.Errorf("decode response: %w", decodeErr)
	}
	return fmt.Sprintf("(%d bytes of unrecognized data)", len(resp.data)), nil
}

// jsonMarshalResult serializes a decoded TL response object as indented JSON.
// It handles tdp.Object values, genericVector values, and falls back to
// fmt.Sprintf for anything else.
func jsonMarshalResult(result interface{}) (string, error) {
	switch v := result.(type) {
	case *genericVector:
		return jsonMarshalVector(v)
	default:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", fmt.Errorf("json marshal response: %w", err)
		}
		return string(data), nil
	}
}

// jsonMarshalVector serializes a genericVector as a JSON array of its elements.
func jsonMarshalVector(vec *genericVector) (string, error) {
	data, err := json.MarshalIndent(vec.Elems, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal vector: %w", err)
	}
	return string(data), nil
}
