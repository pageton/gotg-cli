package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/gotd/td/telegram"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pageton/gotg"
	"github.com/pageton/gotg-cli/internal/config"
	"github.com/pageton/gotg-cli/internal/ipc"
	"github.com/pageton/gotg-cli/internal/mcpserver"
	"github.com/pageton/gotg-cli/invoke"
	"github.com/pageton/gotg-cli/trace"
	"github.com/pageton/gotg/adapter"
	"github.com/pageton/gotg/ext/sqlite"
	"github.com/pageton/gotg/functions"
	"github.com/pageton/gotg/session"
	"github.com/pageton/gotg/storage"
)

func main() {
	os.Exit(run())
}

// run executes the CLI and returns an exit code.
func run() int {
	// Handle --version flag before command dispatch.
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			printVersion()
			return 0
		}
	}

	if len(os.Args) < 2 {
		usage()
		return 2
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "invoke":
		err = cmdInvoke(args)
	case "listen":
		err = cmdListen(args)
	case "trace":
		err = cmdTrace(args)
	case "methods":
		err = cmdMethods(args)
	case "completion":
		err = cmdCompletion(args)
	case "mcp":
		err = cmdMCP(args)
	case "export-session":
		err = cmdExportSession(args)
	case "getme":
		err = cmdGetMe(args)
	case "help":
		usage()
		return 0
	case "version":
		printVersion()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		return 2
	}

	return exitCode(err)
}

// exitCode maps an error to a Unix exit code.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ue *usageError
	if errors.As(err, &ue) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return 1
}

// usageError wraps an error caused by invalid user input (flags, args).
type usageError struct {
	err error
}

func usageErrorf(format string, args ...any) *usageError {
	return &usageError{err: fmt.Errorf(format, args...)}
}

func (e *usageError) Error() string { return e.err.Error() }

func (e *usageError) Unwrap() error { return e.err }

func usage() {
	fmt.Fprintln(os.Stderr, `tgdev — Telegram MTProto debug & invoke tool

Usage:
  tgdev <command> [flags]

Commands:
  invoke <method> [json]  Invoke a TL method (via running listener or standalone)
  listen                  Start persistent client with update stream + invoke server
  trace                   Full lifecycle tracing with correlation IDs
  methods                 List all available TL methods
  completion <shell>      Generate shell completion script (bash, zsh, fish)
  mcp                     Start an MCP server exposing tgdev tools over stdio or HTTP
  export-session          Export session string
  getme                   Get info about the current user/bot (users.getFullUser)
  version                 Print version information
  help                    Show this help message

Invoke Modes:
  If tgdev listen is running, invoke sends commands via IPC socket.
  Otherwise, invoke creates its own connection (slower, requires flags).

  # Terminal 1 — start listener
  tgdev listen --db ~/tgdev.db --api-id 123 --api-hash abc --bot-token TOKEN

  # Terminal 2 — invoke through the listener
  tgdev invoke messages.sendMessage '{"Message":"hello","Peer":{...}}'

Global Flags:
  --api-id INT        Telegram API ID
  --api-hash STRING   Telegram API Hash
  --session STRING    Session string
  --bot-token STRING  Bot token (alternative to session)
  --phone STRING      Phone number (alternative to session)
  --db PATH           SQLite database path for persistent session storage
  --db-name STRING    Session name within the database (default: "default")
  --socket PATH       Unix socket path for IPC (default: ~/.tgdev/tgdev.sock)
  --config PATH       Config file path (default: ~/.tgdev.json)
  --no-color          Disable colored output
  --debug             Enable verbose debug output
  --format FORMAT     Output format: text (default), json

Environment Variables:
  TGDEV_API_ID        Telegram API ID (avoids exposing in process args)
  TGDEV_API_HASH      Telegram API Hash
  TGDEV_BOT_TOKEN     Bot token
  TGDEV_SESSION       Session string
  TGDEV_PHONE         Phone number

MCP Flags:
  --http ADDR         Serve MCP over stateless Streamable HTTP instead of stdio

Note:
  --debug logs full API request/response payloads to stderr.
  Do not use in shared terminals or redirect to persistent logs.`)
}

