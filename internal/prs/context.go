package prs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectInfo holds basic information about a detected project.
type ProjectInfo struct {
	Name         string
	Type         string
	Path         string
	Dependencies []string
}

// DetectProjectContext attempts to detect the project type and gather context.
func DetectProjectContext(workingDir string) (string, error) {
	var detectedProjects []ProjectInfo

	// Check for package.json (Node.js)
	packageJSONPath := filepath.Join(workingDir, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		info, err := parsePackageJSON(packageJSONPath)
		if err == nil {
			info.Type = "Node.js"
			info.Path = packageJSONPath
			detectedProjects = append(detectedProjects, info)
		}
		// Log error if parsing fails, but continue
		// logging.Error("Error parsing package.json", "error", err)
	}

	// Check for pubspec.yaml (Flutter/Dart)
	pubspecYAMLPath := filepath.Join(workingDir, "pubspec.yaml")
	if _, err := os.Stat(pubspecYAMLPath); err == nil {
		info, err := parsePubspecYAML(pubspecYAMLPath)
		if err == nil {
			info.Type = "Flutter/Dart"
			info.Path = pubspecYAMLPath
			detectedProjects = append(detectedProjects, info)
		}
		// Log error
	}

	// Check for requirements.txt (Python)
	requirementsTXTPath := filepath.Join(workingDir, "requirements.txt")
	if _, err := os.Stat(requirementsTXTPath); err == nil {
		info, err := parseRequirementsTXT(requirementsTXTPath)
		if err == nil {
			info.Type = "Python"
			info.Path = requirementsTXTPath
			detectedProjects = append(detectedProjects, info)
		}
		// Log error
	}

	if len(detectedProjects) == 0 {
		return "No known project structure detected in the current directory.", nil
	}

	var contextStrings []string
	for _, p := range detectedProjects {
		projStr := fmt.Sprintf("Detected %s project (%s):\n  Name: %s", p.Type, filepath.Base(p.Path), p.Name)
		if len(p.Dependencies) > 0 {
			projStr += fmt.Sprintf("\n  Dependencies: %s", strings.Join(p.Dependencies, ", "))
		}
		contextStrings = append(contextStrings, projStr)
	}

	return strings.Join(contextStrings, "\n\n"), nil
}

func parsePackageJSON(filePath string) (ProjectInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ProjectInfo{}, err
	}

	var result struct {
		Name         string            `json:"name"`
		Dependencies map[string]string `json:"dependencies"`
		Dev          map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ProjectInfo{}, err
	}

	var deps []string
	for dep := range result.Dependencies {
		deps = append(deps, dep)
	}
	for dep := range result.DevDependencies {
		deps = append(deps, dep) // Include dev dependencies as well
	}

	return ProjectInfo{Name: result.Name, Dependencies: deps}, nil
}

func parsePubspecYAML(filePath string) (ProjectInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ProjectInfo{}, err
	}

	var result struct {
		Name         string            `yaml:"name"`
		Dependencies map[string]any `yaml:"dependencies"`
		DevDependencies map[string]any `yaml:"dev_dependencies"` // Corrected field name to match common usage + tag
	}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return ProjectInfo{}, err
	}

	var deps []string
	for dep := range result.Dependencies {
		deps = append(deps, dep)
	}
    for dep := range result.DevDependencies { // Corrected struct field access
		deps = append(deps, dep)
	}


	return ProjectInfo{Name: result.Name, Dependencies: deps}, nil
}

func parseRequirementsTXT(filePath string) (ProjectInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ProjectInfo{}, err
	}

	lines := strings.Split(string(data), "\n")
	var deps []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "==", 2) // Also handles >=, <=, etc. by taking the part before version specifier
			deps = append(deps, strings.TrimSpace(parts[0]))
		}
	}
	// For requirements.txt, project name is not usually in the file.
	// Use the name of the directory containing requirements.txt, relative to the overall workingDir.
	// Or, more simply, just use the base name of the directory where requirements.txt resides.
	// This is a proxy, as Python projects don't have a standard metadata file for project name like package.json.
	// The filePath is absolute here if workingDir was absolute.
	// We want the name of the directory that filePath is in.
	projectName := filepath.Base(filepath.Dir(filePath))
	// If requirements.txt is at the root of workingDir, then Dir(filePath) is workingDir.
	// So Base(Dir(filePath)) would be the last component of workingDir.

	if projectName == "." || projectName == "" || projectName == filepath.Base(os.Getenv("HOME")) { // Avoid using home dir name
		projectName = "Python Project" // Fallback
	}

	return ProjectInfo{Name: projectName, Dependencies: deps}, nil
}
