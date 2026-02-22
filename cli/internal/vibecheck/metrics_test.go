package vibecheck

import (
	"testing"
	"time"
)

// helper to create a TimelineEvent with minimal fields.
func makeTestEvent(sha, message string, ts time.Time) TimelineEvent {
	return TimelineEvent{
		SHA:          sha,
		Message:      message,
		Timestamp:    ts,
		Author:       "test-author",
		FilesChanged: 1,
		Insertions:   10,
		Deletions:    2,
	}
}

func baseTime() time.Time {
	return time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
}

// --- Velocity Tests ---

func TestMetricVelocity_Empty(t *testing.T) {
	m := MetricVelocity(nil)
	if m.Value != 0 {
		t.Errorf("expected 0 for empty events, got %f", m.Value)
	}
	if m.Passed {
		t.Error("expected not passed for empty events")
	}
}

func TestMetricVelocity_HighPace(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: one", base),
		makeTestEvent("a2", "feat: two", base.Add(1*time.Hour)),
		makeTestEvent("a3", "feat: three", base.Add(2*time.Hour)),
		makeTestEvent("a4", "feat: four", base.Add(3*time.Hour)),
		makeTestEvent("a5", "feat: five", base.Add(4*time.Hour)),
	}
	m := MetricVelocity(events)
	if !m.Passed {
		t.Errorf("expected passed for 5 commits in one day, got value=%f", m.Value)
	}
	if m.Value < 3 {
		t.Errorf("expected velocity >= 3, got %f", m.Value)
	}
}

func TestMetricVelocity_LowPace(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: one", base),
		makeTestEvent("a2", "feat: two", base.Add(48*time.Hour)),
	}
	m := MetricVelocity(events)
	if m.Passed {
		t.Errorf("expected not passed for 2 commits over 2 days, got value=%f", m.Value)
	}
}

// --- Rework Tests ---

func TestMetricRework_Empty(t *testing.T) {
	m := MetricRework(nil)
	if m.Value != 0 {
		t.Errorf("expected 0, got %f", m.Value)
	}
	if !m.Passed {
		t.Error("expected passed for empty events (0% rework)")
	}
}

func TestMetricRework_LowRework(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: add login", base),
		makeTestEvent("a2", "feat: add dashboard", base.Add(time.Hour)),
		makeTestEvent("a3", "feat: add settings", base.Add(2*time.Hour)),
		makeTestEvent("a4", "fix: login edge case", base.Add(3*time.Hour)),
	}
	m := MetricRework(events)
	if !m.Passed {
		t.Errorf("expected passed for 25%% rework, got value=%f", m.Value)
	}
}

func TestMetricRework_HighRework(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "fix: broken auth", base),
		makeTestEvent("a2", "fix: broken auth again", base.Add(time.Hour)),
		makeTestEvent("a3", "fix: still broken", base.Add(2*time.Hour)),
		makeTestEvent("a4", "feat: finally works", base.Add(3*time.Hour)),
	}
	m := MetricRework(events)
	if m.Passed {
		t.Errorf("expected not passed for 75%% rework, got value=%f", m.Value)
	}
}

// --- Trust Tests ---

func TestMetricTrust_Empty(t *testing.T) {
	m := MetricTrust(nil)
	if m.Value != 0 {
		t.Errorf("expected 0, got %f", m.Value)
	}
	if m.Passed {
		t.Error("expected not passed for empty events")
	}
}

func TestMetricTrust_WithTests(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: add login", base),
		makeTestEvent("a2", "test: add login tests", base.Add(time.Hour)),
		makeTestEvent("a3", "feat: add dashboard", base.Add(2*time.Hour)),
		makeTestEvent("a4", "test: add dashboard tests", base.Add(3*time.Hour)),
	}
	m := MetricTrust(events)
	// 2 test commits / 2 code commits = 1.0 ratio
	if !m.Passed {
		t.Errorf("expected passed with ratio=1.0, got value=%f", m.Value)
	}
}

func TestMetricTrust_NoTests(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: add login", base),
		makeTestEvent("a2", "feat: add dashboard", base.Add(time.Hour)),
		makeTestEvent("a3", "feat: add settings", base.Add(2*time.Hour)),
	}
	m := MetricTrust(events)
	if m.Passed {
		t.Errorf("expected not passed with 0 test commits, got value=%f", m.Value)
	}
	if m.Value != 0 {
		t.Errorf("expected value=0, got %f", m.Value)
	}
}

// --- Spirals Tests ---

func TestMetricSpirals_Empty(t *testing.T) {
	m := MetricSpirals(nil)
	if m.Value != 0 {
		t.Errorf("expected 0, got %f", m.Value)
	}
	if !m.Passed {
		t.Error("expected passed for empty events")
	}
}

func TestMetricSpirals_NoSpirals(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: add login", base),
		makeTestEvent("a2", "fix: typo in readme", base.Add(time.Hour)),
		makeTestEvent("a3", "feat: add dashboard", base.Add(2*time.Hour)),
		makeTestEvent("a4", "fix: dashboard color", base.Add(3*time.Hour)),
	}
	m := MetricSpirals(events)
	if !m.Passed {
		t.Errorf("expected passed with no spirals, got value=%f", m.Value)
	}
}

