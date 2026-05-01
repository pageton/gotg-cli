package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pageton/gotg-cli/internal/ipc"
	"github.com/pageton/gotg-cli/invoke"
)

const (
	serverName    = "tgdev"
	defaultLimit  = 100
	maxLimit      = 500
	defaultFormat = "text"
)

// InvokeFunc invokes a Telegram TL method and returns either formatted text or
// JSON text, depending on format.
type InvokeFunc func(ctx context.Context, method string, params json.RawMessage, format string) (string, error)

// Options configures the tgdev MCP server.
type Options struct {
	Version    string
	SocketPath string
	ConfigPath string
	Format     string
	Invoke     InvokeFunc
}

// New returns an MCP server exposing tgdev registry, IPC, and invocation tools.
func New(opts Options) *mcp.Server {
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Format == "" {
		opts.Format = defaultFormat
	}

	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: opts.Version}, nil)
	registerTools(server, opts)
	return server
}

func registerTools(server *mcp.Server, opts Options) {
	openWorldTrue := true
	openWorld := false
	destructiveFalse := false

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_list_methods",
		Title:       "List Telegram TL methods",
		Description: "List available Telegram TL method constructors by optional prefix with cursor pagination.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			OpenWorldHint:   &openWorld,
			IdempotentHint:  true,
			DestructiveHint: &destructiveFalse,
		},
	}, ListMethods)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_describe_method",
		Title:       "Describe Telegram TL method",
		Description: "Describe a Telegram TL method request type, constructor ID, and JSON fields accepted by tgdev_invoke.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			OpenWorldHint:   &openWorld,
			IdempotentHint:  true,
			DestructiveHint: &destructiveFalse,
		},
	}, DescribeMethod)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_listener_status",
		Title:       "Check tgdev listener status",
		Description: "Check whether a tgdev listen or trace process is reachable on the configured Unix socket.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			OpenWorldHint:   &openWorld,
			IdempotentHint:  true,
			DestructiveHint: &destructiveFalse,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListenerStatusInput) (*mcp.CallToolResult, ListenerStatusOutput, error) {
		return nil, ListenerStatus(opts, input), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_config_info",
		Title:       "Show tgdev MCP configuration",
		Description: "Show non-secret tgdev MCP configuration such as socket, config path, and default output format.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			OpenWorldHint:   &openWorld,
			IdempotentHint:  true,
			DestructiveHint: &destructiveFalse,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ConfigInfoInput) (*mcp.CallToolResult, ConfigInfoOutput, error) {
		return nil, ConfigInfo(opts), nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_invoke",
		Title:       "Invoke Telegram TL method",
		Description: "Invoke a Telegram TL method through a running tgdev listener when available, or through tgdev standalone credentials supplied to the MCP command.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			OpenWorldHint:   &openWorldTrue,
			IdempotentHint:  false,
			DestructiveHint: nil,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input InvokeInput) (*mcp.CallToolResult, InvokeOutput, error) {
		return Invoke(ctx, opts, input)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tgdev_get_me",
		Title:       "Get current Telegram account",
		Description: "Return users.getFullUser for inputUserSelf using the same invocation path as tgdev_invoke.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			OpenWorldHint:   &openWorldTrue,
			IdempotentHint:  true,
			DestructiveHint: &destructiveFalse,
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetMeInput) (*mcp.CallToolResult, InvokeOutput, error) {
		format := normalizeFormat(input.Format, opts.Format)
		return Invoke(ctx, opts, InvokeInput{
			Method: "users.getFullUser",
			Params: map[string]any{"ID": map[string]any{"_": "inputUserSelf"}},
			Format: format,
		})
	})
}

// ListMethodsInput is the input for tgdev_list_methods.
type ListMethodsInput struct {
	Prefix string `json:"prefix,omitempty" jsonschema:"optional method prefix, for example messages. or users.get"`
	Cursor string `json:"cursor,omitempty" jsonschema:"opaque numeric cursor returned by a previous tgdev_list_methods call"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum methods to return; defaults to 100 and is capped at 500"`
}

// MethodSummary is a compact method listing entry.
type MethodSummary struct {
	Name     string `json:"name" jsonschema:"bare TL method name, for example messages.sendMessage"`
	ID       string `json:"id" jsonschema:"hex constructor ID"`
	FullName string `json:"full_name" jsonschema:"full TL schema name including hash suffix"`
}

// ListMethodsOutput is the output for tgdev_list_methods.
type ListMethodsOutput struct {
	Methods    []MethodSummary `json:"methods" jsonschema:"method entries for this page"`
	Total      int             `json:"total" jsonschema:"total number of methods after prefix filtering"`
	NextCursor string          `json:"next_cursor,omitempty" jsonschema:"cursor for the next page, absent when there are no more methods"`
}

