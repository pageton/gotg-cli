package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectTestServer(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	t.Cleanup(func() { serverSession.Close() })
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { clientSession.Close() })
	return clientSession
}

func callTool[T any](t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, T) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	var out T
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content for %s: %v", name, err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal structured content for %s: %v\n%s", name, err, data)
	}
	return res, out
}

func TestListMethodsToolPaginatesAndFilters(t *testing.T) {
	session := connectTestServer(t, New(Options{Version: "test", SocketPath: t.TempDir() + "/tgdev.sock"}))

	res, out := callTool[ListMethodsOutput](t, session, "tgdev_list_methods", map[string]any{
		"prefix": "users.",
		"limit":  2,
	})
	if res.IsError {
		t.Fatalf("tgdev_list_methods returned tool error: %#v", res.Content)
	}
	if out.Total < 2 {
		t.Fatalf("expected at least two users.* methods, got total %d", out.Total)
	}
	if len(out.Methods) != 2 {
		t.Fatalf("expected page size 2, got %d", len(out.Methods))
	}
	if out.NextCursor == "" {
		t.Fatal("expected next cursor for truncated page")
	}
	for _, method := range out.Methods {
		if !strings.HasPrefix(method.Name, "users.") {
			t.Fatalf("method %q does not match users. prefix", method.Name)
		}
		if method.ID == "0x0" || method.FullName == "" {
			t.Fatalf("method summary missing ID/full name: %+v", method)
		}
	}
}

func TestDescribeMethodToolReportsFields(t *testing.T) {
	session := connectTestServer(t, New(Options{Version: "test", SocketPath: t.TempDir() + "/tgdev.sock"}))

	res, out := callTool[DescribeMethodOutput](t, session, "tgdev_describe_method", map[string]any{
		"method": "users.getFullUser",
	})
	if res.IsError {
		t.Fatalf("tgdev_describe_method returned tool error: %#v", res.Content)
	}
	if out.Name != "users.getFullUser" {
		t.Fatalf("method name = %q, want users.getFullUser", out.Name)
	}
	if out.ID == "0x0" || out.FullName == "" || out.GoType == "" {
		t.Fatalf("description missing identity fields: %+v", out)
	}
	foundID := false
	for _, field := range out.Fields {
		if field.Name == "ID" {
			foundID = true
			if field.ConstructorHint == "" {
				t.Fatalf("ID field should include constructor hint: %+v", field)
			}
		}
	}
	if !foundID {
		t.Fatalf("expected users.getFullUser description to include ID field, got %+v", out.Fields)
	}
}

func TestStatusAndConfigToolsDoNotExposeSecrets(t *testing.T) {
	socket := t.TempDir() + "/tgdev.sock"
	session := connectTestServer(t, New(Options{Version: "test", SocketPath: socket, ConfigPath: "/tmp/tgdev.json", Format: "json"}))

	_, status := callTool[ListenerStatusOutput](t, session, "tgdev_listener_status", nil)
	if status.SocketPath != socket {
		t.Fatalf("socket path = %q, want %q", status.SocketPath, socket)
	}
	if status.Running {
		t.Fatal("listener should not be running on temp socket")
	}

	_, info := callTool[ConfigInfoOutput](t, session, "tgdev_config_info", nil)
	if info.SocketPath != socket || info.ConfigPath != "/tmp/tgdev.json" || info.Format != "json" || info.Version != "test" {
		t.Fatalf("unexpected config info: %+v", info)
	}
}

func TestInvokeToolUsesConfiguredInvoker(t *testing.T) {
	var gotMethod string
	var gotParams json.RawMessage
	var gotFormat string
	session := connectTestServer(t, New(Options{
		Version:    "test",
		SocketPath: t.TempDir() + "/tgdev.sock",
		Invoke: func(ctx context.Context, method string, params json.RawMessage, format string) (string, error) {
			gotMethod = method
			gotParams = append(json.RawMessage(nil), params...)
			gotFormat = format
			return `{"ok":true}`, nil
		},
	}))

	res, out := callTool[InvokeOutput](t, session, "tgdev_invoke", map[string]any{
		"method": "users.getFullUser",
		"params": map[string]any{"ID": map[string]any{"_": "inputUserSelf"}},
		"format": "json",
	})
	if res.IsError {
		t.Fatalf("tgdev_invoke returned tool error: %#v", res.Content)
	}
	if gotMethod != "users.getFullUser" || gotFormat != "json" {
		t.Fatalf("invoker got method=%q format=%q", gotMethod, gotFormat)
	}
	if !json.Valid(gotParams) || !strings.Contains(string(gotParams), "inputUserSelf") {
		t.Fatalf("invoker got invalid params: %s", gotParams)
	}
	resultJSON, ok := out.ResultJSON.(map[string]any)
	if out.ResultText != `{"ok":true}` || !ok || resultJSON["ok"] != true {
		t.Fatalf("unexpected invoke output: %+v", out)
	}
}

func TestInvokeToolUnknownMethodIsToolError(t *testing.T) {
	session := connectTestServer(t, New(Options{Version: "test", SocketPath: t.TempDir() + "/tgdev.sock"}))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "tgdev_invoke",
		Arguments: map[string]any{"method": "missing.method"},
	})
	if err != nil {
		t.Fatalf("call tgdev_invoke: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected unknown method to be reported as a tool error, got %#v", res)
	}
}