// parseGlobalFlags extracts global flags from args.
func parseGlobalFlags(args []string) (*config.Config, []string, error) {
	var (
		apiID, apiHash, sessionStr, botToken, phone string
		dbPath, dbName, socketPath, configPath      string
		formatStr                                   string
		noColor, debug                              bool
		remaining                                   []string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--api-id":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--api-id requires a value")
			}
			apiID = args[i]
		case "--api-hash":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--api-hash requires a value")
			}
			apiHash = args[i]
		case "--session":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--session requires a value")
			}
			sessionStr = args[i]
		case "--bot-token":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--bot-token requires a value")
			}
			botToken = args[i]
		case "--phone":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--phone requires a value")
			}
			phone = args[i]
		case "--db":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--db requires a value")
			}
			dbPath = args[i]
		case "--db-name":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--db-name requires a value")
			}
			dbName = args[i]
		case "--socket":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--socket requires a value")
			}
			socketPath = args[i]
		case "--config":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--config requires a value")
			}
			configPath = args[i]
		case "--no-color":
			noColor = true
		case "--debug":
			debug = true
		case "--format":
			i++
			if i >= len(args) {
				return nil, nil, usageErrorf("--format requires a value")
			}
			formatStr = args[i]
		default:
			remaining = append(remaining, args[i])
		}
	}

	if configPath == "" {
		configPath = config.DefaultPath()
	}

	var apiIDInt int
	if apiID != "" {
		var err error
		apiIDInt, err = config.ParseAPIID(apiID)
		if err != nil {
			return nil, nil, usageErrorf("invalid --api-id: %v", err)
		}
	}

	cfg, err := config.FromFlags(apiIDInt, apiHash, sessionStr, botToken, phone, dbPath, dbName, socketPath, configPath)
	if err != nil {
		return nil, nil, err
	}
	cfg.ConfigPath = configPath
	cfg.NoColor = noColor
	cfg.Debug = debug
	cfg.Format = formatStr

	return cfg, remaining, nil
}

// buildSessionConstructor creates the session constructor from config.
// Priority: --db (SQLite) > --session (string) > in-memory.
func buildSessionConstructor(cfg *config.Config) (session.SessionConstructor, bool, error) {
	if cfg.HasDatabase() {
		dir := filepath.Dir(cfg.DatabasePath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, false, fmt.Errorf("create db directory %s: %w", dir, err)
		}
		adapter, err := sqlitedb.NewFromDSN(cfg.DatabasePath,
			sqlitedb.SessionName(cfg.DBName()),
		)
		if err != nil {
			return nil, false, fmt.Errorf("open sqlite %s: %w", cfg.DatabasePath, err)
		}
		// Restrict database permissions to owner-only; SQLite defaults to 0644.
		os.Chmod(cfg.DatabasePath, 0600)
		return session.Adapter(adapter), false, nil
	}

	if cfg.Session != "" {
		return session.StringSession(cfg.Session), false, nil
	}

	return session.SimpleSession(), true, nil
}

// createClient creates a gotg client with debug middleware from the given config.
func createClient(cfg *config.Config, middlewares ...telegram.Middleware) (*gotg.Client, error) {
	allMiddlewares := make([]telegram.Middleware, 0)
	if cfg.Debug {
		allMiddlewares = append(allMiddlewares, invoke.NewMiddleware(invoke.Config{
			Enabled:  true,
			LogReqs:  true,
			LogResps: true,
			UseColor: cfg.UseColor(),
		}))
	}
	allMiddlewares = append(allMiddlewares, middlewares...)

	opts := &gotg.ClientOpts{
		DisableCopyright: true,
		Middlewares:      allMiddlewares,
	}

	sessionCtor, inMemory, err := buildSessionConstructor(cfg)
	if err != nil {
		return nil, fmt.Errorf("session: %w", err)
	}
	opts.Session = sessionCtor
	if inMemory {
		opts.InMemory = true
	}

	var client *gotg.Client
	switch {
	case cfg.BotToken != "":
		client, err = gotg.NewClient(cfg.APIID, cfg.APIHash, gotg.AsBot(cfg.BotToken), opts)
	case cfg.Phone != "":
		client, err = gotg.NewClient(cfg.APIID, cfg.APIHash, gotg.AsUser(cfg.Phone), opts)
	default:
		client, err = gotg.NewClient(cfg.APIID, cfg.APIHash, gotg.Simple(), opts)
	}
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	return client, nil
}

