package prs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/message"
	"github.com/opencode-ai/opencode/internal/llm/provider"
	// "github.com/opencode-ai/opencode/internal/logging" // For potential future logging
)

const prsLogTemplate = `\
# PRS Log - {{.Timestamp.Format "2006-01-02 15:04:05"}}

## Task
{{.Task}}
{{if .AdditionalContext}}
## Additional Context Provided by User
{{.AdditionalContext}}
{{end}}
{{if .Constraints}}
## Constraints Provided by User
{{.Constraints}}
{{end}}
{{if .ProjectContext}}
## Detected Project Context
{{.ProjectContext}}
{{end}}
## Reasoning
{{.Reasoning}}

## Evaluation
{{.Evaluation}}

## Adaptation
{{.Adaptation}}

## Final Output Summary
{{.FinalOutputSummary}}

# [GOD MODE: ON]
`

var (
	logTmpl *template.Template
)

func init() {
	var err error
	logTmpl, err = template.New("prsLog").Parse(prsLogTemplate)
	if err != nil {
		// This is a panic because it's a programmer error if the template is invalid.
		panic(fmt.Errorf("failed to parse PRS log template: %w", err))
	}
}

// GeneratePRS orchestrates the multi-step LLM interaction to produce a PRSLog.
func GeneratePRS(
	ctx context.Context,
	taskDesc string,
	userAdditionalContext string,
	userConstraints string,
	prsProvider provider.Provider,
	detectedProjectContext string,
) (*PRSLog, error) {
	logEntry := &PRSLog{
		Task:           taskDesc,
		ProjectContext: detectedProjectContext,
		AdditionalContext: userAdditionalContext,
		Constraints: userConstraints,
		Timestamp:      time.Now(),
	}

	// 1. Initial Prompt Construction
	var initialPromptBuilder strings.Builder
	fmt.Fprintf(&initialPromptBuilder, "Task: %s\n\n", taskDesc)

	if detectedProjectContext != "" && detectedProjectContext != "No known project structure detected in the current directory." {
		fmt.Fprintf(&initialPromptBuilder, "Given the following project context:\n%s\n\n", detectedProjectContext)
	}
	if userAdditionalContext != "" {
		fmt.Fprintf(&initialPromptBuilder, "Additional context provided:\n%s\n\n", userAdditionalContext)
	}
	if userConstraints != "" {
		fmt.Fprintf(&initialPromptBuilder, "Constraints to follow:\n%s\n\n", userConstraints)
	}
	initialPromptBuilder.WriteString("Please provide your reasoning on how to approach this task.")


	// 2. Reasoning Phase
	// logging.Debug("PRS: Sending reasoning prompt", "prompt", initialPromptBuilder.String())
	reasoningResponse, err := prsProvider.SendMessages(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: initialPromptBuilder.String()}}},
	}, nil) // No tools needed for these internal PRS steps
	if err != nil {
		return nil, fmt.Errorf("PRS reasoning phase failed: %w", err)
	}
	logEntry.Reasoning = reasoningResponse.Content
	// logging.Debug("PRS: Received reasoning response")

	// 3. Evaluation Phase
	evaluationPrompt := fmt.Sprintf("Evaluate the following reasoning for the task '%s':\n\nReasoning:\n%s\n\nProvide your evaluation.", taskDesc, logEntry.Reasoning)
	// logging.Debug("PRS: Sending evaluation prompt")
	evaluationResponse, err := prsProvider.SendMessages(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: evaluationPrompt}}},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("PRS evaluation phase failed: %w", err)
	}
	logEntry.Evaluation = evaluationResponse.Content
	// logging.Debug("PRS: Received evaluation response")

	// 4. Adaptation Phase
	adaptationPrompt := fmt.Sprintf("Based on the following evaluation, refactor or adapt the approach for the task '%s'. If the evaluation was positive, confirm the approach or suggest minor enhancements. If negative, propose a revised approach.\n\nEvaluation:\n%s\n\nProvide your adapted approach.", taskDesc, logEntry.Evaluation)
	// logging.Debug("PRS: Sending adaptation prompt")
	adaptationResponse, err := prsProvider.SendMessages(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: adaptationPrompt}}},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("PRS adaptation phase failed: %w", err)
	}
	logEntry.Adaptation = adaptationResponse.Content
	// logging.Debug("PRS: Received adaptation response")

	// 5. Final Synthesis Phase
	synthesisPrompt := fmt.Sprintf("Original Task: %s\n\nImproved/Confirmed Strategy after adaptation:\n%s\n\nProvide a final summary of the plan or the direct answer if the task was a question.", taskDesc, logEntry.Adaptation)
	// logging.Debug("PRS: Sending synthesis prompt")
	synthesisResponse, err := prsProvider.SendMessages(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: synthesisPrompt}}},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("PRS final synthesis phase failed: %w", err)
	}
	logEntry.FinalOutputSummary = synthesisResponse.Content
	// logging.Debug("PRS: Received synthesis response")

	return logEntry, nil
}

// SavePRSLog saves the PRSLog entry to a markdown file.
func SavePRSLog(logEntry *PRSLog, appConfig *config.Config) error {
	logsDir := appConfig.PRS.LogsPath // Assuming PRS.LogsPath will be added to config.Config
	if logsDir == "" {
		// Default path if not configured
		dataDir, err := config.GetDataDir()
		if err != nil {
			return fmt.Errorf("could not get opencode data directory: %w", err)
		}
		logsDir = filepath.Join(dataDir, "prs_logs")
	}

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create PRS logs directory '%s': %w", logsDir, err)
	}

	filename := fmt.Sprintf("prs_%s.prompt.md", logEntry.Timestamp.Format("20060102_150405"))
	logEntry.FilePath = filepath.Join(logsDir, filename)

	file, err := os.Create(logEntry.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create PRS log file '%s': %w", logEntry.FilePath, err)
	}
	defer file.Close()

	if err := logTmpl.Execute(file, logEntry); err != nil {
		return fmt.Errorf("failed to execute PRS log template: %w", err)
	}

	// logging.Info("PRS Log saved", "path", logEntry.FilePath)
	return nil
}
