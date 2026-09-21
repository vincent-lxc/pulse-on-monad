package data

import "time"

// Quote is a CMC-style observation (architecture.md Perceive).
type Quote struct {
	Symbol      string    `json:"symbol"`
	SymbolID    uint64    `json:"symbol_id"`
	PriceUSD    float64   `json:"price_usd"`
	Volume24h   float64   `json:"volume_24h"`
	Change24hPct float64  `json:"change_24h_pct"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}

// DemoQuote is the canned K-line used by `pulse demo`.
func DemoQuote() Quote {
	return Quote{
		Symbol:       "BTC",
		SymbolID:     1,
		PriceUSD:     64210,
		Volume24h:    2.1e10,
		Change24hPct: 2.10,
		Timestamp:    time.Unix(1_700_000_000, 0).UTC(),
		Source:       "mock",
	}
}
