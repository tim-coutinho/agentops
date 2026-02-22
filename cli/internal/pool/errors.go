package pool

import (
	"errors"
	"fmt"
)

// Sentinel errors for common pool operations.
var (
	// ErrEmptyID is returned when a candidate ID is empty.
	ErrEmptyID = errors.New("candidate ID cannot be empty")
	// ErrIDTooLong is returned when a candidate ID exceeds 128 characters.
	ErrIDTooLong = errors.New("candidate ID too long (max 128 characters)")
	// ErrIDInvalidChars is returned when a candidate ID contains disallowed characters.
	ErrIDInvalidChars = errors.New("candidate ID contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	// ErrCandidateNotFound is returned when a candidate cannot be located in the pool.
	ErrCandidateNotFound = errors.New("candidate not found")
	// ErrStageRejected is returned when attempting to stage a rejected candidate.
	ErrStageRejected = errors.New("cannot stage rejected candidate")
	// ErrPromoteRejected is returned when attempting to promote a rejected candidate.
	ErrPromoteRejected = errors.New("cannot promote rejected candidate")
	// ErrNotStaged is returned when attempting to promote a candidate that is not staged.
	ErrNotStaged = errors.New("candidate must be staged before promotion")
	// ErrThresholdTooLow is returned when bulk approval threshold is below minimum.
	ErrThresholdTooLow = errors.New("threshold must be >= 1h")
	// ErrReasonTooLong is returned when reason/note exceeds MaxReasonLength.
	ErrReasonTooLong = fmt.Errorf("reason/note exceeds maximum length of %d characters", MaxReasonLength)
)
