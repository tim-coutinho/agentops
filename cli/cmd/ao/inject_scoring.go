package main

import "math"

// freshnessScore calculates decay-adjusted score: exp(-ageWeeks * decayRate)
// Based on knowledge decay rate δ = 0.17/week (Darr et al.)
func freshnessScore(ageWeeks float64) float64 {
	const decayRate = 0.17
	score := math.Exp(-ageWeeks * decayRate)
	// Clamp to [0.1, 1.0] - old knowledge still has some value
	if score < 0.1 {
		return 0.1
	}
	return score
}

// applyCompositeScoring implements MemRL Two-Phase scoring.
// Score = z_norm(freshness) + λ × z_norm(utility)
// This combines recency (Phase A) with learned utility (Phase B).
func applyCompositeScoring(learnings []learning, lambda float64) {
	if len(learnings) == 0 {
		return
	}

	// Calculate means and standard deviations for z-normalization
	var sumF, sumU float64
	for _, l := range learnings {
		sumF += l.FreshnessScore
		sumU += l.Utility
	}
	n := float64(len(learnings))
	meanF := sumF / n
	meanU := sumU / n

	// Calculate standard deviations
	var varF, varU float64
	for _, l := range learnings {
		varF += (l.FreshnessScore - meanF) * (l.FreshnessScore - meanF)
		varU += (l.Utility - meanU) * (l.Utility - meanU)
	}
	stdF := math.Sqrt(varF / n)
	stdU := math.Sqrt(varU / n)

	// Avoid division by zero - use minimum of 0.001
	if stdF < 0.001 {
		stdF = 0.001
	}
	if stdU < 0.001 {
		stdU = 0.001
	}

	// Apply z-normalization and calculate composite scores
	for i := range learnings {
		zFresh := (learnings[i].FreshnessScore - meanF) / stdF
		zUtility := (learnings[i].Utility - meanU) / stdU

		// Composite score: z_norm(freshness) + λ × z_norm(utility)
		learnings[i].CompositeScore = zFresh + lambda*zUtility
	}
}