// ListMethods returns registered TL methods with cursor pagination.
func ListMethods(ctx context.Context, req *mcp.CallToolRequest, input ListMethodsInput) (*mcp.CallToolResult, ListMethodsOutput, error) {
	reg := invoke.GlobalRegistry()
	methods := reg.Methods()
	sort.Strings(methods)

	if input.Prefix != "" {
		filtered := methods[:0]
		for _, method := range methods {
			if strings.HasPrefix(method, input.Prefix) {
				filtered = append(filtered, method)
			}
		}
		methods = filtered
	}

	start, err := parseCursor(input.Cursor)
	if err != nil {
		return nil, ListMethodsOutput{}, err
	}
	if start > len(methods) {
		start = len(methods)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	end := start + limit
	if end > len(methods) {
		end = len(methods)
	}

	out := ListMethodsOutput{
		Methods: make([]MethodSummary, 0, end-start),
		Total:   len(methods),
	}
	for _, method := range methods[start:end] {
		id, _ := reg.IDByName(method)
		_, full := reg.NameByID(id)
		out.Methods = append(out.Methods, MethodSummary{
			Name:     method,
			ID:       invoke.FormatHexID(id),
			FullName: full,
		})
	}
	if end < len(methods) {
		out.NextCursor = strconv.Itoa(end)
	}
	return nil, out, nil
}

func parseCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(cursor)
	if err != nil || start < 0 {
		return 0, fmt.Errorf("cursor must be a non-negative integer returned by tgdev_list_methods")
	}
	return start, nil
}

// DescribeMethodInput is the input for tgdev_describe_method.
type DescribeMethodInput struct {
	Method string `json:"method" jsonschema:"bare TL method name, for example users.getFullUser"`
}

// FieldDescription describes a Go request field accepted as JSON input.
type FieldDescription struct {
	Name            string `json:"name" jsonschema:"Go field name accepted case-insensitively in params JSON"`
	Type            string `json:"type" jsonschema:"Go type used by gotd for this field"`
	Kind            string `json:"kind" jsonschema:"reflection kind such as string, int, struct, interface, slice, or ptr"`
	JSONKey         string `json:"json_key" jsonschema:"recommended JSON key for tgdev_invoke params"`
	ConstructorHint string `json:"constructor_hint,omitempty" jsonschema:"hint for interface fields that require an object with a _ constructor key"`
}

// DescribeMethodOutput is the output for tgdev_describe_method.
type DescribeMethodOutput struct {
	Name     string             `json:"name" jsonschema:"bare TL method name"`
	ID       string             `json:"id" jsonschema:"hex constructor ID"`
	FullName string             `json:"full_name" jsonschema:"full TL schema name including hash suffix"`
	GoType   string             `json:"go_type" jsonschema:"gotd Go request type"`
	Fields   []FieldDescription `json:"fields" jsonschema:"fields accepted in params JSON"`
	Example  map[string]any     `json:"example,omitempty" jsonschema:"minimal example shape for interface constructor fields when detectable"`
}

// DescribeMethod describes a registered TL method request type.
func DescribeMethod(ctx context.Context, req *mcp.CallToolRequest, input DescribeMethodInput) (*mcp.CallToolResult, DescribeMethodOutput, error) {
	if input.Method == "" {
		return nil, DescribeMethodOutput{}, fmt.Errorf("method is required")
	}
	reg := invoke.GlobalRegistry()
	id, ok := reg.IDByName(input.Method)
	if !ok {
		return nil, DescribeMethodOutput{}, fmt.Errorf("unknown method %q; call tgdev_list_methods to discover supported method names", input.Method)
	}
	_, full := reg.NameByID(id)
	typ := reg.TypeByName(input.Method)
	if typ == nil {
		return nil, DescribeMethodOutput{}, fmt.Errorf("method %q has no registered Go request type", input.Method)
	}

	out := DescribeMethodOutput{
		Name:     input.Method,
		ID:       invoke.FormatHexID(id),
		FullName: full,
		GoType:   typ.String(),
	}
	structType := typ
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil, out, nil
	}

	out.Fields = make([]FieldDescription, 0, structType.NumField())
	out.Example = make(map[string]any)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		desc := FieldDescription{
			Name:    field.Name,
			Type:    field.Type.String(),
			Kind:    field.Type.Kind().String(),
			JSONKey: field.Name,
		}
		if field.Type.Kind() == reflect.Interface || (field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Interface) {
			desc.ConstructorHint = "provide an object with a \"_\" constructor name, for example {\"_\":\"inputUserSelf\"} when applicable"
			out.Example[field.Name] = map[string]any{"_": "constructorName"}
		}
		out.Fields = append(out.Fields, desc)
	}
	if len(out.Example) == 0 {
		out.Example = nil
	}
	return nil, out, nil
}

// ListenerStatusInput is the input for tgdev_listener_status.
type ListenerStatusInput struct{}

