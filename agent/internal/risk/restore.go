package risk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vincent-lxc/pulse-on-monad/agent/internal/audit"
	"github.com/vincent-lxc/pulse-on-monad/agent/internal/outcome"
)

// Restore replays durable fills and outcomes before each risk check. UTC days
// bound gross realized losses; gains do not replenish the loss budget. Limits
// remain in policy.json. Corrupt or duplicate history fails closed.
func (e *Engine) Restore(dir string, now time.Time) error {
	e.LastFillAt = map[string]time.Time{}
	e.DailyLoss = 0
	seen := map[string]bool{}
	auditOutcomes := map[string]outcome.Outcome{}
	if err := readLines(filepath.Join(dir, "decisions.jsonl"), func(b []byte) error {
		var r audit.Record
		if err := json.Unmarshal(b, &r); err != nil {
			return err
		}
		if r.RunID == "" || seen[r.RunID] {
			return fmt.Errorf("missing or duplicate decision run_id")
		}
		seen[r.RunID] = true
		h, err := r.HashHex()
		if err != nil || h != r.DecisionHash || h != r.ContentHash {
			return fmt.Errorf("decision hash mismatch")
		}
		for _, step := range r.Steps {
			if step.Name == "outcome" {
				var o outcome.Outcome
				if err := json.Unmarshal(step.Outputs, &o); err != nil {
					return err
				}
				if o.RunID != r.RunID {
					return fmt.Errorf("outcome run mismatch")
				}
				auditOutcomes[r.RunID] = o
			}
			if step.Name != "execute" {
				continue
			}
			var fill struct {
				Symbol   string    `json:"symbol"`
				Status   string    `json:"status"`
				FilledAt time.Time `json:"filled_at"`
			}
			if err := json.Unmarshal(step.Outputs, &fill); err != nil {
				return err
			}
			if fill.Status == "filled" {
				if fill.Symbol == "" || fill.FilledAt.IsZero() {
					return fmt.Errorf("invalid fill")
				}
				symbol := strings.ToUpper(fill.Symbol)
				if fill.FilledAt.After(e.LastFillAt[symbol]) {
					e.LastFillAt[symbol] = fill.FilledAt
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	seen = map[string]bool{}
	applyOutcome := func(o outcome.Outcome) error {
		if o.RecordedAt <= 0 {
			return fmt.Errorf("invalid outcome timestamp")
		}
		if _, err := outcome.Classify(o.Kind, o.PnL); err != nil {
			return err
		}
		recorded := time.Unix(o.RecordedAt, 0).UTC()
		if recorded.After(now.Add(time.Second)) {
			return fmt.Errorf("future outcome timestamp")
		}
		if o.Kind == outcome.KindPnL && o.PnL < 0 && recorded.Format("2006-01-02") == now.UTC().Format("2006-01-02") {
			e.DailyLoss -= o.PnL
		}
		return nil
	}
	if err := readLines(filepath.Join(dir, "outcomes.jsonl"), func(b []byte) error {
		var o outcome.Outcome
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		if o.RunID == "" || seen[o.RunID] || o.RecordedAt <= 0 {
			return fmt.Errorf("invalid or duplicate outcome")
		}
		seen[o.RunID] = true
		if audited, ok := auditOutcomes[o.RunID]; ok {
			if audited.Kind != o.Kind || audited.PnL != o.PnL || audited.RecordedAt != o.RecordedAt {
				return fmt.Errorf("outcome conflicts with audit")
			}
			delete(auditOutcomes, o.RunID)
		}
		return applyOutcome(o)
	}); err != nil {
		return err
	}
	for _, o := range auditOutcomes {
		if err := applyOutcome(o); err != nil {
			return err
		}
	}
	return nil
}

func readLines(path string, apply func([]byte) error) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 65536), 4*1024*1024)
	line := 0
	for scan.Scan() {
		line++
		if err := apply(scan.Bytes()); err != nil {
			return fmt.Errorf("risk history %s:%d: %w", filepath.Base(path), line, err)
		}
	}
	return scan.Err()
}
