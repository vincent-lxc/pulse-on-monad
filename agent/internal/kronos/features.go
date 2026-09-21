package kronos

import "os"

// Features is an optional sidecar forecast. This repo does not online-train
// Kronos; when no sidecar URL is configured we pass through a zero score.
type Features struct {
	Horizon string  `json:"horizon"`
	Score   float64 `json:"score"`
	Source  string  `json:"source"` // "off" | "sidecar"
}

func Observe() Features {
	if os.Getenv("KRONOS_SIDECAR_URL") == "" {
		return Features{Horizon: "1h", Score: 0, Source: "off"}
	}
	// Shadow/sidecar hook — keep offline for CI. A future client can fill Score.
	return Features{Horizon: "1h", Score: 0, Source: "sidecar"}
}
