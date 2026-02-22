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

	results := make([]DriftResult, 0, len(current.Goals))
	for _, cur := range current.Goals {
		results = append(results, computeGoalDrift(cur, baseMap))
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

// classifyDelta determines the drift direction between two results.
func classifyDelta(before, after string) string {
	switch {
	case before == "fail" && after == "pass":
		return "improved"
	case before == "pass" && after == "fail":
		return "regressed"
	default:
		return "unchanged"
	}
}

// computeValueDelta returns a pointer to the numeric difference between two
// optional values, or nil if either is nil.
func computeValueDelta(baseVal, curVal *float64) *float64 {
	if baseVal != nil && curVal != nil {
		vd := *curVal - *baseVal
		return &vd
	}
	return nil
}

// computeGoalDrift computes the drift result for a single goal measurement.
func computeGoalDrift(cur Measurement, baseMap map[string]Measurement) DriftResult {
	dr := DriftResult{
		GoalID: cur.GoalID,
		After:  cur.Result,
		Weight: cur.Weight,
	}

	base, found := baseMap[cur.GoalID]
	if !found {
		dr.Before = "new"
		dr.Delta = "unchanged"
		return dr
	}

	dr.Before = base.Result
	dr.Delta = classifyDelta(base.Result, cur.Result)
	dr.ValueDelta = computeValueDelta(base.Value, cur.Value)
	return dr
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
