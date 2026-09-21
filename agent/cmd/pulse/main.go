package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/pipeline"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	cmd := "demo"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "demo":
		cfg := pipeline.DefaultConfig()
		res, err := pipeline.Run(context.Background(), cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "demo: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(pipeline.Format(res))
	case "version", "-v", "--version":
		fmt.Println("pulse", version)
	case "help", "-h", "--help":
		fmt.Print(usage())
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage())
		os.Exit(2)
	}
}

func usage() string {
	return strings.TrimSpace(`
Pulse on Monad — auditable off-chain trading agent

Usage:
  pulse demo      one simulated decision → audit JSONL → dry-run stamp → fake outcome → weight update
  pulse version

Env: see repo-root .env.example (MONAD_RPC_URL, PULSE_TRADE_STAMP, PRIVATE_KEY, PULSE_DRY_RUN, …).
Demo always dry-runs the stamp unless PULSE_LIVE_STAMP=1.
`) + "\n"
}
