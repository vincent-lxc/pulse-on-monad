package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vincent-lxc/pulse-on-monad/agent/erc8004"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/audit"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/data"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/executor"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/jev"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/kronos"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/outcome"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/planner"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/policy"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/risk"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/runlock"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/stamp"
)

type Config struct {
	AgentID string
	DataDir string
	FakePnL float64
	// Live broadcasts stamp() on Monad. Default false (dry-run / CI).
	Live bool
	// ForceJev requires a TypeSafe/Jev key and will not silently stub.
	ForceJev bool
	// Jev overrides NewFromEnv (tests inject a mock HTTP client).
	Jev *jev.SoftGate
}

type Result struct {
	Record        audit.Record
	Stamp         *stamp.Result
	Outcome       outcome.Outcome
	WeightsBefore map[string]float64
	WeightsAfter  map[string]float64
	RiskBefore    policy.RiskLimits
	RiskAfter     policy.RiskLimits
	Picked        planner.Candidate
}

func DefaultConfig() Config {
	dir := os.Getenv("PULSE_DATA_DIR")
	if dir == "" {
		dir = "./data"
	}
	id := os.Getenv("PULSE_AGENT_ID")
	if id == "" {
		id = "pulse-demo"
	}
	return Config{AgentID: id, DataDir: dir, FakePnL: 1.50, Live: stamp.WantLiveFromEnv()}
}

func newRunID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "run-" + hex.EncodeToString(b[:])
}

