package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOutputValidateResult_Valid(t *testing.T) {
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	prevGoalsJSON := goalsJSON
	goalsJSON = false
	defer func() { goalsJSON = prevGoalsJSON }()

	result := validateResult{
		Valid:     true,
		GoalCount: 3,
		Version:   2,
		Errors:    nil,
		Warnings:  nil,
	}
	err := outputValidateResult(result)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("expected no error for valid result, got: %v", err)
	}
}

func TestOutputValidateResult_Invalid(t *testing.T) {
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	prevGoalsJSON := goalsJSON
	goalsJSON = false
	defer func() { goalsJSON = prevGoalsJSON }()

	result := validateResult{
		Valid:  false,
		Errors: []string{"goal foo: check required"},
	}
	err := outputValidateResult(result)

	w.Close()
	os.Stdout = old

	if err == nil {
		t.Fatal("expected error for invalid result")
	}
	if err.Error() != "validation failed" {
		t.Errorf("error = %q, want 'validation failed'", err.Error())
	}
}

func TestOutputValidateResult_JSONMode(t *testing.T) {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	prevGoalsJSON := goalsJSON
	goalsJSON = true
	defer func() { goalsJSON = prevGoalsJSON }()

	result := validateResult{
		Valid:     true,
		GoalCount: 5,
		Version:   3,
	}
	err := outputValidateResult(result)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputValidateResult JSON: %v", err)
	}

	tmp := make([]byte, 4096)
	n, _ := r.Read(tmp)

	var decoded validateResult
	if err := json.Unmarshal(tmp[:n], &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v (raw: %s)", err, string(tmp[:n]))
	}
	if decoded.GoalCount != 5 {
		t.Errorf("GoalCount = %d, want 5", decoded.GoalCount)
	}
	if decoded.Version != 3 {
		t.Errorf("Version = %d, want 3", decoded.Version)
	}
}
