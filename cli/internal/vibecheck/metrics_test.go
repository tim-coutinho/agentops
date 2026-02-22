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

func TestComputeOverallRating_NonStandardMetricCount(t *testing.T) {
	// Use only 3 metrics instead of 5 to exercise the normalization path
	metrics := map[string]Metric{
		"velocity": {Name: "velocity", Value: 5, Threshold: 3, Passed: true},
		"trust":    {Name: "trust", Value: 0.5, Threshold: 0.3, Passed: true},
		"flow":     {Name: "flow", Value: 80, Threshold: 50, Passed: true},
	}
	score, grade := ComputeOverallRating(metrics)
	if score != 100 {
		t.Errorf("expected score 100 for all-passing metrics, got %f", score)
	}
	if grade != "A" {
		t.Errorf("expected grade A, got %s", grade)
	}
}

func TestMetricPartialCredit_ReworkHighValue(t *testing.T) {
	// Rework with value > threshold (100%) -- ratio should be negative, clamped to 0
	m := Metric{Name: "rework", Value: 100, Threshold: 30, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 0 {
		t.Errorf("expected 0 credit for 100%% rework, got %f", credit)
	}
}

func TestMetricPartialCredit_ReworkModerateValue(t *testing.T) {
	// Rework at 50% with threshold 30%: ratio = (100-50)/(100-30) = 50/70 = ~0.714
	m := Metric{Name: "rework", Value: 50, Threshold: 30, Passed: false}
	credit := metricPartialCredit(m)
	expected := 50.0 / 70.0 * 20
	if credit < expected-0.1 || credit > expected+0.1 {
		t.Errorf("expected credit ~%.1f, got %f", expected, credit)
	}
}

func TestMetricPartialCredit_ReworkBelowThreshold(t *testing.T) {
	// Rework below threshold should get full 20 points
	m := Metric{Name: "rework", Value: 10, Threshold: 30, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 20 {
		t.Errorf("expected 20 credit for rework below threshold, got %f", credit)
	}
}

func TestMetricPartialCredit_VelocityPartial(t *testing.T) {
	// Velocity halfway to threshold: value=1.5, threshold=3 -> ratio 0.5 -> credit 10
	m := Metric{Name: "velocity", Value: 1.5, Threshold: 3, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 10 {
		t.Errorf("expected 10 credit for velocity at half threshold, got %f", credit)
	}
}

func TestMetricPartialCredit_VelocityAboveThreshold(t *testing.T) {
	// Value > threshold but somehow not passed: ratio clamped to 1
	m := Metric{Name: "velocity", Value: 5, Threshold: 3, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 20 {
		t.Errorf("expected 20 credit (clamped), got %f", credit)
	}
}

func TestMetricPartialCredit_ZeroThreshold(t *testing.T) {
	// Zero threshold (like spirals) -- if not passed, value>0, so no credit
	m := Metric{Name: "spirals", Value: 2, Threshold: 0, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 0 {
		t.Errorf("expected 0 credit for zero threshold, got %f", credit)
	}
}

func TestMetricPartialCredit_UnknownMetric(t *testing.T) {
	m := Metric{Name: "unknown-metric", Value: 5, Threshold: 10, Passed: false}
	credit := metricPartialCredit(m)
	if credit != 0 {
		t.Errorf("expected 0 credit for unknown metric, got %f", credit)
	}
}

func TestMetricPartialCredit_TrustAndFlow(t *testing.T) {
	// Trust partial credit
	m := Metric{Name: "trust", Value: 0.15, Threshold: 0.3, Passed: false}
	credit := metricPartialCredit(m)
	expected := 0.15 / 0.3 * 20 // = 10
	if credit < expected-0.1 || credit > expected+0.1 {
		t.Errorf("expected trust credit ~%.1f, got %f", expected, credit)
	}

	// Flow partial credit
	m = Metric{Name: "flow", Value: 25, Threshold: 50, Passed: false}
	credit = metricPartialCredit(m)
	expected = 25.0 / 50.0 * 20 // = 10
	if credit < expected-0.1 || credit > expected+0.1 {
		t.Errorf("expected flow credit ~%.1f, got %f", expected, credit)
	}
}

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{100, "A"},
		{80, "A"},
		{79, "B"},
		{60, "B"},
		{59, "C"},
		{40, "C"},
		{39, "D"},
		{20, "D"},
		{19, "F"},
		{0, "F"},
	}
	for _, tt := range tests {
		got := scoreToGrade(tt.score)
		if got != tt.want {
			t.Errorf("scoreToGrade(%f) = %s, want %s", tt.score, got, tt.want)
		}
	}
}

func TestCountSpirals_DifferentComponents(t *testing.T) {
	base := baseTime()
	// Fix commits on different components should NOT form a spiral
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): token issue", base),
		makeTestEvent("a2", "fix(db): connection pool", base.Add(10*time.Minute)),
		makeTestEvent("a3", "fix(auth): another auth issue", base.Add(20*time.Minute)),
	}
	count := countSpirals(events)
	if count != 0 {
		t.Errorf("expected 0 spirals for different components, got %d", count)
	}
}

func TestCountSpirals_ChainBrokenByFeat(t *testing.T) {
	base := baseTime()
	// 3 fix commits interrupted by feat
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): issue 1", base),
		makeTestEvent("a2", "fix(auth): issue 2", base.Add(10*time.Minute)),
		makeTestEvent("a3", "feat: new feature", base.Add(20*time.Minute)),
		makeTestEvent("a4", "fix(auth): issue 3", base.Add(30*time.Minute)),
	}
	count := countSpirals(events)
	if count != 0 {
		t.Errorf("expected 0 spirals (chain broken by feat), got %d", count)
	}
}

