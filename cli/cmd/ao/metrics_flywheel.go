package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/boshu2/agentops/cli/internal/types"
	"github.com/spf13/cobra"
)

// flywheelCmd provides a convenient alias for flywheel status operations.
var flywheelCmd = &cobra.Command{
	Use:   "flywheel",
	Short: "Knowledge flywheel operations",
	Long: `Knowledge flywheel operations and status.

The flywheel equation:
  dK/dt = I(t) - δ·K + σ·ρ·K - B(K, K_crit)

Escape velocity: σρ > δ → Knowledge compounds

Commands:
  status   Show comprehensive flywheel health

Examples:
  ao flywheel status
  ao flywheel status -o json`,
}

func init() {
	rootCmd.AddCommand(flywheelCmd)

	// flywheel status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show flywheel health status",
		Long: `Display comprehensive flywheel health status.

Shows:
  - Delta (δ): Knowledge decay rate
  - Sigma (σ): Retrieval effectiveness
  - Rho (ρ): Citation rate
  - Velocity: σρ - δ (net growth rate)
  - Status: COMPOUNDING / NEAR ESCAPE / DECAYING

Examples:
  ao flywheel status
  ao flywheel status --days 30
  ao flywheel status -o json`,
		RunE: runFlywheelStatus,
	}
	statusCmd.Flags().IntVar(&metricsDays, "days", 7, "Period in days for metrics calculation")
	flywheelCmd.AddCommand(statusCmd)
}

// runFlywheelStatus displays comprehensive flywheel health.
func runFlywheelStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	metrics, err := computeMetrics(cwd, metricsDays)
	if err != nil {
		return fmt.Errorf("compute metrics: %w", err)
	}

	w := cmd.OutOrStdout()
	switch GetOutput() {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"status":      metrics.EscapeVelocityStatus(),
			"delta":       metrics.Delta,
			"sigma":       metrics.Sigma,
			"rho":         metrics.Rho,
			"sigma_rho":   metrics.SigmaRho,
			"velocity":    metrics.Velocity,
			"compounding": metrics.AboveEscapeVelocity,
			"metrics":     metrics,
		})

	case "yaml":
		enc := yaml.NewEncoder(w)
		return enc.Encode(map[string]interface{}{
			"status":      metrics.EscapeVelocityStatus(),
			"delta":       metrics.Delta,
			"sigma":       metrics.Sigma,
			"rho":         metrics.Rho,
			"sigma_rho":   metrics.SigmaRho,
			"velocity":    metrics.Velocity,
			"compounding": metrics.AboveEscapeVelocity,
		})

	default:
		printFlywheelStatus(w, metrics)
	}

	return nil
}

// printFlywheelStatus prints a focused flywheel status display.
func printFlywheelStatus(w io.Writer, m *types.FlywheelMetrics) {
	status := m.EscapeVelocityStatus()

	// Status indicator (ASCII for accessibility)
	var statusIcon string
	switch status {
	case "COMPOUNDING":
		statusIcon = "[COMPOUNDING]"
	case "NEAR ESCAPE":
		statusIcon = "[NEAR_ESCAPE]"
	default:
		statusIcon = "[DECAYING]"
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Flywheel Status: %s\n", statusIcon)
	fmt.Fprintln(w, "  ═══════════════════════════════")
	fmt.Fprintln(w)

	// Core equation
	fmt.Fprintln(w, "  EQUATION: dK/dt = I(t) - δ·K + σ·ρ·K")
	fmt.Fprintln(w)

	// Parameters
	fmt.Fprintf(w, "  δ (decay):      %.2f/week\n", m.Delta)
	fmt.Fprintf(w, "  σ (retrieval):  %.2f (%d%% of artifacts surfaced)\n", m.Sigma, int(m.Sigma*100))
	fmt.Fprintf(w, "  ρ (citation):   %.2f refs/artifact/week\n", m.Rho)
	fmt.Fprintln(w)

	// Critical comparison
	fmt.Fprintln(w, "  ESCAPE VELOCITY CHECK:")
	fmt.Fprintf(w, "    σ × ρ = %.3f\n", m.SigmaRho)
	fmt.Fprintf(w, "    δ     = %.3f\n", m.Delta)
	fmt.Fprintln(w, "    ───────────────")

	if m.AboveEscapeVelocity {
		fmt.Fprintf(w, "    σρ > δ ✓ (velocity: +%.3f/week)\n", m.Velocity)
		fmt.Fprintln(w, "    → Knowledge is COMPOUNDING")
	} else if m.Velocity > -0.05 {
		fmt.Fprintf(w, "    σρ ≈ δ (velocity: %.3f/week)\n", m.Velocity)
		fmt.Fprintln(w, "    → NEAR escape velocity, keep building!")
	} else {
		fmt.Fprintf(w, "    σρ < δ ✗ (velocity: %.3f/week)\n", m.Velocity)
		fmt.Fprintln(w, "    → Knowledge is DECAYING")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  RECOMMENDATIONS:")
		if m.Sigma < 0.3 {
			fmt.Fprintln(w, "    • Improve retrieval: run 'ao inject' more often")
		}
		if m.Rho < 0.5 {
			fmt.Fprintln(w, "    • Cite more learnings: reference artifacts in your work")
		}
		if m.StaleArtifacts > 5 {
			fmt.Fprintf(w, "    • Review %d stale artifacts (90+ days uncited)\n", m.StaleArtifacts)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Period: %s to %s (%d days)\n",
		m.PeriodStart.Format("2006-01-02"),
		m.PeriodEnd.Format("2006-01-02"),
		metricsDays)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Tip: 'ao status' shows flywheel health alongside session info.")
}
