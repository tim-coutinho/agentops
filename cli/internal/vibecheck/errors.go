package vibecheck

import "errors"

// Sentinel errors for the vibecheck package.
var (
	// ErrRepoPathRequired is returned when Analyze is called without a RepoPath.
	ErrRepoPathRequired = errors.New("RepoPath is required")
)
