package main

import (
	"encoding/json"
	"testing"
)

// Tests for doctor.go helper functions

func TestCountInstalledEvents(t *testing.T) {
	t.Run("empty map returns 0", func(t *testing.T) {
		got := countInstalledEvents(map[string]interface{}{})
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("SessionStart with 1 group counts as 1", func(t *testing.T) {
		hooksMap := map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{"command": "ao context"},
			},
		}
		got := countInstalledEvents(hooksMap)
		if got < 1 {
			t.Errorf("expected at least 1, got %d", got)
		}
	})

	t.Run("empty slice for event not counted", func(t *testing.T) {
		hooksMap := map[string]interface{}{
			"SessionStart": []interface{}{},
		}
		got := countInstalledEvents(hooksMap)
		if got != 0 {
			t.Errorf("expected 0 for empty slice, got %d", got)
		}
	})
}

func TestExtractHooksMap(t *testing.T) {
	t.Run("settings.json format with hooks key", func(t *testing.T) {
		data, _ := json.Marshal(map[string]interface{}{
			"hooks": map[string]interface{}{
				"SessionStart": []interface{}{},
			},
		})
		got, ok := extractHooksMap(data)
		if !ok {
			t.Fatal("expected ok=true for settings.json format")
		}
		if got == nil {
			t.Error("expected non-nil hooks map")
		}
	})

	t.Run("hooks.json format with top-level events", func(t *testing.T) {
		hooksMap := map[string]interface{}{
			"SessionStart": []interface{}{},
		}
		data, _ := json.Marshal(hooksMap)
		got, ok := extractHooksMap(data)
		if !ok {
			t.Fatal("expected ok=true for hooks.json format")
		}
		if got == nil {
			t.Error("expected non-nil hooks map")
		}
	})

	t.Run("invalid JSON returns false", func(t *testing.T) {
		_, ok := extractHooksMap([]byte("{invalid"))
		if ok {
			t.Error("expected ok=false for invalid JSON")
		}
	})

	t.Run("JSON with no hooks or events returns false", func(t *testing.T) {
		data, _ := json.Marshal(map[string]interface{}{
			"unrelated": "value",
		})
		_, ok := extractHooksMap(data)
		if ok {
			t.Error("expected ok=false for JSON with no hooks/events")
		}
	})
}

func TestEvaluateHookCoverage(t *testing.T) {
	t.Run("empty hooks map returns warn", func(t *testing.T) {
		result := evaluateHookCoverage(map[string]interface{}{})
		if result.Status != "warn" {
			t.Errorf("expected warn status, got %q", result.Status)
		}
	})

	t.Run("has events but no ao command returns warn", func(t *testing.T) {
		// Create a hooks map with a non-ao command for SessionStart
		hooksMap := map[string]interface{}{}
		for _, event := range AllEventNames() {
			hooksMap[event] = []interface{}{
				map[string]interface{}{
					"command": "echo hello",
				},
			}
		}
		result := evaluateHookCoverage(hooksMap)
		if result.Status != "warn" {
			t.Errorf("expected warn for non-ao hooks, got %q", result.Status)
		}
	})

	t.Run("has ao SessionStart but partial coverage returns warn", func(t *testing.T) {
		hooksMap := map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{"command": "ao context status"},
					},
				},
			},
			// Only 1 of many events
		}
		result := evaluateHookCoverage(hooksMap)
		if result.Status == "pass" {
			t.Error("expected non-pass for partial hook coverage")
		}
	})

	t.Run("full coverage returns pass", func(t *testing.T) {
		// Build a complete hooks map with ao commands for all events
		hooksMap := map[string]interface{}{}
		for _, event := range AllEventNames() {
			hooksMap[event] = []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{"command": "ao context status"},
					},
				},
			}
		}
		result := evaluateHookCoverage(hooksMap)
		if result.Status != "pass" {
			t.Errorf("expected pass for full coverage, got %q (detail: %s)", result.Status, result.Detail)
		}
	})
}
