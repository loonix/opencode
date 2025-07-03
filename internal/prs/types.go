package prs

import "time"

// PRSLog holds the structured information for a Personal Reasoning System log entry.
type PRSLog struct {
	Task               string
	ProjectContext     string // Context detected from the project environment
	AdditionalContext  string // User-provided additional context
	Constraints        string // User-provided constraints
	Reasoning          string // LLM response for reasoning
	Evaluation         string // LLM response for evaluation
	Adaptation         string // LLM response for adaptation
	FinalOutputSummary string // LLM response for final synthesis
	Timestamp          time.Time
	FilePath           string // Full path where the log is saved
}

// TaskData represents the structure for tasks defined in YAML or JSON files.
// This might be used by the GeneratePRSLogTool if it supports loading tasks from files.
type TaskData struct {
	Task        string `json:"task" yaml:"task"`
	Context     string `json:"context,omitempty" yaml:"context,omitempty"`
	Constraints string `json:"constraints,omitempty" yaml:"constraints,omitempty"`
}
