package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollectOLConstraints_NoOLDir verifies that collectOLConstraints returns nil
// when the .ol/ directory doesn't exist (no-op test).
func TestCollectOLConstraints_NoOLDir(t *testing.T) {
	tmpDir := t.TempDir()

	// No .ol/ directory exists
	constraints, err := collectOLConstraints(tmpDir, "")

	if err != nil {
		t.Errorf("collectOLConstraints() error = %v, want nil", err)
	}
	if constraints != nil {
		t.Errorf("collectOLConstraints() = %v, want nil (no-op)", constraints)
	}
}

// TestCollectOLConstraints_NoQuarantineFile verifies that collectOLConstraints
// returns nil when .ol/ exists but quarantine.json doesn't.
func TestCollectOLConstraints_NoQuarantineFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ol/ directory but no quarantine.json
	olDir := filepath.Join(tmpDir, ".ol", "constraints")
	if err := os.MkdirAll(olDir, 0755); err != nil {
		t.Fatal(err)
	}

	constraints, err := collectOLConstraints(tmpDir, "")

	if err != nil {
		t.Errorf("collectOLConstraints() error = %v, want nil", err)
	}
	if constraints != nil {
		t.Errorf("collectOLConstraints() = %v, want nil (no quarantine file)", constraints)
	}
}

