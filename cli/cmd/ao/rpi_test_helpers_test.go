package main

// newTestPhasedState returns a phasedState with sensible test defaults.
// Use the With* builder methods to override specific fields.
func newTestPhasedState() *phasedState {
	return &phasedState{
		SchemaVersion: 1,
		Goal:          "test goal",
		Phase:         1,
		StartPhase:    1,
		Cycle:         1,
		Verdicts:      make(map[string]string),
		Attempts:      make(map[string]int),
		StartedAt:     "2026-02-19T00:00:00Z",
	}
}

// WithGoal sets the Goal field.
func (s *phasedState) WithGoal(g string) *phasedState {
	s.Goal = g
	return s
}

// WithEpicID sets the EpicID field.
func (s *phasedState) WithEpicID(id string) *phasedState {
	s.EpicID = id
	return s
}

// WithPhase sets both Phase and StartPhase.
func (s *phasedState) WithPhase(p int) *phasedState {
	s.Phase = p
	s.StartPhase = p
	return s
}

// WithStartPhase sets only StartPhase.
func (s *phasedState) WithStartPhase(p int) *phasedState {
	s.StartPhase = p
	return s
}

// WithOpts sets the phasedEngineOptions.
func (s *phasedState) WithOpts(opts phasedEngineOptions) *phasedState {
	s.Opts = opts
	return s
}

// WithRunID sets the RunID field.
func (s *phasedState) WithRunID(id string) *phasedState {
	s.RunID = id
	return s
}

// WithFastPath sets the FastPath field.
func (s *phasedState) WithFastPath(fp bool) *phasedState {
	s.FastPath = fp
	return s
}

// WithSwarmFirst sets the SwarmFirst field.
func (s *phasedState) WithSwarmFirst(sf bool) *phasedState {
	s.SwarmFirst = sf
	return s
}
