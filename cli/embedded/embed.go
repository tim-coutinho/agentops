// Package embedded provides hooks, scripts, and skill files embedded in the ao binary.
// These are used as a fallback when the agentops repo checkout is not available
// (e.g., Homebrew or npx installs).
package embedded

import "embed"

// HooksJSON contains the raw hooks.json configuration.
//
//go:embed hooks/hooks.json
var HooksJSON []byte

// HooksFS contains all embedded hook scripts, lib helpers, and skill files.
// Use fs.WalkDir to extract files to disk.
//
//go:embed all:hooks all:lib all:skills
var HooksFS embed.FS
