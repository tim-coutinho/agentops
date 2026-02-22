package rpi

import "fmt"

// Sentinel errors for the rpi package. Using sentinels instead of ad-hoc
// fmt.Errorf allows callers to match with errors.Is for reliable error handling.
var (
	// ErrDetachedHEAD is returned when a worktree operation requires a named branch
	// but the repository is in detached HEAD state.
	ErrDetachedHEAD = fmt.Errorf("detached HEAD: worktree requires a named branch")

	// ErrDetachedSelfHealFailed is returned when the automatic recovery from
	// detached HEAD state fails.
	ErrDetachedSelfHealFailed = fmt.Errorf("detached HEAD self-heal failed")

	// ErrNotGitRepo is returned when a command is run outside a git repository.
	ErrNotGitRepo = fmt.Errorf("not a git repository (run ao rpi phased from inside a git repo)")

	// ErrResolveHEAD is returned when HEAD commit cannot be resolved.
	ErrResolveHEAD = fmt.Errorf("unable to resolve HEAD commit for detached worktree creation")

	// ErrWorktreeCollision is returned after 3 failed attempts to create a
	// unique worktree path.
	ErrWorktreeCollision = fmt.Errorf("failed to create unique worktree path after 3 attempts")

	// ErrMergeSourceUnavailable is returned when neither worktree path nor
	// run ID is provided for a merge operation.
	ErrMergeSourceUnavailable = fmt.Errorf("merge source unavailable: missing worktree path and run ID")

	// ErrRepoUnclean is returned when the repository has uncommitted changes
	// that persist after multiple retries.
	ErrRepoUnclean = fmt.Errorf("original repo has uncommitted changes after 5 retries: commit or stash before merge")

	// ErrEmptyMergeSource is returned when the worktree merge source commit
	// resolves to an empty string.
	ErrEmptyMergeSource = fmt.Errorf("worktree merge source commit is empty")
)
