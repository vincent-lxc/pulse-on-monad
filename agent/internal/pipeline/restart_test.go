package pipeline

import (
	"context"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/runlock"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolatedConfig(t *testing.T) Config {
	t.Helper()
	for _, name := range []string{"PRIVATE_KEY", "TYPESAFE_API_KEY", "JEV_API_KEY", "PULSE_STAMP_LIVE", "PULSE_LIVE_STAMP", "KRONOS_SIDECAR_URL"} {
		t.Setenv(name, "")
	}
	return Config{AgentID: "restart-test", DataDir: t.TempDir(), FakePnL: 1.5}
}

func TestRestartRestoresCooldown(t *testing.T) {
	cfg := isolatedConfig(t)
	first, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if first.Record.FinalAction != "buy" {
		t.Fatal("first run should fill")
	}
	second, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if second.Record.FinalAction != "hold" || second.Outcome.Reason != "cooldown" {
		t.Fatalf("restart bypassed cooldown: %+v", second.Outcome)
	}
	if findStep(second.Record, "jev").ModelVersion != "skipped" {
		t.Fatal("hard-blocked trade must not call model")
	}
}

func TestRestartRestoresDailyLossEvenAfterStampFailure(t *testing.T) {
	cfg := isolatedConfig(t)
	cfg.FakePnL = -60
	cfg.Live = true // no key: the simulated fill/outcome must survive stamp failure
	if _, err := Run(context.Background(), cfg); err == nil {
		t.Fatal("expected no-key error")
	}
	cfg.Live = false
	second, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if second.Record.FinalAction != "hold" || second.Outcome.Reason != "daily loss limit" {
		t.Fatalf("loss was forgotten: %+v", second.Outcome)
	}
}

func TestCorruptHistoryFailsClosed(t *testing.T) {
	cfg := isolatedConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.DataDir, "outcomes.jsonl"), []byte("partial-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "risk history") {
		t.Fatalf("expected closed failure: %v", err)
	}
}

func TestConcurrentDataDirectoryRejected(t *testing.T) {
	cfg := isolatedConfig(t)
	unlock, err := runlock.Acquire(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := Run(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("expected exclusive lock: %v", err)
	}
}

func TestInterruptedOutcomeWriteRestoresLossFromAudit(t *testing.T) {
	cfg := isolatedConfig(t)
	cfg.FakePnL = -60
	if _, err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	// Simulate a crash after the durable decision but before the outcome append.
	if err := os.Remove(filepath.Join(cfg.DataDir, "outcomes.jsonl")); err != nil {
		t.Fatal(err)
	}
	res, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome.Reason != "daily loss limit" {
		t.Fatalf("interrupted loss forgotten: %+v", res.Outcome)
	}
}
