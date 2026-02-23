package main

import (
	"github.com/spf13/cobra"
)

var rpiCmd = &cobra.Command{
	Use:   "rpi",
	Short: "RPI lifecycle automation",
	Long: `Commands for automating the RPI (Research-Plan-Implement) lifecycle.

Commands:
  loop       Run continuous RPI cycles from next-work queue

The RPI loop reads .agents/rpi/next-work.jsonl for harvested work items
and spawns fresh Claude sessions for each cycle (Ralph Wiggum pattern).`,
}

func init() {
	rpiCmd.GroupID = "workflow"
	rootCmd.AddCommand(rpiCmd)
}