// TestCollectOLConstraints_ReadConstraints verifies that collectOLConstraints
// reads constraints from .ol/constraints/quarantine.json when present.
func TestCollectOLConstraints_ReadConstraints(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ol/constraints/quarantine.json
	olDir := filepath.Join(tmpDir, ".ol", "constraints")
	if err := os.MkdirAll(olDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConstraints := []olConstraint{
		{
			Pattern:    "no-exposed-secrets",
			Detection:  "API keys or tokens in code",
			Source:     "security-audit",
			Confidence: 0.95,
			Status:     "active",
		},
		{
			Pattern:    "require-error-handling",
			Detection:  "Unchecked error returns",
			Source:     "code-review",
			Confidence: 0.85,
			Status:     "active",
		},
	}

	data, err := json.MarshalIndent(testConstraints, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	quarantinePath := filepath.Join(olDir, "quarantine.json")
	if err := os.WriteFile(quarantinePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Collect constraints without query
	constraints, err := collectOLConstraints(tmpDir, "")

	if err != nil {
		t.Errorf("collectOLConstraints() error = %v, want nil", err)
	}
	if len(constraints) != 2 {
		t.Errorf("collectOLConstraints() returned %d constraints, want 2", len(constraints))
	}
	if constraints[0].Pattern != "no-exposed-secrets" {
		t.Errorf("constraints[0].Pattern = %q, want %q", constraints[0].Pattern, "no-exposed-secrets")
	}
	if constraints[1].Pattern != "require-error-handling" {
		t.Errorf("constraints[1].Pattern = %q, want %q", constraints[1].Pattern, "require-error-handling")
	}
}

// TestCollectOLConstraints_FilterByQuery verifies that collectOLConstraints
// filters constraints by query string.
func TestCollectOLConstraints_FilterByQuery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ol/constraints/quarantine.json
	olDir := filepath.Join(tmpDir, ".ol", "constraints")
	if err := os.MkdirAll(olDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConstraints := []olConstraint{
		{
			Pattern:    "no-exposed-secrets",
			Detection:  "Security issue: API keys or tokens in code",
			Source:     "security-audit",
			Confidence: 0.95,
		},
		{
			Pattern:    "require-error-handling",
			Detection:  "Unchecked error returns",
			Source:     "code-review",
			Confidence: 0.85,
		},
		{
			Pattern:    "authentication-required",
			Detection:  "Security issue: Endpoints without auth checks",
			Source:     "security-audit",
			Confidence: 0.90,
		},
	}

	data, err := json.MarshalIndent(testConstraints, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	quarantinePath := filepath.Join(olDir, "quarantine.json")
	if err := os.WriteFile(quarantinePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantFirst string
	}{
		{
			name:      "filter by 'security'",
			query:     "security",
			wantCount: 2,
			wantFirst: "no-exposed-secrets",
		},
		{
			name:      "filter by 'error'",
			query:     "error",
			wantCount: 1,
			wantFirst: "require-error-handling",
		},
		{
			name:      "filter by 'authentication'",
			query:     "authentication",
			wantCount: 1,
			wantFirst: "authentication-required",
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantCount: 0,
			wantFirst: "",
		},
		{
			name:      "case insensitive",
			query:     "SECURITY",
			wantCount: 2,
			wantFirst: "no-exposed-secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints, err := collectOLConstraints(tmpDir, tt.query)

			if err != nil {
				t.Errorf("collectOLConstraints() error = %v, want nil", err)
			}
			if len(constraints) != tt.wantCount {
				t.Errorf("collectOLConstraints(%q) returned %d constraints, want %d",
					tt.query, len(constraints), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if len(constraints) == 0 {
					t.Errorf("expected at least one constraint, got none")
				} else if constraints[0].Pattern != tt.wantFirst {
					t.Errorf("constraints[0].Pattern = %q, want %q",
						constraints[0].Pattern, tt.wantFirst)
				}
			}
		})
	}
}

// TestCollectOLConstraints_InvalidJSON verifies that collectOLConstraints
// returns an error when quarantine.json contains invalid JSON.
func TestCollectOLConstraints_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ol/constraints/quarantine.json with invalid JSON
	olDir := filepath.Join(tmpDir, ".ol", "constraints")
	if err := os.MkdirAll(olDir, 0755); err != nil {
		t.Fatal(err)
	}

	quarantinePath := filepath.Join(olDir, "quarantine.json")
	if err := os.WriteFile(quarantinePath, []byte("invalid json {"), 0644); err != nil {
		t.Fatal(err)
	}

	constraints, err := collectOLConstraints(tmpDir, "")

	if err == nil {
		t.Error("collectOLConstraints() expected error for invalid JSON, got nil")
	}
	if constraints != nil {
		t.Errorf("collectOLConstraints() = %v, want nil on error", constraints)
	}
}

// TestFormatKnowledgeMarkdown_WithConstraints verifies that formatKnowledgeMarkdown
// includes the Olympus Constraints section when constraints are present.
func TestFormatKnowledgeMarkdown_WithConstraints(t *testing.T) {
	knowledge := &injectedKnowledge{
		OLConstraints: []olConstraint{
			{
				Pattern:   "no-exposed-secrets",
				Detection: "API keys or tokens in code",
			},
			{
				Pattern:   "require-error-handling",
				Detection: "Unchecked error returns",
			},
		},
	}

	output := formatKnowledgeMarkdown(knowledge)

	// Verify Olympus Constraints section exists
	if !strings.Contains(output, "### Olympus Constraints") {
		t.Error("formatKnowledgeMarkdown() missing '### Olympus Constraints' header")
	}

	// Verify constraint patterns are included
	if !strings.Contains(output, "no-exposed-secrets") {
		t.Error("formatKnowledgeMarkdown() missing 'no-exposed-secrets' constraint")
	}
	if !strings.Contains(output, "require-error-handling") {
		t.Error("formatKnowledgeMarkdown() missing 'require-error-handling' constraint")
	}

	// Verify detection strings are included
	if !strings.Contains(output, "API keys or tokens in code") {
		t.Error("formatKnowledgeMarkdown() missing detection string for no-exposed-secrets")
	}
	if !strings.Contains(output, "Unchecked error returns") {
		t.Error("formatKnowledgeMarkdown() missing detection string for require-error-handling")
	}

	// Verify olympus constraint marker
	if !strings.Contains(output, "[olympus constraint]") {
		t.Error("formatKnowledgeMarkdown() missing '[olympus constraint]' marker")
	}
}

// TestFormatKnowledgeMarkdown_NoConstraints verifies that formatKnowledgeMarkdown
// does NOT include the Olympus Constraints section when no constraints are present.
func TestFormatKnowledgeMarkdown_NoConstraints(t *testing.T) {
	knowledge := &injectedKnowledge{
		OLConstraints: nil,
	}

	output := formatKnowledgeMarkdown(knowledge)

	// Verify Olympus Constraints section does NOT exist
	if strings.Contains(output, "### Olympus Constraints") {
		t.Error("formatKnowledgeMarkdown() should not include '### Olympus Constraints' when empty")
	}
	if strings.Contains(output, "[olympus constraint]") {
		t.Error("formatKnowledgeMarkdown() should not include '[olympus constraint]' marker when empty")
	}

	// Should show "No prior knowledge found" message
	if !strings.Contains(output, "*No prior knowledge found.*") {
		t.Error("formatKnowledgeMarkdown() should show 'No prior knowledge found' when empty")
	}
}

// TestFormatKnowledgeMarkdown_EmptyConstraints verifies that formatKnowledgeMarkdown
// does NOT include the Olympus Constraints section when constraints slice is empty.
func TestFormatKnowledgeMarkdown_EmptyConstraints(t *testing.T) {
	knowledge := &injectedKnowledge{
		OLConstraints: []olConstraint{},
	}

	output := formatKnowledgeMarkdown(knowledge)

	// Verify Olympus Constraints section does NOT exist
	if strings.Contains(output, "### Olympus Constraints") {
		t.Error("formatKnowledgeMarkdown() should not include '### Olympus Constraints' when empty")
	}
}

// TestFormatKnowledgeMarkdown_MixedContent verifies that formatKnowledgeMarkdown
// correctly formats output when both learnings and OL constraints are present.
func TestFormatKnowledgeMarkdown_MixedContent(t *testing.T) {
	knowledge := &injectedKnowledge{
		Learnings: []learning{
			{
				ID:      "L42",
				Title:   "Test Learning",
				Summary: "This is a test learning",
			},
		},
		Patterns: []pattern{
			{
				Name:        "test-pattern",
				Description: "A test pattern",
			},
		},
		OLConstraints: []olConstraint{
			{
				Pattern:   "no-exposed-secrets",
				Detection: "API keys in code",
			},
		},
	}

	output := formatKnowledgeMarkdown(knowledge)

	// Verify all sections exist
	if !strings.Contains(output, "### Recent Learnings") {
		t.Error("formatKnowledgeMarkdown() missing '### Recent Learnings'")
	}
	if !strings.Contains(output, "### Active Patterns") {
		t.Error("formatKnowledgeMarkdown() missing '### Active Patterns'")
	}
	if !strings.Contains(output, "### Olympus Constraints") {
		t.Error("formatKnowledgeMarkdown() missing '### Olympus Constraints'")
	}

	// Verify content from each section
	if !strings.Contains(output, "L42") {
		t.Error("formatKnowledgeMarkdown() missing learning ID")
	}
	if !strings.Contains(output, "test-pattern") {
		t.Error("formatKnowledgeMarkdown() missing pattern name")
	}
	if !strings.Contains(output, "no-exposed-secrets") {
		t.Error("formatKnowledgeMarkdown() missing constraint pattern")
	}

	// Should NOT show "No prior knowledge found" message
	if strings.Contains(output, "*No prior knowledge found.*") {
		t.Error("formatKnowledgeMarkdown() should not show 'No prior knowledge found' when content exists")
	}
}

// TestOLConstraintMarshaling verifies that olConstraint JSON marshaling works correctly.
func TestOLConstraintMarshaling(t *testing.T) {
	constraint := olConstraint{
		Pattern:    "test-pattern",
		Detection:  "test detection",
		Source:     "test-source",
		Confidence: 0.95,
		Status:     "active",
	}

	data, err := json.Marshal(constraint)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded olConstraint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Pattern != constraint.Pattern {
		t.Errorf("decoded.Pattern = %q, want %q", decoded.Pattern, constraint.Pattern)
	}
	if decoded.Detection != constraint.Detection {
		t.Errorf("decoded.Detection = %q, want %q", decoded.Detection, constraint.Detection)
	}
	if decoded.Source != constraint.Source {
		t.Errorf("decoded.Source = %q, want %q", decoded.Source, constraint.Source)
	}
	if decoded.Confidence != constraint.Confidence {
		t.Errorf("decoded.Confidence = %f, want %f", decoded.Confidence, constraint.Confidence)
	}
	if decoded.Status != constraint.Status {
		t.Errorf("decoded.Status = %q, want %q", decoded.Status, constraint.Status)
	}
}
