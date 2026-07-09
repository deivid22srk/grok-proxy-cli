// Command grok-proxy-cli is a terminal-only build of Grok Proxy Plus.
//
// It does NOT depend on Wails or any GUI toolkit, so it can be built and
// run from a single command on Linux, macOS, and Windows without a
// desktop environment. It supports:
//
//   - `login`      : start the OAuth device-code flow and save the account
//   - `accounts`   : list configured accounts
//   - `use <id>`   : set the active account
//   - `logout <id>`: remove an account
//   - `models`     : list upstream models
//   - `chat`       : interactive REPL with streaming + thinking
//   - `serve`      : (default) start the local OpenAI-compatible proxy
//   - `ask <text>` : one-shot prompt from the command line
//
// All state (accounts, settings, usage, history) is stored under the same
// AppData directory used by the desktop app, so the two builds can share
// credentials.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"grok-desktop/internal/app"
	"grok-desktop/internal/store"
)

const version = "1.0.0-cli"

func main() {
	if len(os.Args) < 2 {
		runServe(nil)
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-h", "--help", "help":
		printHelp()
		return
	case "-v", "--version", "version":
		fmt.Println("grok-proxy-cli", version)
		return
	case "serve":
		runServe(args)
	case "login":
		runLogin(args)
	case "logout":
		runLogout(args)
	case "accounts":
		runAccounts(args)
	case "use":
		runUse(args)
	case "models":
		runModels(args)
	case "chat":
		runChat(args)
	case "ask":
		runAsk(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Printf(`grok-proxy-cli %s — terminal-only Grok proxy

Usage:
  grok-proxy-cli                 start the local OpenAI proxy (default = serve)
  grok-proxy-cli serve           same as above; flags: --listen, --api-key, --no-proxy
  grok-proxy-cli login           sign in with xAI device-code OAuth
  grok-proxy-cli accounts        list accounts
  grok-proxy-cli use <id>        set active account (id prefix supported)
  grok-proxy-cli logout <id>     remove an account
  grok-proxy-cli models          list models available to the active account
  grok-proxy-cli chat            interactive streaming chat REPL
  grok-proxy-cli ask "<prompt>"  one-shot prompt; flags: --effort, --model, --no-think

Global flags (any command):
  --data-dir <path>   override AppData directory (default: %s)

Proxy base URL once running:
  http://127.0.0.1:8787/v1

Endpoints:
  GET  /v1/models
  POST /v1/chat/completions
  POST /v1/responses
  POST /v1/messages
`, version, defaultDataDirHint())
}

func defaultDataDirHint() string {
	dir, err := store.DefaultDataDir()
	if err != nil {
		return "<user-appdata>/GrokDesktop"
	}
	return dir
}

// ---- shared helpers ----

func mustApp() *app.App {
	a, err := app.Open("")
	if err != nil {
		fail("open store: %v", err)
	}
	return a
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// ---- commands ----

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "", "override listen address (default 127.0.0.1:8787)")
	apiKey := fs.String("api-key", "", "require this API key on requests")
	noProxy := fs.Bool("no-proxy", false, "do not start the HTTP proxy")
	_ = fs.Parse(args)

	a := mustApp()
	defer a.Close()

	if *apiKey != "" {
		_ = a.UpdateSettings(func(s *store.Settings) { s.ProxyAPIKey = *apiKey })
	}
	if *listen != "" {
		_ = a.UpdateSettings(func(s *store.Settings) { s.ProxyListen = *listen; s.ProxyEnabled = true })
	}

	if !*noProxy {
		addr, err := a.StartProxy()
		if err != nil {
			fail("start proxy: %v", err)
		}
		fmt.Printf("grok-proxy-plus listening on http://%s/v1\n", addr)
		fmt.Println("endpoints:")
		fmt.Println("  GET  /v1/models")
		fmt.Println("  POST /v1/chat/completions")
		fmt.Println("  POST /v1/responses")
		fmt.Println("  POST /v1/messages")
		fmt.Println("press Ctrl+C to stop")
	} else {
		fmt.Println("proxy disabled (--no-proxy)")
	}

	// Show active account if any
	if acc, ok := a.ActiveAccount(); ok {
		fmt.Printf("active account: %s <%s>\n", acc.Label, acc.Email)
	} else {
		fmt.Println("no account configured — run `grok-proxy-cli login` first")
	}

	// Wait for signal
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()
	fmt.Println("\nshutting down…")
}

func runLogin(args []string) {
	a := mustApp()
	defer a.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Println("starting xAI device login…")
	start, err := a.StartDeviceLogin(ctx)
	if err != nil {
		fail("start device login: %v", err)
	}
	url := start.VerificationURL
	fmt.Println()
	fmt.Println("  1) Open this URL in a browser:")
	fmt.Printf("     %s\n", url)
	fmt.Println()
	fmt.Println("  2) When asked, enter the code:")
	fmt.Printf("     %s\n", start.UserCode)
	fmt.Println()
	fmt.Println("waiting for authorization (press Ctrl+C to cancel)…")

	acc, err := a.WaitDeviceLogin(ctx)
	if err != nil {
		fail("login: %v", err)
	}
	fmt.Printf("\nlogin OK — account: %s <%s>\n", acc.Label, acc.Email)
	fmt.Printf("saved to: %s\n", filepath.Join(a.DataDir(), "accounts"))
}

func runAccounts(args []string) {
	a := mustApp()
	defer a.Close()
	accs := a.ListAccounts()
	if len(accs) == 0 {
		fmt.Println("no accounts — run `grok-proxy-cli login`")
		return
	}
	active := a.ActiveAccountID()
	fmt.Printf("%-24s %-32s %-20s %s\n", "ID", "LABEL", "EMAIL", "ACTIVE")
	for _, acc := range accs {
		marker := ""
		if acc.ID == active {
			marker = "*"
		}
		fmt.Printf("%-24s %-32s %-20s %s\n", truncate(acc.ID, 24), truncate(acc.Label, 32), truncate(acc.Email, 20), marker)
	}
}

func runUse(args []string) {
	if len(args) < 1 {
		fail("usage: grok-proxy-cli use <id-or-prefix>")
	}
	a := mustApp()
	defer a.Close()
	id, err := a.ResolveAccountID(args[0])
	if err != nil {
		fail("%v", err)
	}
	if err := a.SetActiveAccount(id); err != nil {
		fail("%v", err)
	}
	fmt.Printf("active account: %s\n", id)
}

func runLogout(args []string) {
	if len(args) < 1 {
		fail("usage: grok-proxy-cli logout <id-or-prefix>")
	}
	a := mustApp()
	defer a.Close()
	id, err := a.ResolveAccountID(args[0])
	if err != nil {
		fail("%v", err)
	}
	if err := a.RemoveAccount(id); err != nil {
		fail("%v", err)
	}
	fmt.Printf("removed account: %s\n", id)
}

func runModels(args []string) {
	a := mustApp()
	defer a.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	models, err := a.ListModels(ctx)
	if err != nil {
		fail("list models: %v", err)
	}
	fmt.Printf("%-32s %-32s %s\n", "ID", "NAME", "MODE")
	for _, m := range models {
		fmt.Printf("%-32s %-32s %s\n", truncate(m.ID, 32), truncate(m.Name, 32), m.APIMode)
	}
}

func runChat(args []string) {
	a := mustApp()
	defer a.Close()
	if _, ok := a.ActiveAccount(); !ok {
		fail("no account — run `grok-proxy-cli login`")
	}

	history := []app.ChatMessage{}
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("grok-proxy-cli chat — type :q to quit, :clear to reset history")
	fmt.Println()
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return
		}
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if text == ":q" || text == ":quit" || text == ":exit" {
			return
		}
		if text == ":clear" {
			history = nil
			fmt.Println("(history cleared)")
			continue
		}
		history = append(history, app.ChatMessage{Role: "user", Content: text})
		fmt.Print("\n")
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGINT)
			<-ch
			cancel()
		}()
		var sb strings.Builder
		err = a.StreamChat(ctx, history, func(ev app.ChatEvent) {
			switch ev.Type {
			case "thinking":
				fmt.Fprintf(os.Stderr, "\033[2m%s\033[0m", ev.Text)
			case "content":
				fmt.Print(ev.Text)
				sb.WriteString(ev.Text)
			case "usage":
				fmt.Fprintf(os.Stderr, "\n\n[usage] prompt=%d completion=%d reasoning=%d total=%d\n",
					ev.Usage.PromptTokens, ev.Usage.CompletionTokens, ev.Usage.ReasoningTokens, ev.Usage.TotalTokens)
			case "error":
				fmt.Fprintf(os.Stderr, "\n[error] %s\n", ev.Error)
			}
		})
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[stream error] %v\n", err)
		}
		fmt.Println("\n")
		history = append(history, app.ChatMessage{Role: "assistant", Content: sb.String()})
	}
}

