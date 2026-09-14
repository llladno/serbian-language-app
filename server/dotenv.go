package main

import (
	"os"
	"strings"
)

// loadDotEnv reads KEY=VALUE pairs from a local .env file into the process
// environment — a dev convenience so secrets like TELEGRAM_BOT_TOKEN or SMTP
// creds can live in one untracked file instead of being retyped on every
// `go run`/`make dev`. A real environment variable always wins: this never
// overwrites one that's already set, so prod (which sets env vars directly,
// no .env file) is unaffected. A missing file is not an error — most
// invocations (tests, CI, prod) have none.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, val)
		}
	}
}