func TestCountSpirals_ExactlyThree(t *testing.T) {
	base := baseTime()
	// Exactly 3 fix commits on same component followed by non-fix
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): issue 1", base),
		makeTestEvent("a2", "fix(auth): issue 2", base.Add(10*time.Minute)),
		makeTestEvent("a3", "fix(auth): issue 3", base.Add(20*time.Minute)),
		makeTestEvent("a4", "feat: done", base.Add(30*time.Minute)),
	}
	count := countSpirals(events)
	if count != 1 {
		t.Errorf("expected 1 spiral for exactly 3 fix commits, got %d", count)
	}
}

func TestCountSpirals_EndOfEventsFlush(t *testing.T) {
	base := baseTime()
	// 3 fix commits at end of events should flush as spiral
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): issue 1", base),
		makeTestEvent("a2", "fix(auth): issue 2", base.Add(10*time.Minute)),
		makeTestEvent("a3", "fix(auth): issue 3", base.Add(20*time.Minute)),
	}
	count := countSpirals(events)
	if count != 1 {
		t.Errorf("expected 1 spiral at end of events, got %d", count)
	}
}

func TestCountSpirals_ComponentSwitchFlush(t *testing.T) {
	base := baseTime()
	// 3 fix on auth, then switch to db (should flush auth spiral)
	events := []TimelineEvent{
		makeTestEvent("a1", "fix(auth): issue 1", base),
		makeTestEvent("a2", "fix(auth): issue 2", base.Add(10*time.Minute)),
		makeTestEvent("a3", "fix(auth): issue 3", base.Add(20*time.Minute)),
		makeTestEvent("a4", "fix(db): different component", base.Add(30*time.Minute)),
	}
	count := countSpirals(events)
	if count != 1 {
		t.Errorf("expected 1 spiral (flushed on component switch), got %d", count)
	}
}

func TestExtractComponent_Variations(t *testing.T) {
	tests := []struct {
		msg  string
		want string
	}{
		{"fix(auth): token issue", "auth"},
		{"fix: broken login page", "broken"},
		{"fix typo in readme", "typo"},
		{"fix(): empty scope", "fix()"},
		{"fix a b c", "unknown"}, // all short words
	}
	for _, tt := range tests {
		got := extractComponent(tt.msg)
		if got != tt.want {
			t.Errorf("extractComponent(%q) = %q, want %q", tt.msg, got, tt.want)
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-5, "-5"},
		{100, "100"},
	}
	for _, tt := range tests {
		got := itoa(tt.n)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatMetricsSummary_MissingMetric(t *testing.T) {
	// Should handle missing metrics gracefully (just skip them)
	metrics := map[string]Metric{
		"velocity": {Name: "velocity", Value: 5, Threshold: 3, Passed: true},
	}
	summary := FormatMetricsSummary(metrics, 20, "D")
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}
