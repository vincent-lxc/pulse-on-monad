package risk

import (
	"encoding/json"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/outcome"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/policy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestoreGrossDailyLossUTCRollover(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
	rows := []outcome.Outcome{
		{RunID: "old", Kind: "pnl", PnL: -100, RecordedAt: now.Add(-24 * time.Hour).Unix()},
		{RunID: "loss1", Kind: "pnl", PnL: -30, RecordedAt: now.Add(-time.Hour).Unix()},
		{RunID: "gain", Kind: "pnl", PnL: 100, RecordedAt: now.Add(-time.Minute).Unix()},
		{RunID: "loss2", Kind: "pnl", PnL: -25, RecordedAt: now.Unix()},
	}
	f, err := os.Create(filepath.Join(dir, "outcomes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if err := json.NewEncoder(f).Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	e := New(policy.DefaultLimits())
	if err := e.Restore(dir, now); err != nil {
		t.Fatal(err)
	}
	if e.DailyLoss != 55 {
		t.Fatalf("loss=%v", e.DailyLoss)
	}
	if e.Check(Intent{Symbol: "BTC", Action: "buy", SizeUSD: 100}, now).Allow {
		t.Fatal("loss must block")
	}
	if err := e.Restore(dir, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if e.DailyLoss != 0 {
		t.Fatalf("UTC day did not roll: %v", e.DailyLoss)
	}
}

func TestRestoreRejectsDuplicateOutcomes(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	row := outcome.Outcome{RunID: "same", Kind: "pnl", PnL: -1, RecordedAt: now.Unix()}
	b, _ := json.Marshal(row)
	b = append(b, '\n')
	b = append(b, b...)
	os.WriteFile(filepath.Join(dir, "outcomes.jsonl"), b, 0600)
	if err := New(policy.DefaultLimits()).Restore(dir, now); err == nil {
		t.Fatal("duplicate history accepted")
	}
}