// Run executes one simulated decision → audit → stamp (dry-run or live) → fake outcome → weight update.
func Run(ctx context.Context, cfg Config) (*Result, error) {
	unlock, err := runlock.Acquire(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer unlock()
	polStore := policy.NewStore(filepath.Join(cfg.DataDir, "policy.json"))
	st, err := polStore.Load()
	if err != nil {
		return nil, err
	}
	beforeW := copyWeights(st.Weights)
	beforeRisk := st.Risk

	auditStore, err := audit.NewStore(filepath.Join(cfg.DataDir, "decisions.jsonl"))
	if err != nil {
		return nil, err
	}
	outStore, err := outcome.NewStore(filepath.Join(cfg.DataDir, "outcomes.jsonl"))
	if err != nil {
		return nil, err
	}

	q := data.DemoQuote()
	feat := kronos.Observe()
	pl := planner.ConservativePlanner{State: st}
	cands := pl.Propose(q, feat)
	picked := pl.Pick(cands)
	intent := picked.Intent(q.Symbol)

	eng := risk.New(st.Risk)
	now := time.Now().UTC()
	if err := eng.Restore(cfg.DataDir, now); err != nil {
		return nil, err
	}
	// Hard risk is code-only and MUST run before the optional Jev soft gate.
	verdict := eng.Check(intent, now)

	recent, err := outStore.Recent(5)
	if err != nil {
		return nil, err
	}
	gate := cfg.Jev
	if gate == nil {
		gate = jev.NewFromEnv()
	}
	if cfg.ForceJev && !gate.Enabled() {
		return nil, fmt.Errorf("demo --jev requires TYPESAFE_API_KEY or JEV_API_KEY")
	}
	soft := jev.Verdict{Source: "skipped", ModelID: "hard-risk", Reason: "hard risk blocked"}
	if verdict.Allow {
		soft = gate.Evaluate(ctx, intent, recent)
	}

	final := intent
	if !verdict.Allow {
		final.Action = "hold"
		final.SizeUSD = 0
	}
	if verdict.Allow && !soft.Pass {
		final.Action = "hold"
		final.SizeUSD = 0
	}

	fill := executor.SimExecutor{}.Execute(final, verdict, q, now)

	rec := audit.NewRecord(newRunID(), cfg.AgentID)
	obsHash, _ := audit.HashInputs(q)
	kronosHash, _ := audit.HashInputs(feat)
	planHash, _ := audit.HashInputs(cands)
	riskHash, _ := audit.HashInputs(intent)
	jevHash, _ := audit.HashInputs(map[string]any{
		"model_id": soft.ModelID,
		"source":   soft.Source,
		"state":    soft.Prompt,
	})

	rec.Steps = []audit.Step{
		{Name: "observe", ModelID: "kline", ModelVersion: "mock-1", InputsHash: obsHash, Outputs: audit.MustRaw(q)},
		{Name: "kronos", ModelID: "kronos", ModelVersion: feat.Source, InputsHash: kronosHash, Outputs: audit.MustRaw(feat)},
		{Name: "plan", ModelID: "conservative-planner", ModelVersion: "1", InputsHash: planHash, Outputs: audit.MustRaw(map[string]any{"picked": picked, "candidates": cands})},
		{Name: "risk", ModelID: "hard-risk", ModelVersion: "code", InputsHash: riskHash, Outputs: audit.MustRaw(verdict), Risk: &audit.RiskIO{Pass: verdict.Allow, Reason: verdict.Reason}},
		{Name: "jev", ModelID: soft.ModelID, ModelVersion: soft.Source, InputsHash: jevHash, Outputs: audit.MustRaw(soft.AuditSafe()), Jev: jevIO(soft)},
		{Name: "execute", ModelID: "sim-executor", ModelVersion: "1", InputsHash: riskHash, Outputs: audit.MustRaw(fill)},
	}
	rec.FinalAction = final.Action
	rec.ActionCode = audit.ParseAction(final.Action)
	if verdict.Allow && soft.Pass {
		rec.SizeHint = uint64(verdict.Size)
	}
	rec.SymbolID = q.SymbolID
	o := outcome.Outcome{RunID: rec.RunID, Reason: "demo fake fill", RecordedAt: now.Unix()}
	switch {
	case !verdict.Allow:
		o.Kind = outcome.KindGateBlock
		o.Gate = "risk"
		o.Reason = verdict.Reason
	case !soft.Pass:
		o.Kind = outcome.KindGateBlock
		o.Gate = "jev"
		o.Reason = soft.Reason
	default:
		o.Kind = outcome.KindPnL
		o.PnL = cfg.FakePnL
		o.Reason = "demo fake pnl"
	}
	rec.Steps = append(rec.Steps, audit.Step{Name: "outcome", ModelID: "demo-fake-pnl", ModelVersion: "1", Outputs: audit.MustRaw(o)})

	if err := rec.Seal(); err != nil {
		return nil, err
	}
	if err := auditStore.Append(&rec); err != nil {
		return nil, err
	}

	if err := outStore.Write(&o); err != nil {
		return nil, err
	}

	st.ApplyOutcome(picked.Rule, o.Label, nil)
	if err := polStore.Save(st); err != nil {
		return nil, err
	}

	hash, err := rec.Hash()
	if err != nil {
		return nil, err
	}
	stampCfg, err := stamp.FromEnv()
	if err != nil {
		return nil, err
	}
	// Demo default is dry-run. Broadcast only when --live or PULSE_STAMP_LIVE is set.
	stampCfg.DryRun = !cfg.Live
	if cfg.Live {
		if stampCfg.PrivateKey == nil {
			return nil, fmt.Errorf("live stamp: PRIVATE_KEY is required (testnet key only; never commit it)")
		}
		if stampCfg.RPC == "" || (stampCfg.Contract == [20]byte{}) {
			return nil, fmt.Errorf("live stamp: MONAD_RPC_URL and PULSE_TRADE_STAMP are required")
		}
	}
	stampRes, err := stampCfg.Stamp(ctx, stamp.Request{
		DecisionHash: hash,
		SymbolID:     new(big.Int).SetUint64(rec.SymbolID),
		SizeHint:     new(big.Int).SetUint64(rec.SizeHint),
		Action:       rec.ActionCode,
		Note:         stamp.NoteForAgent(cfg.AgentID, rec.RunID),
	})
	if err != nil {
		return nil, err
	}

	return &Result{
		Record:        rec,
		Stamp:         stampRes,
		Outcome:       o,
		WeightsBefore: beforeW,
		WeightsAfter:  copyWeights(st.Weights),
		RiskBefore:    beforeRisk,
		RiskAfter:     st.Risk,
		Picked:        picked,
	}, nil
}

func copyWeights(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func boolStr(v bool) string {
	if v {
		return "pass"
	}
	return "block"
}

func Format(res *Result) string {
	var b strings.Builder
	fmt.Fprintln(&b, "╔══════════════════════════════════════════════════════╗")
	fmt.Fprintln(&b, "║  Pulse on Monad — one-shot decision demo             ║")
	fmt.Fprintln(&b, "╚══════════════════════════════════════════════════════╝")
	fmt.Fprintf(&b, "\n1. Observe    BTC  +2.10%%  $64,210  (mock K-line)\n")
	fmt.Fprintf(&b, "2. Kronos     sidecar off (passthrough; not online-trained)\n")
	fmt.Fprintf(&b, "3. Candidate  %-16s weight=%.4f  action=%s  size=%.0f\n",
		res.Picked.Rule, res.Picked.Weight, res.Picked.Action, res.Picked.SizeUSD)
	riskStep := findStep(res.Record, "risk")
	fmt.Fprintf(&b, "4. Hard risk  %s  %s\n", passFail(riskStep), reasonOf(riskStep))
	jevStep := findStep(res.Record, "jev")
	fmt.Fprintf(&b, "5. Jev        %s  %s  %s\n", jevSource(jevStep), passFail(jevStep), jevReason(jevStep))
	fmt.Fprintf(&b, "6. Execute    sim  action=%s  size_hint=%d\n", res.Record.FinalAction, res.Record.SizeHint)
	fmt.Fprintf(&b, "7. Audit      run_id=%s\n", res.Record.RunID)
	fmt.Fprintf(&b, "              decisionHash=%s\n", res.Record.DecisionHash)
	if res.Stamp != nil && !res.Stamp.DryRun {
		fmt.Fprintf(&b, "8. Stamp      LIVE  to=%s\n", res.Stamp.To)
		fmt.Fprintf(&b, "              tx=%s\n", res.Stamp.TxHash)
		fmt.Fprintf(&b, "              explorer=%s\n", res.Stamp.ExplorerURL)
		if res.Stamp.ReceiptID != "" {
			match := "MISMATCH"
			if res.Stamp.HashMatch {
				match = "MATCH"
			}
			fmt.Fprintf(&b, "              receipt_id=%s  on-chain decisionHash %s\n", res.Stamp.ReceiptID, match)
			if res.Stamp.OnchainDecisionHash != "" {
				fmt.Fprintf(&b, "              onchain=%s\n", res.Stamp.OnchainDecisionHash)
			}
		}
		if res.Stamp.ReadbackErr != "" {
			fmt.Fprintf(&b, "              readback: %s\n", res.Stamp.ReadbackErr)
		}
	} else {
		to := ""
		if res.Stamp != nil {
			to = res.Stamp.To
		}
		fmt.Fprintf(&b, "8. Stamp      DRY-RUN  to=%s\n", to)
		if res.Stamp != nil {
			fmt.Fprintf(&b, "              calldata=%s\n", trim(res.Stamp.Calldata, 72))
		}
	}
	fmt.Fprintf(&b, "9. Outcome    kind=%s  pnl=%.2f  label=%s\n", res.Outcome.Kind, res.Outcome.PnL, res.Outcome.Label)
	fmt.Fprintf(&b, "10. Weights   %s  %.4f → %.4f   (PRIMARY loop)\n",
		res.Picked.Rule, res.WeightsBefore[res.Picked.Rule], res.WeightsAfter[res.Picked.Rule])
	fmt.Fprintf(&b, "    Hard risk unchanged: max_pos=%.0f  daily_loss=%.0f  cooldown=%ds\n",
		res.RiskAfter.MaxPositionUSD, res.RiskAfter.MaxDailyLossUSD, res.RiskAfter.CooldownSeconds)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, erc8004.FromEnv().HowToRegister())
	return b.String()
}

func findStep(r audit.Record, name string) *audit.Step {
	for i := range r.Steps {
		if r.Steps[i].Name == name {
			return &r.Steps[i]
		}
	}
	return nil
}

func passFail(s *audit.Step) string {
	if s == nil {
		return "?"
	}
	if s.Risk != nil {
		if s.Risk.Pass {
			return "PASS"
		}
		return "FAIL"
	}
	if s.Jev != nil {
		if s.Jev.Verdict == "pass" {
			return "PASS"
		}
		return "BLOCK"
	}
	return "ok"
}

func reasonOf(s *audit.Step) string {
	if s != nil && s.Risk != nil {
		return s.Risk.Reason
	}
	return ""
}

func jevReason(s *audit.Step) string {
	if s != nil && s.Jev != nil {
		return s.Jev.Reason
	}
	return ""
}

func jevSource(s *audit.Step) string {
	if s == nil {
		return "?"
	}
	if s.ModelVersion != "" {
		return s.ModelVersion
	}
	return s.ModelID
}

func jevIO(v jev.Verdict) *audit.JevIO {
	return &audit.JevIO{
		Prompt:    v.Prompt,
		Verdict:   boolStr(v.Pass),
		Reason:    v.Reason,
		ModelID:   v.ModelID,
		Questions: audit.MustRaw(v.Questions),
		Answers:   audit.MustRaw(v.Answers),
	}
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
