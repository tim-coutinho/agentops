package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestBuildRPIStatusOutput_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	output := buildRPIStatusOutput(dir)
	if output.Count != 0 {
		t.Errorf("expected Count=0 for empty dir, got %d", output.Count)
	}
}

func TestWriteRPIStatusJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	output := rpiStatusOutput{
		Count: 2,
	}
	err := writeRPIStatusJSON(output)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("writeRPIStatusJSON: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	var decoded rpiStatusOutput
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}
	if decoded.Count != 2 {
		t.Errorf("Count = %d, want 2", decoded.Count)
	}
}

func TestRenderRPIStatusTable_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	output := rpiStatusOutput{}

	// Should not error and should print "No active RPI runs found."
	// Capture stdout
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := renderRPIStatusTable(dir, output)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("renderRPIStatusTable with empty output: %v", err)
	}
}

func TestRenderRPIStatusTable_WithActiveRuns(t *testing.T) {
	dir := t.TempDir()
	output := rpiStatusOutput{
		Active: []rpiRunInfo{
			{RunID: "test-run", Goal: "test goal", Phase: 1, PhaseName: "research"},
		},
		Runs:  []rpiRunInfo{{RunID: "test-run", Goal: "test goal", Phase: 1, PhaseName: "research"}},
		Count: 1,
	}

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := renderRPIStatusTable(dir, output)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Errorf("renderRPIStatusTable with active runs: %v", err)
	}
}
