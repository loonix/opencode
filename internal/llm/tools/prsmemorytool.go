package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/prs"
	// "github.com/opencode-ai/opencode/internal/logging" // For potential future logging
)

const PRSMemoryToolName = "PRSMemory"

// PRSMemoryToolParams defines parameters for interacting with PRS memory.
type PRSMemoryToolParams struct {
	Action  string `json:"action"`            // "list", "view", "search"
	Index   string `json:"index,omitempty"`   // Index for "view" (as string, to be parsed to int)
	Keyword string `json:"keyword,omitempty"` // Keyword for "search"
}

type prsMemoryTool struct {
	appConfig *config.Config
}

// NewPRSMemoryTool creates a new tool instance.
func NewPRSMemoryTool(appCfg *config.Config) BaseTool {
	return &prsMemoryTool{
		appConfig: appCfg,
	}
}

func (t *prsMemoryTool) Info() ToolInfo {
	return ToolInfo{
		Name: PRSMemoryToolName,
		Description: "Manages and interacts with saved Personal Reasoning System (PRS) logs. " +
			"Allows listing, viewing, and searching PRS logs.",
		Parameters: map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform: 'list', 'view', or 'search'.",
			},
			"index": map[string]any{
				"type":        "string", // Keep as string for flexibility, parse to int later
				"description": "The 1-based index of the log to view (required for 'view' action). Get indices from 'list' action.",
			},
			"keyword": map[string]any{
				"type":        "string",
				"description": "The keyword to search for in logs (required for 'search' action).",
			},
		},
		Required: []string{"action"},
	}
}

func (t *prsMemoryTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params PRSMemoryToolParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		// logging.Error("PRSMemoryTool: Failed to unmarshal params", "input", call.Input, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("invalid parameters: %v", err)), nil
	}

	params.Action = strings.ToLower(strings.TrimSpace(params.Action))

	switch params.Action {
	case "list":
		return t.listLogs(ctx)
	case "view":
		if params.Index == "" {
			return NewTextErrorResponse("missing 'index' parameter for 'view' action"), nil
		}
		// The python CLI used 1-based indexing. Let's stick to that for the tool's user-facing part.
		idx, err := strconv.Atoi(params.Index)
		if err != nil || idx <= 0 {
			return NewTextErrorResponse(fmt.Sprintf("invalid 'index' parameter: must be a positive integer, got '%s'", params.Index)), nil
		}
		return t.viewLog(ctx, idx-1) // Convert to 0-based for internal use
	case "search":
		if params.Keyword == "" {
			return NewTextErrorResponse("missing 'keyword' parameter for 'search' action"), nil
		}
		return t.searchLogs(ctx, params.Keyword)
	default:
		// logging.Warn("PRSMemoryTool: Unknown action", "action", params.Action)
		return NewTextErrorResponse(fmt.Sprintf("unknown action: '%s'. Valid actions are 'list', 'view', 'search'.", params.Action)), nil
	}
}

func (t *prsMemoryTool) listLogs(ctx context.Context) (ToolResponse, error) {
	// logging.Info("PRSMemoryTool: Listing logs")
	logFiles, err := prs.ListPRSLogs(t.appConfig)
	if err != nil {
		// logging.Error("PRSMemoryTool: Failed to list logs", "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to list PRS logs: %v", err)), nil
	}

	if len(logFiles) == 0 {
		return NewTextResponse("No PRS logs found."), nil
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString("Available PRS logs (most recent first):\n")
	for i, fileName := range logFiles {
		responseBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, fileName)) // 1-based index for display
	}
	return NewTextResponse(responseBuilder.String()), nil
}

func (t *prsMemoryTool) viewLog(ctx context.Context, zeroBasedIndex int) (ToolResponse, error) {
	// logging.Info("PRSMemoryTool: Viewing log", "index", zeroBasedIndex)
	logFiles, err := prs.ListPRSLogs(t.appConfig) // List to map index to filename
	if err != nil {
		// logging.Error("PRSMemoryTool: Failed to list logs for view", "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to access PRS logs: %v", err)), nil
	}

	if zeroBasedIndex < 0 || zeroBasedIndex >= len(logFiles) {
		return NewTextErrorResponse(fmt.Sprintf("invalid index. Max index is %d.", len(logFiles))), nil
	}

	fileNameToView := logFiles[zeroBasedIndex]
	content, err := prs.ReadPRSLogFile(fileNameToView, t.appConfig)
	if err != nil {
		// logging.Error("PRSMemoryTool: Failed to read log file", "filename", fileNameToView, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to read PRS log '%s': %v", fileNameToView, err)), nil
	}
	return NewTextResponse(fmt.Sprintf("Content of %s:\n\n%s", fileNameToView, content)), nil
}

func (t *prsMemoryTool) searchLogs(ctx context.Context, keyword string) (ToolResponse, error) {
	// logging.Info("PRSMemoryTool: Searching logs", "keyword", keyword)
	matchingFiles, err := prs.SearchPRSLogs(keyword, t.appConfig)
	if err != nil {
		// logging.Error("PRSMemoryTool: Failed to search logs", "keyword", keyword, "error", err)
		return NewTextErrorResponse(fmt.Sprintf("failed to search PRS logs: %v", err)), nil
	}

	if len(matchingFiles) == 0 {
		return NewTextResponse(fmt.Sprintf("No PRS logs found containing the keyword: '%s'", keyword)), nil
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("PRS logs containing '%s':\n", keyword))
	for _, fileName := range matchingFiles {
		responseBuilder.WriteString(fileName + "\n")
	}
	return NewTextResponse(responseBuilder.String()), nil
}
