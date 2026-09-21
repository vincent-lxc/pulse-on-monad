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
	loadDotEnv()
	cmd, live := parseArgs(os.Args[1:])
	switch cmd {
	case "demo":
		cfg := pipeline.DefaultConfig()
		if live {
			cfg.Live = true
		}
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

func parseArgs(args []string) (cmd string, live bool) {
	cmd = "demo"
	if len(args) == 0 {
		return cmd, false
	}
	cmd = args[0]
	if cmd == "--live" {
		return "demo", true
	}
	for _, a := range args[1:] {
		if a == "--live" {
			live = true
		}
	}
	return cmd, live
}

func usage() string {
	return strings.TrimSpace(`
Pulse on Monad — auditable off-chain trading agent

Usage:
  pulse demo           one simulated decision → audit JSONL → dry-run stamp → fake outcome → weight update
  pulse demo --live    same, then broadcast stamp() on Monad testnet (needs PRIVATE_KEY + testnet MON)
  pulse version

Live can also be enabled with PULSE_STAMP_LIVE=1.

Env: see repo-root .env.example. Never commit a real private key.
Default is dry-run (CI-safe). --live spends testnet MON.
`) + "\n"
}
