package pipeline

import (
	"context"
	"testing"
)

func TestDemoLoop(t *testing.T) {
	cfg := Config{AgentID: "pulse-test", DataDir: t.TempDir(), FakePnL: 1.5}
	res, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Record.DecisionHash == "" || res.Stamp == nil || !res.Stamp.DryRun {
		t.Fatalf("incomplete result: %+v", res.Stamp)
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
}