// makeIPCHandler creates an IPC handler that invokes methods on a tg.Client.
func makeIPCHandler(client *gotg.Client) ipc.Handler {
	return func(ctx context.Context, req ipc.InvokeRequest) ipc.InvokeResponse {
		var result string
		var err error

		if req.Format == "json" {
			result, err = invoke.InvokeJSON(ctx, client.API(), req.Method, req.Params)
		} else {
			result, err = invoke.Invoke(ctx, client.API(), req.Method, req.Params)
		}

		if err != nil {
			return ipc.InvokeResponse{OK: false, Error: err.Error()}
		}
		return ipc.InvokeResponse{OK: true, Result: result}
	}
}

// invokeMethod dispatches a TL method invoke via IPC (if a listener is running)
// or creates a standalone client. Used by cmdInvoke and cmdGetMe.
func invokeMethod(cfg *config.Config, method string, params json.RawMessage) (string, error) {
	socketPath := cfg.GetSocketPath()

	// Try IPC first — if a listener is running, use it.
	if ipc.Ping(socketPath) {
		client := ipc.NewClient(socketPath)
		resp, err := client.Invoke(context.Background(), ipc.InvokeRequest{
			Method: method,
			Params: params,
			Format: cfg.Format,
		})
		if err != nil {
			return "", fmt.Errorf("ipc invoke: %w", err)
		}
		if !resp.OK {
			return "", fmt.Errorf("invoke error: %s", resp.Error)
		}
		return resp.Result, nil
	}

	// No listener running — create a standalone client (slower, requires auth flags).
	client, err := createClient(cfg)
	if err != nil {
		return "", fmt.Errorf("client error: %w (start 'tgdev listen' first or provide auth flags)", err)
	}

	if cfg.Format == "json" {
		return invoke.InvokeJSON(context.Background(), client.API(), method, params)
	}
	return invoke.Invoke(context.Background(), client.API(), method, params)
}

func cmdMCP(args []string) error {
	cfg, remaining, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	httpAddr := ""
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--http":
			i++
			if i >= len(remaining) {
				return usageErrorf("--http requires a value")
			}
			httpAddr = remaining[i]
		default:
			return usageErrorf("unsupported mcp flag or argument: %s", remaining[i])
		}
	}

	baseCfg := *cfg
	server := mcpserver.New(mcpserver.Options{
		Version:    version,
		SocketPath: cfg.GetSocketPath(),
		ConfigPath: cfg.ConfigPath,
		Format:     cfg.Format,
		Invoke: func(ctx context.Context, method string, params json.RawMessage, format string) (string, error) {
			invokeCfg := baseCfg
			invokeCfg.Format = format
			return invokeMethod(&invokeCfg, method, params)
		},
	})

	if httpAddr != "" {
		handler := sdkmcp.NewStreamableHTTPHandler(func(r *http.Request) *sdkmcp.Server {
			return server
		}, &sdkmcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		})
		fmt.Fprintf(os.Stderr, "MCP HTTP listening on %s\n", httpAddr)
		return http.ListenAndServe(httpAddr, handler)
	}

	return server.Run(context.Background(), &sdkmcp.StdioTransport{})
}

