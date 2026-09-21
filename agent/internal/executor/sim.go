package executor

import (
	"fmt"
	"time"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/data"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/risk"
)

// Fill is a paper fill at mark (architecture.md SimExecutor).
type Fill struct {
	Symbol    string    `json:"symbol"`
	Action    string    `json:"action"`
	SizeUSD   float64   `json:"size_usd"`
	PriceUSD  float64   `json:"price_usd"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	FilledAt  time.Time `json:"filled_at"`
}

type SimExecutor struct{}

func (SimExecutor) Execute(in risk.Intent, v risk.Verdict, q data.Quote, now time.Time) Fill {
	if in.Action == "hold" || v.Size == 0 {
		return Fill{
			Symbol:   q.Symbol,
			Action:   "hold",
			Status:   "noop",
			Reason:   v.Reason,
			FilledAt: now,
			PriceUSD: q.PriceUSD,
		}
	}
	if !v.Allow {
		return Fill{
			Symbol:   q.Symbol,
			Action:   in.Action,
			Status:   "rejected",
			Reason:   v.Reason,
			FilledAt: now,
			PriceUSD: q.PriceUSD,
		}
	}
	return Fill{
		Symbol:   q.Symbol,
		Action:   in.Action,
		SizeUSD:  v.Size,
		PriceUSD: q.PriceUSD,
		Status:   "filled",
		Reason:   fmt.Sprintf("sim @ %.2f", q.PriceUSD),
		FilledAt: now,
	}
}
