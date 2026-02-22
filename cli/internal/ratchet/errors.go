package ratchet

import "errors"

// Sentinel errors for the ratchet package. Using sentinels instead of ad-hoc
// fmt.Errorf allows callers to match with errors.Is for reliable error handling.
var (
	// ErrChainNoPath is returned when a chain operation is attempted without a path set.
	ErrChainNoPath = errors.New("chain has no path set")

	// ErrEmptyLearningFile is returned when a learning file has no content.
	ErrEmptyLearningFile = errors.New("empty learning file")

	// ErrEmptyFile is returned when a file is unexpectedly empty.
	ErrEmptyFile = errors.New("empty file")

	// ErrAgentsDirNotFound is returned when no .agents directory is found.
	ErrAgentsDirNotFound = errors.New(".agents directory not found")

	// ErrNoRigRoot is returned when no rig root directory can be found.
	ErrNoRigRoot = errors.New("no rig root found")
)
