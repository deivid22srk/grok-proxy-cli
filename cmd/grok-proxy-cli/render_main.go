//go:build render

package main

// render_main.go is a small bootstrap for the Render deployment.
// It reads GROK_ACCOUNTS_JSON (a JSON array of store.Account objects)
// and writes each one to <data_dir>/accounts/<id>.json before the
// normal grok-proxy-cli flow runs.
//
// Build with: go build -tags render -o grok-proxy-cli ./cmd/grok-proxy-cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"grok-desktop/internal/store"
)

func init() {
	accountsJSON := os.Getenv("GROK_ACCOUNTS_JSON")
	if accountsJSON == "" {
		return
	}

	dataDir := os.Getenv("GROK_DATA_DIR")
	if dataDir == "" {
		// fall back to the default the binary would use
		dd, err := store.DefaultDataDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[render] could not resolve data dir: %v\n", err)
			return
		}
		dataDir = dd
	}

	accountsDir := filepath.Join(dataDir, "accounts")
	if err := os.MkdirAll(accountsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[render] mkdir accounts: %v\n", err)
		return
	}

	var accounts []store.Account
	if err := json.Unmarshal([]byte(accountsJSON), &accounts); err != nil {
		fmt.Fprintf(os.Stderr, "[render] failed to parse GROK_ACCOUNTS_JSON: %v\n", err)
		return
	}

	written := 0
	for _, acc := range accounts {
		if acc.ID == "" || acc.AccessToken == "" {
			continue
		}
		safe := acc.ID
		for _, c := range []string{`\`, `/`, `:`, `*`, `?`, `"`, `<`, `>`, `|`} {
			safe = filepathClean(safe, c)
		}
		path := filepath.Join(accountsDir, safe+".json")
		b, err := json.MarshalIndent(acc, "", "  ")
		if err != nil {
			continue
		}
		if err := os.WriteFile(path, b, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "[render] write %s: %v\n", path, err)
			continue
		}
		written++
	}
	fmt.Fprintf(os.Stderr, "[render] wrote %d accounts to %s\n", written, accountsDir)
}

func filepathClean(s, bad string) string {
	out := ""
	for _, r := range s {
		if string(r) == bad {
			out += "_"
		} else {
			out += string(r)
		}
	}
	return out
}
