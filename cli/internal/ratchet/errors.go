package ratchet

import "fmt"

// Sentinel errors for the ratchet package. Using sentinels instead of ad-hoc
// fmt.Errorf allows callers to match with errors.Is for reliable error handling.
var (
	// ErrChainNoPath is returned when a chain operation is attempted without a path set.
	ErrChainNoPath = fmt.Errorf("chain has no path set")

	// ErrEmptyLearningFile is returned when a learning file has no content.
	ErrEmptyLearningFile = fmt.Errorf("empty learning file")

	// ErrEmptyFile is returned when a file is unexpectedly empty.
	ErrEmptyFile = fmt.Errorf("empty file")

	// ErrAgentsDirNotFound is returned when no .agents directory is found.
	ErrAgentsDirNotFound = fmt.Errorf(".agents directory not found")

	// ErrNoRigRoot is returned when no rig root directory can be found.
	ErrNoRigRoot = fmt.Errorf("no rig root found")
)