func cmdInvoke(args []string) error {
	cfg, remaining, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: tgdev invoke <method> [json]")
		fmt.Fprintln(os.Stderr, "Example: tgdev invoke messages.sendMessage '{\"Message\":\"hello\",...}'")
		return usageErrorf("method argument required")
	}

	method := remaining[0]
	var params json.RawMessage
	if len(remaining) > 1 {
		params = json.RawMessage(remaining[1])
	}

	result, err := invokeMethod(cfg, method, params)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

func cmdListen(args []string) error {
	cfg, _, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	tracer := trace.NewTracer(trace.Config{
		Output:   os.Stderr,
		UseColor: cfg.UseColor(),
	})

	client, err := createClient(cfg, tracer.Middleware())
	if err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	// Start IPC server for invoke commands.
	socketPath := cfg.GetSocketPath()
	ipcServer := ipc.NewServer(socketPath, makeIPCHandler(client))
	if err := ipcServer.Start(); err != nil {
		return fmt.Errorf("ipc server: %w", err)
	}
	defer ipcServer.Close()

	dbInfo := "in-memory"
	if cfg.HasDatabase() {
		dbInfo = cfg.DatabasePath
	}

	fmt.Fprintf(os.Stderr, "Listening... (session: %s, socket: %s)\n", dbInfo, socketPath)
	fmt.Fprintf(os.Stderr, "Invoke from another terminal: tgdev invoke <method> [json]\n")

	listener := trace.NewUpdateListener(tracer)
	stopHeartbeat := listener.StartUpdateLogger(context.Background())

	client.Dispatcher.AddHandlerToGroup(&updateHandler{tracer: tracer}, 0)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	stopHeartbeat()
	fmt.Fprintf(os.Stderr, "\nStopped.\n")
	return nil
}

func cmdTrace(args []string) error {
	cfg, _, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	tracer := trace.NewTracer(trace.Config{
		Output:   os.Stderr,
		UseColor: cfg.UseColor(),
	})

	client, err := createClient(cfg, tracer.Middleware())
	if err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	socketPath := cfg.GetSocketPath()
	ipcServer := ipc.NewServer(socketPath, makeIPCHandler(client))
	if err := ipcServer.Start(); err != nil {
		return fmt.Errorf("ipc server: %w", err)
	}
	defer ipcServer.Close()

	fmt.Fprintf(os.Stderr, "Tracing... (socket: %s, Ctrl+C to stop)\n", socketPath)

	listener := trace.NewUpdateListener(tracer)
	stopHeartbeat := listener.StartUpdateLogger(context.Background())

	client.Dispatcher.AddHandlerToGroup(&updateHandler{tracer: tracer}, 0)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	stopHeartbeat()
	fmt.Fprintf(os.Stderr, "\nTrace stopped.\n")
	return nil
}

func cmdMethods(args []string) error {
	cfg, remaining, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	reg := invoke.GlobalRegistry()
	methods := reg.Methods()
	sort.Strings(methods)

	quiet := false
	prefix := ""

	for _, a := range remaining {
		switch a {
		case "--quiet":
			quiet = true
		default:
			if prefix == "" {
				prefix = a
			}
		}
	}

	if prefix != "" {
		filtered := make([]string, 0)
		for _, m := range methods {
			if strings.HasPrefix(m, prefix) {
				filtered = append(filtered, m)
			}
		}
		methods = filtered
	}

	switch cfg.Format {
	case "json":
		data, err := json.Marshal(methods)
		if err != nil {
			return fmt.Errorf("marshal methods: %w", err)
		}
		fmt.Println(string(data))
	default:
		for _, m := range methods {
			fmt.Println(m)
		}
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "\n%d methods\n", len(methods))
	}
	return nil
}

func cmdCompletion(args []string) error {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}
	switch shell {
	case "bash":
		printBashCompletion()
	case "zsh":
		printZshCompletion()
	case "fish":
		printFishCompletion()
	default:
		return usageErrorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
	}
	return nil
}

