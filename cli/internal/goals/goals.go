package goals

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// GoalType categorizes goals by domain.
type GoalType string

const (
	GoalTypeHealth       GoalType = "health"
	GoalTypeArchitecture GoalType = "architecture"
	GoalTypeQuality      GoalType = "quality"
	GoalTypeMeta         GoalType = "meta"
)

// ContinuousMetric defines a metric and threshold for continuous evaluation.
type ContinuousMetric struct {
	Metric    string  `yaml:"metric"`
	Threshold float64 `yaml:"threshold"`
}

// Goal represents a single goal entry.
type Goal struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Check       string            `yaml:"check"`
	Weight      int               `yaml:"weight"`
	Type        GoalType          `yaml:"type,omitempty"`
	Pillar      string            `yaml:"pillar,omitempty"`
	Continuous  *ContinuousMetric `yaml:"continuous,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
}

// GoalFile is the top-level structure of a goals YAML file.
type GoalFile struct {
	Version int    `yaml:"version"`
	Mission string `yaml:"mission,omitempty"`
	Goals   []Goal `yaml:"goals"`
}

// ValidationError describes a validation problem with a specific goal field.
type ValidationError struct {
	GoalID  string
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("goal %q field %q: %s", e.GoalID, e.Field, e.Message)
}

// KebabRe matches kebab-case identifiers (e.g. "my-goal-id").
var KebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidTypes enumerates the allowed GoalType values.
var ValidTypes = map[GoalType]bool{
	GoalTypeHealth:       true,
	GoalTypeArchitecture: true,
	GoalTypeQuality:      true,
	GoalTypeMeta:         true,
}

// defaultGoalTypes sets the default Type to GoalTypeHealth for any goal that has none.
func defaultGoalTypes(goals []Goal) {
	for i := range goals {
		if goals[i].Type == "" {
			goals[i].Type = GoalTypeHealth
		}
	}
}

// LoadGoals reads and parses a goals YAML file.
// Accepts version 2 or 3. Defaults Goal.Type to "health" if empty.
func LoadGoals(path string) (*GoalFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var gf GoalFile
	if err := yaml.Unmarshal(data, &gf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if gf.Version != 2 && gf.Version != 3 {
		return nil, fmt.Errorf("unsupported version %d (expected 2 or 3)", gf.Version)
	}

	defaultGoalTypes(gf.Goals)
	return &gf, nil
}

// ValidateGoals checks a GoalFile for structural correctness.
// Returns an empty slice if all goals are valid.
func ValidateGoals(gf *GoalFile) []ValidationError {
	var errs []ValidationError
	seen := make(map[string]bool)
	for _, g := range gf.Goals {
		errs = append(errs, validateGoalID(g, seen)...)
		errs = append(errs, validateGoalFields(g)...)
	}
	return errs
}

// validateGoalID checks ID presence, uniqueness, and format.
func validateGoalID(g Goal, seen map[string]bool) []ValidationError {
	var errs []ValidationError
	if g.ID == "" {
		return append(errs, ValidationError{GoalID: g.ID, Field: "id", Message: "required"})
	}
	if seen[g.ID] {
		errs = append(errs, ValidationError{GoalID: g.ID, Field: "id", Message: "duplicate"})
	}
	seen[g.ID] = true
	if !KebabRe.MatchString(g.ID) {
		errs = append(errs, ValidationError{GoalID: g.ID, Field: "id", Message: "must be kebab-case"})
	}
	return errs
}

// requireField appends a "required" validation error if value is empty.
func requireField(errs *[]ValidationError, goalID, field, value string) {
	if value == "" {
		*errs = append(*errs, ValidationError{GoalID: goalID, Field: field, Message: "required"})
	}
}

// validateGoalFields checks description, check, weight, and type fields.
func validateGoalFields(g Goal) []ValidationError {
	var errs []ValidationError
	requireField(&errs, g.ID, "description", g.Description)
	requireField(&errs, g.ID, "check", g.Check)
	if g.Weight < 1 || g.Weight > 10 {
		errs = append(errs, ValidationError{GoalID: g.ID, Field: "weight", Message: "must be 1-10"})
	}
	if g.Type != "" && !ValidTypes[g.Type] {
		errs = append(errs, ValidationError{GoalID: g.ID, Field: "type", Message: fmt.Sprintf("invalid type %q", g.Type)})
	}
	return errs
}
