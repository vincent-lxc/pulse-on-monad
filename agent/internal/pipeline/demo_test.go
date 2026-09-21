package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/jev"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/stamp"
)

func TestDemoLoopDryRun(t *testing.T) {
	t.Setenv("PULSE_STAMP_LIVE", "")
	t.Setenv("PULSE_LIVE_STAMP", "")
	t.Setenv("PRIVATE_KEY", "")
	t.Setenv("TYPESAFE_API_KEY", "")
	t.Setenv("JEV_API_KEY", "")
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

func TestForceJevWithoutKeyFails(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	t.Setenv("JEV_API_KEY", "")
	cfg := Config{AgentID: "pulse-test", DataDir: t.TempDir(), FakePnL: 1.5, ForceJev: true}
	_, err := Run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "TYPESAFE_API_KEY") {
		t.Fatalf("expected key error, got %v", err)
	}
}

func TestPipelineUsesMockJevAfterHardRisk(t *testing.T) {
	var sawState string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		sawState = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"jev-1.13.0","answers":{"allow_action":{"type":"noul","noul":0.88},"confidence":{"type":"score","score":3}}}`))
	}))
	defer srv.Close()

	cfg := Config{
		AgentID: "pulse-test",
		DataDir: t.TempDir(),
		FakePnL: 1.5,
		Jev:     &jev.SoftGate{Endpoint: srv.URL, APIKey: "mock-key", HTTP: srv.Client()},
	}
	res, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sawState, `"model":"jev-latest"`) {
		t.Fatalf("expected systemone body, got %s", sawState)
	}
	riskIdx, jevIdx := -1, -1
	for i, s := range res.Record.Steps {
		if s.Name == "risk" {
			riskIdx = i
		}
		if s.Name == "jev" {
			jevIdx = i
		}
	}
	if riskIdx < 0 || jevIdx < 0 || riskIdx > jevIdx {
		t.Fatalf("hard risk must run before Jev: risk=%d jev=%d", riskIdx, jevIdx)
	}
	jevStep := findStep(res.Record, "jev")
	if jevStep.ModelID != "jev-1.13.0" || jevStep.Jev == nil || jevStep.Jev.Verdict != "pass" {
		t.Fatalf("audit jev %+v", jevStep)
	}
	if strings.Contains(string(jevStep.Outputs), "mock-key") {
		t.Fatal("audit leaked API key")
	}
	blob, _ := json.Marshal(res.Record)
	if strings.Contains(string(blob), "mock-key") {
		t.Fatal("record leaked API key")
	}
}
