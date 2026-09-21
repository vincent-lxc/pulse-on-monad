package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/stamp"
)

func TestDemoLoopDryRun(t *testing.T) {
	t.Setenv("PULSE_STAMP_LIVE", "")
	t.Setenv("PULSE_LIVE_STAMP", "")
	t.Setenv("PRIVATE_KEY", "")
	cfg := Config{AgentID: "pulse-test", DataDir: t.TempDir(), FakePnL: 1.5, Live: false}
	res, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Record.DecisionHash == "" || res.Stamp == nil || !res.Stamp.DryRun {
		t.Fatalf("incomplete result: %+v", res.Stamp)
	}
	if res.Stamp.TxHash != "" {
		t.Fatal("dry-run must not set a tx hash")
	}
	if res.Outcome.Label != "positive" {
		t.Fatalf("label %s", res.Outcome.Label)
	}
	if res.WeightsAfter[res.Picked.Rule] <= res.WeightsBefore[res.Picked.Rule] {
		t.Fatalf("expected weight increase")
	}
	if res.RiskAfter.MaxPositionUSD != res.RiskBefore.MaxPositionUSD {
		t.Fatal("hard risk loosened or mutated")
	}
	out := Format(res)
	if !strings.Contains(out, "DRY-RUN") {
		t.Fatalf("format missing DRY-RUN:\n%s", out)
	}
}

func TestLiveWithoutKeyFails(t *testing.T) {
	t.Setenv("PRIVATE_KEY", "")
	t.Setenv("PULSE_STAMP_LIVE", "")
	cfg := Config{AgentID: "pulse-test", DataDir: t.TempDir(), FakePnL: 1.5, Live: true}
	_, err := Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "PRIVATE_KEY") {
		t.Fatalf("expected PRIVATE_KEY error, got %v", err)
	}
}

func TestFormatLivePrintsExplorer(t *testing.T) {
	r := Result{
		WeightsBefore: map[string]float64{"momentum": 0.5},
		WeightsAfter:  map[string]float64{"momentum": 0.54},
		Stamp: &stamp.Result{
			To:                  stamp.DefaultContract,
			DryRun:              false,
			TxHash:              "0xabc",
			ExplorerURL:         stamp.ExplorerTxURL("0xabc"),
			ReceiptID:           "9",
			OnchainDecisionHash: "0x11",
			HashMatch:           true,
		},
	}
	r.Record.RunID = "run-x"
	r.Record.DecisionHash = "0x11"
	r.Record.FinalAction = "buy"
	r.Picked.Rule = "momentum"
	r.Picked.Action = "buy"
	r.Picked.Weight = 0.5
	out := Format(&r)
	if !strings.Contains(out, "LIVE") || !strings.Contains(out, "testnet.monadvision.com/tx/") {
		t.Fatalf("format missing live explorer:\n%s", out)
	}
	if !strings.Contains(out, "MATCH") || !strings.Contains(out, "receipt_id=9") {
		t.Fatalf("format missing readback:\n%s", out)
	}
}
