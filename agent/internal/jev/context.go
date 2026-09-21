package jev

import (
	"fmt"
	"os"
	"strings"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/outcome"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/risk"
)

// Verdict is the optional LLM soft gate. Hard risk has already run.
type Verdict struct {
	Pass    bool   `json:"pass"`
	Reason  string `json:"reason"`
	Prompt  string `json:"prompt"`
	Enabled bool   `json:"enabled"`
}

// SoftGate injects a recent-outcome summary into the next prompt.
// Without JEV_API_KEY this is a stub that always passes (CI-safe).
type SoftGate struct {
	APIKey string
}

func NewFromEnv() *SoftGate {
	return &SoftGate{APIKey: strings.TrimSpace(os.Getenv("JEV_API_KEY"))}
}

func (g *SoftGate) Enabled() bool { return g != nil && g.APIKey != "" }

// OutcomeSummary is the helper the user asked for: a compact, prompt-ready
// recap of recent labeled outcomes for the next soft-gate turn.
func OutcomeSummary(recent []outcome.Outcome) string {
	if len(recent) == 0 {
		return "no prior outcomes"
	}
	var b strings.Builder
	b.WriteString("recent outcomes (oldest→newest):\n")
	for _, o := range recent {
		fmt.Fprintf(&b, "- run=%s label=%s kind=%s pnl=%.4f gate=%s reason=%s\n",
			o.RunID, o.Label, o.Kind, o.PnL, o.Gate, o.Reason)
	}
	return strings.TrimSpace(b.String())
}

func (g *SoftGate) BuildPrompt(in risk.Intent, recent []outcome.Outcome) string {
	return fmt.Sprintf(
		"Soft-gate a Pulse trade intent.\naction=%s symbol=%s size_usd=%.2f rule=%s\nplanner_reason=%s\n\n%s\n\nReply PASS or BLOCK with one sentence.",
		in.Action, in.Symbol, in.SizeUSD, in.Rule, in.Reason, OutcomeSummary(recent),
	)
}

func (g *SoftGate) Evaluate(in risk.Intent, recent []outcome.Outcome) Verdict {
	prompt := g.BuildPrompt(in, recent)
	if !g.Enabled() {
		return Verdict{
			Pass:    true,
			Reason:  "jev_disabled_stub",
			Prompt:  prompt,
			Enabled: false,
		}
	}
	// Live Jev HTTP is intentionally not wired: no API contract is pinned and
	// CI must not require a key. The prompt is ready for a future client.
	return Verdict{
		Pass:    true,
		Reason:  "jev_key_present_passthrough",
		Prompt:  prompt,
		Enabled: true,
	}
}
