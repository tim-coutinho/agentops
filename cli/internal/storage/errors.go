package storage

import "fmt"

// Sentinel errors for the storage package. Using sentinels instead of ad-hoc
// fmt.Errorf allows callers to match with errors.Is for reliable error handling.
var (
	// ErrSessionIDRequired is returned when a session write is attempted without an ID.
	ErrSessionIDRequired = fmt.Errorf("session ID is required")

	// ErrEmptySessionFile is returned when a session file has no content.
	ErrEmptySessionFile = fmt.Errorf("empty session file")
)
