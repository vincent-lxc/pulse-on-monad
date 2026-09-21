package planner

import (
	"fmt"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/data"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/kronos"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/policy"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/risk"
)

// Candidate is a rule firing on the current observation.
type Candidate struct {
	Rule    string
	Action  string
	SizeUSD float64
	Reason  string
	Weight  float64
}

// ConservativePlanner is the architecture.md planner: explicit reasons,
// never invents a fill, defers to risk.Engine.
type ConservativePlanner struct {
	State policy.State
}

func (p ConservativePlanner) Propose(obs data.Quote, feat kronos.Features) []Candidate {
	var out []Candidate
	w := p.State.Weights
	size := 100.0
	if p.State.Risk.MaxPositionUSD > 0 && size > p.State.Risk.MaxPositionUSD {
		size = p.State.Risk.MaxPositionUSD
	}

	// Momentum: ride a clear 24h move. Optional Kronos score can reinforce.
	if obs.Change24hPct >= 1.5 || feat.Score >= 0.6 {
		out = append(out, Candidate{
			Rule:    "momentum",
			Action:  "buy",
			SizeUSD: size,
			Reason:  fmt.Sprintf("change_24h=%.2f kronos=%.2f", obs.Change24hPct, feat.Score),
			Weight:  w["momentum"],
		})
	}
	if obs.Change24hPct <= -1.5 || feat.Score <= -0.6 {
		out = append(out, Candidate{
			Rule:    "momentum",
			Action:  "sell",
			SizeUSD: size,
			Reason:  fmt.Sprintf("change_24h=%.2f kronos=%.2f", obs.Change24hPct, feat.Score),
			Weight:  w["momentum"],
		})
	}

	// Mean reversion on an extended move.
	if obs.Change24hPct >= 8 {
		out = append(out, Candidate{
			Rule:    "mean_reversion",
			Action:  "sell",
			SizeUSD: size,
			Reason:  "extended up-move, fade",
			Weight:  w["mean_reversion"],
		})
	}
	if obs.Change24hPct <= -8 {
		out = append(out, Candidate{
			Rule:    "mean_reversion",
			Action:  "buy",
			SizeUSD: size,
			Reason:  "extended down-move, fade",
			Weight:  w["mean_reversion"],
		})
	}

	// Conservative always available as a hold.
	out = append(out, Candidate{
		Rule:    "conservative",
		Action:  "hold",
		SizeUSD: 0,
		Reason:  "default hold",
		Weight:  w["conservative"],
	})
	return out
}

func (p ConservativePlanner) Pick(cands []Candidate) Candidate {
	best := cands[0]
	for _, c := range cands[1:] {
		if c.Weight > best.Weight {
			best = c
		}
	}
	return best
}

func (c Candidate) Intent(symbol string) risk.Intent {
	return risk.Intent{
		Symbol:  symbol,
		Action:  c.Action,
		SizeUSD: c.SizeUSD,
		Rule:    c.Rule,
		Reason:  c.Reason,
	}
}