func printBashCompletion() {
	script := `#!/bin/bash
_tgdev_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    local commands="invoke listen trace methods completion mcp export-session getme version help"

    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        return
    fi

    case "${COMP_WORDS[1]}" in
        invoke|methods)
            local methods=$(tgdev methods --quiet 2>/dev/null || true)
            COMPREPLY=($(compgen -W "$methods" -- "$cur"))
            ;;
        *)
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            ;;
    esac
}
complete -F _tgdev_completions tgdev
`
	fmt.Print(script)
}

func printZshCompletion() {
	script := `#compdef tgdev
_tgdev() {
    local -a commands
    commands=('invoke:Invoke TL method' 'listen:Start persistent client' 'trace:Full lifecycle tracing' 'methods:List TL methods' 'completion:Shell completion' 'mcp:Start MCP server' 'export-session:Export session string' 'getme:Get current user/bot info' 'version:Print version information' 'help:Show help')
    _arguments "1:command:->command" "*::arg:->args"
    case $state in
        command) _describe 'command' commands ;;
    esac
}
_tgdev "$@"
`
	fmt.Print(script)
}

func printFishCompletion() {
	script := `# fish completion for tgdev
complete -c tgdev -n '__fish_use_subcommand' -a 'invoke' -d 'Invoke TL method'
complete -c tgdev -n '__fish_use_subcommand' -a 'listen' -d 'Start persistent client'
complete -c tgdev -n '__fish_use_subcommand' -a 'trace' -d 'Full lifecycle tracing'
complete -c tgdev -n '__fish_use_subcommand' -a 'methods' -d 'List TL methods'
complete -c tgdev -n '__fish_use_subcommand' -a 'completion' -d 'Shell completion'
complete -c tgdev -n '__fish_use_subcommand' -a 'mcp' -d 'Start MCP server'
complete -c tgdev -n '__fish_use_subcommand' -a 'export-session' -d 'Export session string'
complete -c tgdev -n '__fish_use_subcommand' -a 'getme' -d 'Get current user/bot info'
complete -c tgdev -n '__fish_use_subcommand' -a 'version' -d 'Print version information'
complete -c tgdev -n '__fish_use_subcommand' -a 'help' -d 'Show help'
complete -c tgdev -n '__fish_seen_subcommand_from invoke methods' -a '(tgdev methods --quiet 2>/dev/null)'
`
	fmt.Print(script)
}

func cmdExportSession(args []string) error {
	cfg, _, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	if !cfg.HasDatabase() {
		return usageErrorf("--db flag is required for export-session")
	}

	adapter, err := sqlitedb.NewFromDSN(cfg.DatabasePath,
		sqlitedb.SessionName(cfg.DBName()),
	)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer adapter.Close()
	// Restrict database permissions to owner-only.
	os.Chmod(cfg.DatabasePath, 0600)

	sess, err := adapter.GetSession(storage.LatestVersion)
	if err != nil {
		return fmt.Errorf("read session: %w", err)
	}
	if sess == nil {
		return fmt.Errorf("no session data found in database")
	}

	str, err := functions.EncodeSessionToString(sess)
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	// Warn the user that the session string is sensitive.
	fmt.Fprintln(os.Stderr, "# WARNING: The session string below grants full account access. Treat it like a password.")
	fmt.Fprintln(os.Stderr, "# Do not share, commit to version control, or store in plaintext logs.")
	fmt.Println(str)
	return nil
}

func cmdGetMe(args []string) error {
	cfg, _, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	result, err := invokeMethod(cfg, "users.getFullUser", json.RawMessage(`{"ID":{"_":"inputUserSelf"}}`))
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}

type updateHandler struct {
	tracer *trace.Tracer
}

func (h *updateHandler) CheckUpdate(ctx *adapter.Context, u *adapter.Update) error {
	if u.UpdateClass != nil {
		if tn, ok := u.UpdateClass.(interface{ TypeName() string }); ok {
			cid := h.tracer.NextID()
			h.tracer.Printf(invoke.ColorMethod, "[%d] UPDATE %s", cid, tn.TypeName())
		}
	}
	return nil
}
