package main

import "fmt"


// Build variables injected via -ldflags at build time.
//
// Build with:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always) -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/tgdev/
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func printVersion() {
	fmt.Printf("tgdev %s (commit: %s, built: %s)\n", version, commit, buildTime)
}