// ListenerStatusOutput is the output for tgdev_listener_status.
type ListenerStatusOutput struct {
	SocketPath string `json:"socket_path" jsonschema:"Unix socket checked for tgdev IPC"`
	Running    bool   `json:"running" jsonschema:"whether a listener accepted an IPC connection"`
}

// ListenerStatus reports whether tgdev IPC is reachable.
func ListenerStatus(opts Options, input ListenerStatusInput) ListenerStatusOutput {
	return ListenerStatusOutput{SocketPath: opts.SocketPath, Running: ipc.Ping(opts.SocketPath)}
}

// ConfigInfoInput is the input for tgdev_config_info.
type ConfigInfoInput struct{}

// ConfigInfoOutput is the output for tgdev_config_info.
type ConfigInfoOutput struct {
	SocketPath string `json:"socket_path" jsonschema:"Unix socket used for tgdev IPC"`
	ConfigPath string `json:"config_path,omitempty" jsonschema:"tgdev config file path used when the MCP server started"`
	Format     string `json:"format" jsonschema:"default invoke output format"`
	Version    string `json:"version" jsonschema:"MCP server version string"`
}

// ConfigInfo returns non-secret MCP configuration.
func ConfigInfo(opts Options) ConfigInfoOutput {
	return ConfigInfoOutput{
		SocketPath: opts.SocketPath,
		ConfigPath: opts.ConfigPath,
		Format:     normalizeFormat("", opts.Format),
		Version:    opts.Version,
	}
}

// InvokeInput is the input for tgdev_invoke.
type InvokeInput struct {
	Method string         `json:"method" jsonschema:"bare TL method name, for example messages.sendMessage"`
	Params map[string]any `json:"params,omitempty" jsonschema:"JSON object passed to the TL request; interface fields require a _ constructor key"`
	Format string         `json:"format,omitempty" jsonschema:"response format: text or json; defaults to MCP server --format"`
}

// InvokeOutput is the output for tgdev_invoke and tgdev_get_me.
type InvokeOutput struct {
	Method     string `json:"method" jsonschema:"invoked TL method name"`
	Format     string `json:"format" jsonschema:"response format used for result_text"`
	ResultText string `json:"result_text" jsonschema:"raw text returned by tgdev invoke"`
	ResultJSON any    `json:"result_json,omitempty" jsonschema:"parsed JSON response when format is json and the response is valid JSON"`
}

// Invoke invokes a TL method through the configured tgdev invocation function.
func Invoke(ctx context.Context, opts Options, input InvokeInput) (*mcp.CallToolResult, InvokeOutput, error) {
	if opts.Invoke == nil {
		return nil, InvokeOutput{}, fmt.Errorf("tgdev invocation is unavailable: MCP server was created without an InvokeFunc")
	}
	if input.Method == "" {
		return nil, InvokeOutput{}, fmt.Errorf("method is required")
	}
	if _, ok := invoke.GlobalRegistry().IDByName(input.Method); !ok {
		return nil, InvokeOutput{}, fmt.Errorf("unknown method %q; call tgdev_list_methods to discover supported method names", input.Method)
	}
	params, err := json.Marshal(input.Params)
	if err != nil {
		return nil, InvokeOutput{}, fmt.Errorf("marshal params: %w", err)
	}
	if input.Params == nil {
		params = nil
	}
	format := normalizeFormat(input.Format, opts.Format)
	result, err := opts.Invoke(ctx, input.Method, params, format)
	if err != nil {
		return nil, InvokeOutput{}, actionableInvokeError(opts.SocketPath, err)
	}

	out := InvokeOutput{Method: input.Method, Format: format, ResultText: result}
	if format == "json" && json.Valid([]byte(result)) {
		var decoded any
		if err := json.Unmarshal([]byte(result), &decoded); err == nil {
			out.ResultJSON = decoded
		}
	}
	return nil, out, nil
}

func actionableInvokeError(socketPath string, err error) error {
	if ipc.Ping(socketPath) {
		return fmt.Errorf("invoke failed through tgdev listener at %s: %w", socketPath, err)
	}
	return fmt.Errorf("invoke failed and no tgdev listener is reachable at %s: start 'tgdev listen'/'tgdev trace' or restart this MCP server with --api-id, --api-hash, and an auth flag: %w", socketPath, err)
}

// GetMeInput is the input for tgdev_get_me.
type GetMeInput struct {
	Format string `json:"format,omitempty" jsonschema:"response format: text or json; defaults to MCP server --format"`
}

func normalizeFormat(requested, fallback string) string {
	format := requested
	if format == "" {
		format = fallback
	}
	if format == "" {
		format = defaultFormat
	}
	switch strings.ToLower(format) {
	case "json":
		return "json"
	default:
		return "text"
	}
}
