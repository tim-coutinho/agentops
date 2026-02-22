package pool

import "fmt"

// Sentinel errors for common pool operations.
var (
	// ErrEmptyID is returned when a candidate ID is empty.
	ErrEmptyID = fmt.Errorf("candidate ID cannot be empty")
	// ErrIDTooLong is returned when a candidate ID exceeds 128 characters.
	ErrIDTooLong = fmt.Errorf("candidate ID too long (max 128 characters)")
	// ErrIDInvalidChars is returned when a candidate ID contains disallowed characters.
	ErrIDInvalidChars = fmt.Errorf("candidate ID contains invalid characters (only alphanumeric, hyphen, underscore allowed)")
	// ErrCandidateNotFound is returned when a candidate cannot be located in the pool.
	ErrCandidateNotFound = fmt.Errorf("candidate not found")
	// ErrStageRejected is returned when attempting to stage a rejected candidate.
	ErrStageRejected = fmt.Errorf("cannot stage rejected candidate")
	// ErrPromoteRejected is returned when attempting to promote a rejected candidate.
	ErrPromoteRejected = fmt.Errorf("cannot promote rejected candidate")
	// ErrNotStaged is returned when attempting to promote a candidate that is not staged.
	ErrNotStaged = fmt.Errorf("candidate must be staged before promotion")
	// ErrThresholdTooLow is returned when bulk approval threshold is below minimum.
	ErrThresholdTooLow = fmt.Errorf("threshold must be >= 1h")
	// ErrReasonTooLong is returned when reason/note exceeds MaxReasonLength.
	ErrReasonTooLong = fmt.Errorf("reason/note exceeds maximum length of %d characters", MaxReasonLength)
)
