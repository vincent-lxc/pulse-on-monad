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
	flags := parseArgs(os.Args[1:])
	switch flags.cmd {
	case "demo":
		cfg := pipeline.DefaultConfig()
		cfg.Live = flags.live
		cfg.ForceJev = flags.jev
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
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", flags.cmd, usage())
		os.Exit(2)
	}
}

type cliFlags struct {
	cmd  string
	live bool
	jev  bool
}

func parseArgs(args []string) cliFlags {
	f := cliFlags{cmd: "demo"}
	if len(args) == 0 {
		return f
	}
	switch args[0] {
	case "--live":
		f.live = true
	case "--jev":
		f.jev = true
	default:
		f.cmd = args[0]
	}
	for _, a := range args[1:] {
		switch a {
		case "--live":
			f.live = true
		case "--jev":
			f.jev = true
		}
	}
	return f
}

func usage() string {
	return strings.TrimSpace(`
Pulse on Monad — auditable off-chain trading agent

Usage:
  pulse demo           one simulated decision → audit JSONL → dry-run stamp → fake outcome → weight update
  pulse demo --live    same, then broadcast stamp() on Monad testnet (needs PRIVATE_KEY + testnet MON)
  pulse demo --jev     require TypeSafe Jev (TYPESAFE_API_KEY / JEV_API_KEY); do not stub
  pulse version

Live can also be enabled with PULSE_STAMP_LIVE=1.
With TYPESAFE_API_KEY set, demo calls native TypeSafe Jev; otherwise the soft gate stubs.

Env: see repo-root .env.example. Never commit a real private key or TypeSafe key.
Default stamp is dry-run (CI-safe). --live spends testnet MON.
`) + "\n"
}
