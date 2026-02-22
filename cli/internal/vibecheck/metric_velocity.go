package vibecheck

// MetricVelocity computes development pace as commits per day.
// Threshold: 3+ commits/day = good (passed).
func MetricVelocity(events []TimelineEvent) Metric {
	if len(events) == 0 {
		return Metric{
			Name:      "velocity",
			Value:     0,
			Threshold: 3,
			Passed:    false,
		}
	}

	// Find the time span covered by events.
	minT, maxT := events[0].Timestamp, events[0].Timestamp
	for _, e := range events[1:] {
		if e.Timestamp.Before(minT) {
			minT = e.Timestamp
		}
		if e.Timestamp.After(maxT) {
			maxT = e.Timestamp
		}
	}

	// Calculate the number of calendar days spanned (at least 1).
	days := maxT.Sub(minT).Hours() / 24.0
	if days < 1 {
		days = 1
	}

	velocity := float64(len(events)) / days

	// Round to 1 decimal place.
	velocity = float64(int(velocity*10+0.5)) / 10.0

	return Metric{
		Name:      "velocity",
		Value:     velocity,
		Threshold: 3,
		Passed:    velocity >= 3,
	}
}

