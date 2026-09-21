package jev

import (
	"strings"
	"testing"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/outcome"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/risk"
)

func TestOutcomeSummaryInjected(t *testing.T) {
	g := &SoftGate{}
	recent := []outcome.Outcome{{
		RunID:  "r1",
		Kind:   outcome.KindPnL,
		PnL:    1.2,
		Label:  outcome.LabelPositive,
		Reason: "sim",
	}}
	v := g.Evaluate(risk.Intent{Symbol: "BTC", Action: "buy", SizeUSD: 100, Rule: "momentum"}, recent)
	if !v.Pass {
		t.Fatal("stub must pass")
	}
	if !strings.Contains(v.Prompt, "run=r1") || !strings.Contains(v.Prompt, "label=positive") {
		t.Fatalf("prompt missing outcome summary:\n%s", v.Prompt)
	}
	if v.Enabled {
		t.Fatal("empty key must disable live Jev")
	}
}
