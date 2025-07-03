package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/provider"
	"github.com/opencode-ai/opencode/internal/prs"
	// "github.com/opencode-ai/opencode/internal/logging" // For potential future logging
)

const GeneratePRSLogToolName = "GeneratePRSLog"

// GeneratePRSLogToolParams defines the expected parameters for the tool.
type GeneratePRSLogToolParams struct {
	Task              string `json:"task"` // The core task description
	AdditionalContext string `json:"additional_context,omitempty"` // Optional additional context from the user
	Constraints       string `json:"constraints,omitempty"`      // Optional constraints from the user
	// FilePathForTask string `json:"file_path_for_task,omitempty"` // Future: To load task from a file
}

type generatePRSLogTool struct {
	appConfig *config.Config
	llmProvider provider.Provider // This provider will be used by the PRS generation logic
}

// NewGeneratePRSLogTool creates a new tool instance.
// It requires the application config and an LLM provider instance
// that the PRS generation logic will use for its internal LLM calls.
func NewGeneratePRSLogTool(appCfg *config.Config, llmProvider provider.Provider) BaseTool {
	return &generatePRSLogTool{
		appConfig: appCfg,
		llmProvider: llmProvider,
	}
}

func (t *generatePRSLogTool) Info() ToolInfo {
	return ToolInfo{
		Name: GeneratePRSLogToolName,
		Description: "Generates a Personal Reasoning System (PRS) log for a given task. " +
			"This involves a multi-step process of reasoning, evaluation, adaptation, and synthesis using an LLM. " +
			"The final log is saved to a file.",
		Parameters: map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "The detailed description of the task to be processed.",
			},
			"additional_context": map[string]any{
				"type":        "string",
				"description": "Optional: Any additional context relevant to the task.",
			},
			"constraints": map[string]any{
				"type":        "string",
				"description": "Optional: Any constraints that must be followed for the task.",
			},
		},
		Required: []string{"task"},
	}
}

func (t *generatePRSLogTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GeneratePRSLogToolParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		// logging.Error("GeneratePRSLogTool: Failed to unmarshal params", "input", call.Input, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	if params.Task == "" {
		return NewTextErrorResponse("missing required parameter 'task'"), nil
	}

	// logging.Info("GeneratePRSLogTool: Starting PRS generation", "task", params.Task)

	// Detect project context from the current working directory.
	// The CWD for tools should be opencode's main CWD.
	workingDir := config.WorkingDirectory() // Get current working directory from opencode's config
	detectedProjCtx, err := prs.DetectProjectContext(workingDir)
	if err != nil {
		// Non-fatal, proceed without project context or log a warning
		// logging.Warn("GeneratePRSLogTool: Failed to detect project context", "error", err)
		detectedProjCtx = "Could not detect project context: " + err.Error()
	}

	logEntry, err := prs.GeneratePRS(
		ctx,
		params.Task,
		params.AdditionalContext,
		params.Constraints,
		t.llmProvider, // Use the provider passed during tool creation
		detectedProjCtx,
	)
	if err != nil {
		// logging.Error("GeneratePRSLogTool: Failed to generate PRS log", "task", params.Task, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to generate PRS log: %v", err)), nil
	}

	err = prs.SavePRSLog(logEntry, t.appConfig)
	if err != nil {
		// logging.Error("GeneratePRSLogTool: Failed to save PRS log", "task", params.Task, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to save PRS log: %v", err)), nil
	}

	// logging.Info("GeneratePRSLogTool: PRS log generated and saved", "path", logEntry.FilePath)
	return NewTextResponse(fmt.Sprintf("PRS log generated and saved to %s.\nSummary: %s", logEntry.FilePath, logEntry.FinalOutputSummary)), nil
}
