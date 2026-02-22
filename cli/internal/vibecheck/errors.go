package vibecheck

import "fmt"

// Sentinel errors for the vibecheck package.
var (
	// ErrRepoPathRequired is returned when Analyze is called without a RepoPath.
	ErrRepoPathRequired = fmt.Errorf("RepoPath is required")
)
