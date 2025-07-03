package prs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/opencode-ai/opencode/internal/config"
	// "github.com/opencode-ai/opencode/internal/logging" // For potential future logging
)

// getLogsDir resolves the directory where PRS logs are stored.
func getLogsDir(appConfig *config.Config) (string, error) {
	logsDir := appConfig.PRS.LogsPath
	if logsDir == "" {
		dataDir, err := config.GetDataDir()
		if err != nil {
			return "", fmt.Errorf("could not get opencode data directory: %w", err)
		}
		logsDir = filepath.Join(dataDir, "prs_logs")
	}
	// Ensure the path is absolute. If Prs.LogsPath was relative, it's relative to where opencode is run.
	// For consistency, it might be better if relative paths in config are resolved relative to DataDir.
	dataDir, err := config.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("could not get opencode data directory: %w", err)
	}

	if logsDir == "" { // Case 1: Path is empty in config
		logsDir = filepath.Join(dataDir, "prs_logs")
	} else if !filepath.IsAbs(logsDir) { // Case 3: Path is relative in config
		logsDir = filepath.Join(dataDir, logsDir)
	}
	// Case 2 (Absolute path in config) is implicitly handled as logsDir remains unchanged.


	// Ensure the directory exists, especially if it's the first time or path is custom.
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create PRS logs directory '%s': %w", logsDir, err)
	}
	return logsDir, nil
}

// ListPRSLogs retrieves a list of PRS log file names (base names).
// Returns a list of file names (e.g., "prs_20231027_103000.prompt.md")
func ListPRSLogs(appConfig *config.Config) ([]string, error) {
	logsDir, err := getLogsDir(appConfig)
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// logging.Info("PRS logs directory does not exist, returning empty list.", "path", logsDir)
			return []string{}, nil // No logs yet is not an error
		}
		return nil, fmt.Errorf("failed to read PRS logs directory '%s': %w", logsDir, err)
	}

	var logFiles []string
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "prs_") && strings.HasSuffix(entry.Name(), ".prompt.md") {
			logFiles = append(logFiles, entry.Name())
		}
	}

	// Sort files, most recent first (by name, which includes timestamp)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i] > logFiles[j] // Descending order
	})

	return logFiles, nil
}

// ReadPRSLogFile reads the content of a specific PRS log file by its base name.
func ReadPRSLogFile(logFileName string, appConfig *config.Config) (string, error) {
	logsDir, err := getLogsDir(appConfig)
	if err != nil {
		return "", err
	}

	// Validate logFileName to prevent path traversal, though it should just be a base name.
	if strings.Contains(logFileName, string(filepath.Separator)) || strings.Contains(logFileName, "..") {
		return "", fmt.Errorf("invalid PRS log file name format: %s", logFileName)
	}

	filePath := filepath.Join(logsDir, logFileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("PRS log file '%s' not found in '%s'", logFileName, logsDir)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read PRS log file '%s': %w", filePath, err)
	}

	return string(content), nil
}

// SearchPRSLogs searches for a keyword in all PRS log files.
// Returns a list of base file names of logs that contain the keyword.
func SearchPRSLogs(keyword string, appConfig *config.Config) ([]string, error) {
	logsDir, err := getLogsDir(appConfig)
	if err != nil {
		return nil, err
	}

	allLogFiles, err := ListPRSLogs(appConfig) // ListPRSLogs already handles dir not existing
	if err != nil {
		return nil, fmt.Errorf("failed to list PRS logs for searching: %w", err)
	}

	var matchingFiles []string
	lowerKeyword := strings.ToLower(keyword)

	for _, logFileName := range allLogFiles {
		// Construct full path for reading
		filePath := filepath.Join(logsDir, logFileName)

		contentBytes, err := os.ReadFile(filePath)
		if err != nil {
			// logging.Error("Failed to read log file during search, skipping.", "file", filePath, "error", err)
			continue // Skip files that can't be read
		}

		if strings.Contains(strings.ToLower(string(contentBytes)), lowerKeyword) {
			matchingFiles = append(matchingFiles, logFileName)
		}
	}
	// ListPRSLogs already sorts them, so matchingFiles will also be sorted if order is preserved.
	// If a different sort order is needed for search results, it can be applied here.
	return matchingFiles, nil
}
