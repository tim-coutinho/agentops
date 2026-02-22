package goals

import "sort"

// DriftResult describes how a single goal changed between two snapshots.
type DriftResult struct {
	GoalID     string   `json:"goal_id"`
	Before     string   `json:"before"`
	After      string   `json:"after"`
	Delta      string   `json:"delta"`       // "improved", "regressed", "unchanged"
	ValueDelta *float64 `json:"value_delta,omitempty"`
	Weight     int      `json:"weight"`
}

// ComputeDrift compares a baseline snapshot against a current snapshot and
// returns a DriftResult per goal in the current snapshot. Results are sorted
// with regressions first (by weight descending), then improvements, then unchanged.
func ComputeDrift(baseline, current *Snapshot) []DriftResult {
	baseMap := make(map[string]Measurement, len(baseline.Goals))
	for _, m := range baseline.Goals {
		baseMap[m.GoalID] = m
	}

	var results []DriftResult
	for _, cur := range current.Goals {
		dr := DriftResult{
			GoalID: cur.GoalID,
			After:  cur.Result,
			Weight: cur.Weight,
		}

		base, found := baseMap[cur.GoalID]
		if !found {
			dr.Before = "new"
			dr.Delta = "unchanged"
		} else {
			dr.Before = base.Result
			switch {
			case base.Result == "fail" && cur.Result == "pass":
				dr.Delta = "improved"
			case base.Result == "pass" && cur.Result == "fail":
				dr.Delta = "regressed"
			default:
				dr.Delta = "unchanged"
			}
			if base.Value != nil && cur.Value != nil {
				vd := *cur.Value - *base.Value
				dr.ValueDelta = &vd
			}
		}

		results = append(results, dr)
	}

	sort.SliceStable(results, func(i, j int) bool {
		ri, rj := deltaRank(results[i].Delta), deltaRank(results[j].Delta)
		if ri != rj {
			return ri < rj
		}
		return results[i].Weight > results[j].Weight
	})

	return results
}

// deltaRank returns a sort key: regressed=0, improved=1, unchanged=2.
func deltaRank(delta string) int {
	switch delta {
	case "regressed":
		return 0
	case "improved":
		return 1
	default:
		return 2
	}
}