func runAsk(args []string) {
	fs := flag.NewFlagSet("ask", flag.ExitOnError)
	effort := fs.String("effort", "high", "reasoning effort: low|medium|high")
	model := fs.String("model", "", "model id (default: settings.default_model)")
	noThink := fs.Bool("no-think", false, "disable reasoning")
	_ = fs.Parse(args)
	prompt := strings.Join(fs.Args(), " ")
	if prompt == "" {
		fail("usage: grok-proxy-cli ask \"your prompt here\"")
	}
	a := mustApp()
	defer a.Close()
	if _, ok := a.ActiveAccount(); !ok {
		fail("no account — run `grok-proxy-cli login`")
	}
	history := []app.ChatMessage{{Role: "user", Content: prompt}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT)
		<-ch
		cancel()
	}()
	opts := app.ChatOptions{Model: *model, Effort: *effort, NoThinking: *noThink}
	err := a.StreamChatWithOptions(ctx, history, opts, func(ev app.ChatEvent) {
		switch ev.Type {
		case "thinking":
			fmt.Fprintf(os.Stderr, "\033[2m%s\033[0m", ev.Text)
		case "content":
			fmt.Print(ev.Text)
		case "usage":
			fmt.Fprintf(os.Stderr, "\n[usage] prompt=%d completion=%d reasoning=%d total=%d\n",
				ev.Usage.PromptTokens, ev.Usage.CompletionTokens, ev.Usage.ReasoningTokens, ev.Usage.TotalTokens)
		case "error":
			fmt.Fprintf(os.Stderr, "\n[error] %s\n", ev.Error)
		}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[stream error] %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
