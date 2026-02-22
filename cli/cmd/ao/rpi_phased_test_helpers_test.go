package main

// testPhasedStateBuilder provides a fluent builder API for constructing
// phasedState values in tests. Use newTestPhasedStateBuilder() to obtain a
// builder with sensible defaults, then chain withX() methods to override
// specific fields, and call build() to produce the final *phasedState.
type testPhasedStateBuilder struct {
	s phasedState
}

// newTestPhasedStateBuilder returns a builder pre-loaded with test-friendly
// defaults. Fields not explicitly set via withX() methods will keep these
// defaults when build() is called.
func newTestPhasedStateBuilder() *testPhasedStateBuilder {
	return &testPhasedStateBuilder{
		s: phasedState{
			SchemaVersion: 1,
			Goal:          "test goal",
			Phase:         1,
			StartPhase:    1,
			Cycle:         1,
			Verdicts:      make(map[string]string),
			Attempts:      make(map[string]int),
			StartedAt:     "2026-02-19T00:00:00Z",
		},
	}
}

// build returns the constructed *phasedState.
func (b *testPhasedStateBuilder) build() *phasedState {
	s := b.s
	// Ensure maps are never nil.
	if s.Verdicts == nil {
		s.Verdicts = make(map[string]string)
	}
	if s.Attempts == nil {
		s.Attempts = make(map[string]int)
	}
	return &s
}

// withSchemaVersion sets the SchemaVersion field.
func (b *testPhasedStateBuilder) withSchemaVersion(v int) *testPhasedStateBuilder {
	b.s.SchemaVersion = v
	return b
}

// withGoal sets the Goal field.
func (b *testPhasedStateBuilder) withGoal(g string) *testPhasedStateBuilder {
	b.s.Goal = g
	return b
}

// withEpicID sets the EpicID field.
func (b *testPhasedStateBuilder) withEpicID(id string) *testPhasedStateBuilder {
	b.s.EpicID = id
	return b
}

// withPhase sets the Phase field (and StartPhase to the same value).
func (b *testPhasedStateBuilder) withPhase(p phase) *testPhasedStateBuilder {
	b.s.Phase = p.Num
	b.s.StartPhase = p.Num
	return b
}

// withPhaseNum sets the Phase field directly by number.
func (b *testPhasedStateBuilder) withPhaseNum(n int) *testPhasedStateBuilder {
	b.s.Phase = n
	return b
}

// withStartPhase sets only the StartPhase field.
func (b *testPhasedStateBuilder) withStartPhase(n int) *testPhasedStateBuilder {
	b.s.StartPhase = n
	return b
}

// withCycle sets the Cycle field.
func (b *testPhasedStateBuilder) withCycle(c int) *testPhasedStateBuilder {
	b.s.Cycle = c
	return b
}

// withRunID sets the RunID field.
func (b *testPhasedStateBuilder) withRunID(id string) *testPhasedStateBuilder {
	b.s.RunID = id
	return b
}

// withWorktreePath sets the WorktreePath field.
func (b *testPhasedStateBuilder) withWorktreePath(p string) *testPhasedStateBuilder {
	b.s.WorktreePath = p
	return b
}

// withOrchestratorPID sets the OrchestratorPID field.
func (b *testPhasedStateBuilder) withOrchestratorPID(pid int) *testPhasedStateBuilder {
	b.s.OrchestratorPID = pid
	return b
}

// withFastPath sets the FastPath field.
func (b *testPhasedStateBuilder) withFastPath(fp bool) *testPhasedStateBuilder {
	b.s.FastPath = fp
	return b
}

// withTestFirst sets the TestFirst field.
func (b *testPhasedStateBuilder) withTestFirst(tf bool) *testPhasedStateBuilder {
	b.s.TestFirst = tf
	return b
}

// withSwarmFirst sets the SwarmFirst field.
func (b *testPhasedStateBuilder) withSwarmFirst(sf bool) *testPhasedStateBuilder {
	b.s.SwarmFirst = sf
	return b
}

// withVerdicts sets the Verdicts map (replacing the default empty map).
func (b *testPhasedStateBuilder) withVerdicts(v map[string]string) *testPhasedStateBuilder {
	b.s.Verdicts = v
	return b
}

// withVerdict adds a single verdict entry to the Verdicts map.
func (b *testPhasedStateBuilder) withVerdict(key, val string) *testPhasedStateBuilder {
	if b.s.Verdicts == nil {
		b.s.Verdicts = make(map[string]string)
	}
	b.s.Verdicts[key] = val
	return b
}

// withAttempts sets the Attempts map (replacing the default empty map).
func (b *testPhasedStateBuilder) withAttempts(a map[string]int) *testPhasedStateBuilder {
	b.s.Attempts = a
	return b
}

// withAttempt adds a single attempt entry to the Attempts map.
func (b *testPhasedStateBuilder) withAttempt(key string, count int) *testPhasedStateBuilder {
	if b.s.Attempts == nil {
		b.s.Attempts = make(map[string]int)
	}
	b.s.Attempts[key] = count
	return b
}

// withStartedAt sets the StartedAt timestamp string.
func (b *testPhasedStateBuilder) withStartedAt(t string) *testPhasedStateBuilder {
	b.s.StartedAt = t
	return b
}

// withOpts sets the phasedEngineOptions.
func (b *testPhasedStateBuilder) withOpts(opts phasedEngineOptions) *testPhasedStateBuilder {
	b.s.Opts = opts
	return b
}

// withFindings is a convenience builder for use in retry-context tests;
// retryContext is constructed separately, but this helper attaches opts
// that embed MaxRetries for gate-retry scenarios.
func (b *testPhasedStateBuilder) withMaxRetries(n int) *testPhasedStateBuilder {
	b.s.Opts.MaxRetries = n
	return b
}

// withTerminalStatus sets the TerminalStatus field.
func (b *testPhasedStateBuilder) withTerminalStatus(s string) *testPhasedStateBuilder {
	b.s.TerminalStatus = s
	return b
}

// withTerminalReason sets the TerminalReason field.
func (b *testPhasedStateBuilder) withTerminalReason(r string) *testPhasedStateBuilder {
	b.s.TerminalReason = r
	return b
}

// withParentEpic sets the ParentEpic field.
func (b *testPhasedStateBuilder) withParentEpic(e string) *testPhasedStateBuilder {
	b.s.ParentEpic = e
	return b
}

// withBackend sets the Backend field.
func (b *testPhasedStateBuilder) withBackend(bk string) *testPhasedStateBuilder {
	b.s.Backend = bk
	return b
}