func TestMetricSpirals_WithSpiral(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): wrong token", base),
		makeTestEvent("a2", "fix(auth): still wrong", base.Add(10*time.Minute)),
		makeTestEvent("a3", "fix(auth): try again", base.Add(20*time.Minute)),
	}
	m := MetricSpirals(events)
	if m.Passed {
		t.Errorf("expected not passed with spiral, got value=%f", m.Value)
	}
	if m.Value < 1 {
		t.Errorf("expected at least 1 spiral, got %f", m.Value)
	}
}

// --- Flow Tests ---

func TestMetricFlow_Empty(t *testing.T) {
	m := MetricFlow(nil)
	if m.Value != 0 {
		t.Errorf("expected 0, got %f", m.Value)
	}
	if m.Passed {
		t.Error("expected not passed for empty events")
	}
}

func TestMetricFlow_EvenDistribution(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: day1", base),
		makeTestEvent("a2", "feat: day1b", base.Add(2*time.Hour)),
		makeTestEvent("a3", "feat: day2", base.Add(24*time.Hour)),
		makeTestEvent("a4", "feat: day2b", base.Add(26*time.Hour)),
		makeTestEvent("a5", "feat: day3", base.Add(48*time.Hour)),
		makeTestEvent("a6", "feat: day3b", base.Add(50*time.Hour)),
	}
	m := MetricFlow(events)
	if !m.Passed {
		t.Errorf("expected passed for even distribution, got value=%f", m.Value)
	}
}

func TestMetricFlow_SpikyDistribution(t *testing.T) {
	base := baseTime()
	// 10 commits on day 1, 0 on day 2, 0 on day 3
	var events []TimelineEvent
	for i := 0; i < 10; i++ {
		events = append(events, makeTestEvent("a"+string(rune('0'+i)), "feat: work",
			base.Add(time.Duration(i)*time.Hour)))
	}
	// Add one commit 3 days later to create the gap.
	events = append(events, makeTestEvent("b1", "feat: late", base.Add(72*time.Hour)))
	m := MetricFlow(events)
	if m.Passed {
		t.Errorf("expected not passed for spiky distribution, got value=%f", m.Value)
	}
}

// --- Aggregator Tests ---

func TestComputeMetrics_ReturnsAllFive(t *testing.T) {
	base := baseTime()
	events := []TimelineEvent{
		makeTestEvent("a1", "feat: one", base),
		makeTestEvent("a2", "feat: two", base.Add(time.Hour)),
		makeTestEvent("a3", "test: three", base.Add(2*time.Hour)),
	}
	metrics := ComputeMetrics(events)

	expected := []string{"velocity", "rework", "trust", "spirals", "flow"}
	for _, name := range expected {
		if _, ok := metrics[name]; !ok {
			t.Errorf("missing metric %q", name)
		}
	}

	if len(metrics) != 5 {
		t.Errorf("expected 5 metrics, got %d", len(metrics))
	}
}

func TestComputeOverallRating_AllPass(t *testing.T) {
	metrics := map[string]Metric{
		"velocity": {Name: "velocity", Value: 5, Threshold: 3, Passed: true},
		"rework":   {Name: "rework", Value: 10, Threshold: 30, Passed: true},
		"trust":    {Name: "trust", Value: 0.5, Threshold: 0.3, Passed: true},
		"spirals":  {Name: "spirals", Value: 0, Threshold: 0, Passed: true},
		"flow":     {Name: "flow", Value: 80, Threshold: 50, Passed: true},
	}
	score, grade := ComputeOverallRating(metrics)
	if score != 100 {
		t.Errorf("expected score 100, got %f", score)
	}
	if grade != "A" {
		t.Errorf("expected grade A, got %s", grade)
	}
}

func TestComputeOverallRating_AllFail(t *testing.T) {
	metrics := map[string]Metric{
		"velocity": {Name: "velocity", Value: 0, Threshold: 3, Passed: false},
		"rework":   {Name: "rework", Value: 90, Threshold: 30, Passed: false},
		"trust":    {Name: "trust", Value: 0, Threshold: 0.3, Passed: false},
		"spirals":  {Name: "spirals", Value: 3, Threshold: 0, Passed: false},
		"flow":     {Name: "flow", Value: 0, Threshold: 50, Passed: false},
	}
	score, grade := ComputeOverallRating(metrics)
	if score >= 40 {
		t.Errorf("expected score < 40, got %f", score)
	}
	_ = grade // Grade depends on exact partial credit
}

func TestComputeOverallRating_Empty(t *testing.T) {
	score, grade := ComputeOverallRating(nil)
	if score != 0 {
		t.Errorf("expected 0, got %f", score)
	}
	if grade != "F" {
		t.Errorf("expected F, got %s", grade)
	}
}

func TestFormatMetricsSummary(t *testing.T) {
	metrics := ComputeMetrics([]TimelineEvent{
		makeTestEvent("a1", "feat: one", baseTime()),
	})
	score, grade := ComputeOverallRating(metrics)
	summary := FormatMetricsSummary(metrics, score, grade)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}
