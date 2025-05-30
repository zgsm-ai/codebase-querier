package scip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	lua "github.com/yuin/gopher-lua"
	yaml "gopkg.in/yaml.v2"
)

type IndexGenerator interface {
	Generate(ctx context.Context, codebasePath string) (path string, err error)
}

type SCIPIndexGenerator struct {
	// Add any necessary fields here
}

// luaRunCommand creates a Go function for Lua to run a terminal command.
// It now closes over the codebasePath to set the working directory directly.
// It expects only the command string as an argument from Lua.
func luaRunCommand(l *lua.LState, codebasePath string) lua.LValue { // Function returns a Lua function value
	return l.NewFunction(func(l *lua.LState) int {
		command := l.CheckString(1) // Get the command string from Lua argument

		fmt.Printf("DEBUG: Executing command: %s\n", command)
		// Use the codebasePath from the outer scope (closure)
		fmt.Printf("DEBUG: Explicitly setting working directory to (from closure): %s\n", codebasePath)

		cmd := exec.Command("bash", "-c", command)
		cmd.Dir = codebasePath // Set the working directory using the closed-over variable

		// Get the current working directory for debugging (this is the Go process's WD, not the command's)
		wd, err := os.Getwd()
		if err == nil {
			fmt.Printf("DEBUG: Go process's current working directory is: %s\n", wd)
		} else {
			fmt.Printf("DEBUG: Failed to get Go process's current working directory: %v\n", err)
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()

		// Print command output and error
		fmt.Printf("DEBUG: Command finished.\n")
		fmt.Printf("DEBUG: Stdout:\n%s\n", stdout.String())
		fmt.Printf("DEBUG: Stderr:\n%s\n", stderr.String())

		// Return output and error message to Lua
		if err != nil {
			fmt.Printf("DEBUG: Command Error: %v\n", err)
			l.Push(lua.LString(stdout.String() + stderr.String())) // Combined output on error
			l.Push(lua.LString(err.Error()))
		} else {
			fmt.Println("DEBUG: Command Success.")
			l.Push(lua.LString(stdout.String())) // stdout on success
			l.Push(lua.LNil)                     // No error message on success
		}

		return 2 // Number of return values: output, error_string
	})
}

// convertToGoValueToLua converts a Go value (derived from YAML parsing) to a Lua value.
// This is a recursive function to handle nested structures (maps, slices).
func convertToGoValueToLua(L *lua.LState, goValue interface{}) lua.LValue {
	switch v := goValue.(type) {
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []interface{}:
		// Convert Go slice to Lua table (list)
		tbl := L.NewTable()
		for _, item := range v {
			tbl.Append(convertToGoValueToLua(L, item))
		}
		return tbl
	case map[interface{}]interface{}:
		// Convert Go map to Lua table (map)
		tbl := L.NewTable()
		for key, value := range v {
			// Ensure key is a string or number for Lua table
			keyLua := convertToGoValueToLua(L, key)
			valueLua := convertToGoValueToLua(L, value)
			tbl.RawSet(keyLua, valueLua)
		}
		return tbl
	case map[string]interface{}:
		// Convert Go map to Lua table (map)
		tbl := L.NewTable()
		for key, value := range v {
			// Ensure key is a string or number for Lua table
			keyLua := lua.LString(key) // Map keys are strings from YAML
			valueLua := convertToGoValueToLua(L, value)
			tbl.RawSet(keyLua, valueLua)
		}
		return tbl
	case nil:
		return lua.LNil
	default:
		// Handle other types or return an error/nil for unsupported types
		fmt.Printf("Warning: Unsupported type in YAML conversion: %T\n", goValue)
		return lua.LNil
	}
}

// fileExists checks if a file or directory exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// findDetectionFile finds the first existing file from a list of potential detection files.
// Supports glob patterns and recursive search within a limited depth or specific subdirectories.
// For now, implementing basic recursive search.
func findDetectionFile(basePath string, detectionFiles []string) (string, error) {
	if len(detectionFiles) == 0 {
		return "", nil
	}

	var foundPath string
	var walkErr error

	// Walk through the base path directory and its subdirectories
	walkErr = filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Log the error but continue walking
			fmt.Printf("Warning: Error walking directory %s: %v\n", path, err)
			return filepath.SkipDir // Skip this directory but continue walking
		}

		if !d.IsDir() { // Only check files
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				// Should not happen if path is within basePath, but handle defensively
				fmt.Printf("Warning: Could not get relative path for %s: %v\n", path, err)
				return nil // Continue walking
			}

			// Check if the relative path matches any of the detection file patterns
			for _, pattern := range detectionFiles {
				// We need to match the relative path against the pattern.
				// filepath.Match handles glob patterns.
				matched, err := filepath.Match(pattern, relPath)
				if err != nil {
					// Invalid pattern, log and skip
					fmt.Printf("Warning: Invalid glob pattern %s: %v\n", pattern, err)
					continue
				}
				if matched {
					foundPath = path        // Found a match, store the absolute path
					return filepath.SkipAll // Stop walking as we found a file
				}
			}
		}

		return nil // Continue walking
	})

	if walkErr != nil && !errors.Is(walkErr, filepath.SkipAll) {
		return "", fmt.Errorf("error during directory walk: %w", walkErr)
	}

	return foundPath, nil // Returns the first found path or empty string
}

// buildCommandString assembles a command string from Command struct and parameters.
func buildCommandString(cmd Command, basePath, outputPath, buildArgs string) string {
	cmdParts := []string{cmd.Base}
	for i, arg := range cmd.Args {
		argStr := string(arg)
		// Substitute placeholders
		argStr = strings.ReplaceAll(argStr, "__sourcePath__", basePath)
		argStr = strings.ReplaceAll(argStr, "__outputPath__", outputPath)
		if buildArgs != "" {
			argStr = strings.ReplaceAll(argStr, "__buildArgs__", buildArgs)
		}

		// Special handling for bash -c commands where the command string is an argument
		if cmd.Base == "bash" && len(cmd.Args) > 1 && cmd.Args[0] == "-c" && i == 1 { // Check if this is the argument immediately after -c
			// The command string inside bash -c might contain placeholders that need substitution
			// We already substituted them above, but this check is for clarity if needed later.
			// For now, the general substitution handles it.
		}

		cmdParts = append(cmdParts, argStr)
	}
	// Join all parts with space to form the final command string
	// Note: This might need careful handling for arguments with spaces or special characters, especially if not using bash -c.
	return strings.Join(cmdParts, " ")
}

func (g *SCIPIndexGenerator) Generate(ctx context.Context, codebasePath string) (path string, err error) {
	// Create a new Lua state
	l := lua.NewState()
	defer l.Close()

	// Register the Go function to run commands in Lua, closing over codebasePath
	l.SetGlobal("run_command", luaRunCommand(l, codebasePath))

	// --- Read and Parse the command configuration file in Go ---
	// Read the configuration file relative to the current file's directory.
	// This assumes the 'scripts' directory is a sibling to the directory containing this Go file.
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file information")
	}
	currentDir := filepath.Dir(currentFile)
	configFilePath := filepath.Join(currentDir, "scripts", "scip_commands.yaml")

	configContent, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read command configuration file %s: %w", configFilePath, err)
	}

	var scipConfig SCIPConfig
	err = yaml.Unmarshal(configContent, &scipConfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse command configuration file %s: %w", configFilePath, err)
	}
	// --- End: Read and Parse config ---

	// --- Language and Build Tool Detection Logic in Go ---
	detectedLanguageConfig := LanguageConfig{}   // Store the detected language config
	detectedBuildToolConfig := BuildToolConfig{} // Store the detected build tool config
	languageDetected := false

	for _, langConfig := range scipConfig.Languages {
		// 1. Check for build tool specific files first and find the highest priority match
		bestMatchBuildTool := BuildToolConfig{Name: "", Priority: 9999} // Initialize with a low priority and empty name
		buildToolFound := false

		if len(langConfig.BuildTools) > 0 {
			for _, buildToolConfig := range langConfig.BuildTools {
				foundFile, _ := findDetectionFile(codebasePath, buildToolConfig.DetectionFiles)
				if foundFile != "" {
					buildToolFound = true
					// Check if this build tool has higher priority or is the first found
					if buildToolConfig.Priority < bestMatchBuildTool.Priority {
						bestMatchBuildTool = buildToolConfig
					}
				}
			}
		}

		if buildToolFound {
			detectedLanguageConfig = langConfig
			detectedBuildToolConfig = bestMatchBuildTool
			languageDetected = true
			break // Language and specific build tool detected, stop checking other languages
		}

		// 2. If no build tool files found for this language, check for general language detection files
		if !languageDetected {
			foundFile, _ := findDetectionFile(codebasePath, langConfig.DetectionFiles)
			if foundFile != "" {
				detectedLanguageConfig = langConfig
				languageDetected = true
				// No specific build tool detected, proceed with language default if available
				break // Language detected, stop checking other languages
			}
		}

		// TODO: Add check for presence of source files as a fallback if needed.
		// This would require walking parts of the directory and checking extensions.
		// For now, relying on explicit detection files.
	}

	if !languageDetected {
		return "", fmt.Errorf("could not detect supported language for codebase: %s based on known detection files", codebasePath)
	}
	// --- End: Detection Logic ---

	// --- Determine and Build Command Strings in Go ---
	var commandsToRun []string
	outputPath := filepath.Join(codebasePath, ".codebase_index") // Determine output path

	// Ensure output directory exists before building/indexing
	mkdirCmd := "mkdir -p " + outputPath
	commandsToRun = append(commandsToRun, mkdirCmd)

	// Handle build commands first if a build tool was detected with highest priority
	buildCommandsArgs := ""                 // To store substituted build command args for languages like Java
	if detectedBuildToolConfig.Name != "" { // Check if a build tool was explicitly detected with highest priority
		if len(detectedBuildToolConfig.BuildCommands) > 0 {
			for _, buildCmdData := range detectedBuildToolConfig.BuildCommands {
				// Build the build command string, substituting source and output paths
				buildCmdString := buildCommandString(buildCmdData, codebasePath, outputPath, "") // No buildArgs for build commands themselves
				commandsToRun = append(commandsToRun, buildCmdString)

				// For languages that need build args appended to the scip command (e.g., Java),
				// capture the substituted arguments part of the build command.
				var argParts []string
				for _, arg := range buildCmdData.Args {
					argStr := string(arg)
					argStr = strings.ReplaceAll(argStr, "__sourcePath__", codebasePath)
					argStr = strings.ReplaceAll(argStr, "__outputPath__", outputPath)
					argParts = append(argParts, argStr)
				}
				// Special handling for bash -c commands - we want the command string *inside* -c for buildCommandsArgs
				if buildCmdData.Base == "bash" && len(argParts) > 1 && argParts[0] == "-c" {
					buildCommandsArgs = argParts[1] // The string after -c
				} else {
					buildCommandsArgs = strings.Join(argParts, " ") // Simple join for other commands
				}
			}
		}
		// TODO: Handle build command sequences if implemented in YAML
	} else {
		// If no specific build tool was detected (even if language config has build tools defined),
		// it means none of the configured detection files for build tools were found for the highest priority one.
		// We proceed without specific build commands.
		if len(detectedLanguageConfig.BuildTools) > 0 { // Check if the language *could* have had build tools
			fmt.Printf("Warning: Language %s has configured build tools, but none of the highest priority detection files were found. Proceeding without specific build commands.\n", detectedLanguageConfig.Name)
		}
	}

	// Add the main scip commands
	if len(detectedLanguageConfig.Tools) > 0 && len(detectedLanguageConfig.Tools[0].Commands) > 0 { // Assuming first tool config is the main one
		for _, scipCmdData := range detectedLanguageConfig.Tools[0].Commands {
			// Build the scip command string, substituting all placeholders including __buildArgs__
			scipCmdString := buildCommandString(scipCmdData, codebasePath, outputPath, buildCommandsArgs)
			commandsToRun = append(commandsToRun, scipCmdString)
		}
	} else {
		return "", fmt.Errorf("no SCIP tool commands defined for language %s", detectedLanguageConfig.Name)
	}

	// --- End: Determine and Build Command Strings ---

	// --- Pass Commands List and Final Output Path to Lua and Execute ---
	luaCommandsTable := l.NewTable()
	for _, cmdStr := range commandsToRun {
		luaCommandsTable.Append(lua.LString(cmdStr))
	}
	l.SetGlobal("commandsToExecute", luaCommandsTable)
	l.SetGlobal("finalResultPath", lua.LString(filepath.Join(outputPath, "index.scip")))

	// Execute the main Lua indexing script
	// The Lua script is expected to iterate and execute commandsToExecute
	luaScriptPath := "scripts/index_codebase.lua"
	if err := l.DoFile(luaScriptPath); err != nil {
		return "", fmt.Errorf("failed to execute lua script %s: %w", luaScriptPath, err)
	}

	// Get the result from Lua global variables
	resultPathLua := l.GetGlobal("resultPath")
	errorMsgLua := l.GetGlobal("errorMsg")

	if errorMsgLua.Type() != lua.LTNil {
		// Lua script returned an error during command execution. The errorMsg from Lua is already detailed.
		return "", fmt.Errorf("SCIP index generation command execution failed: %s", errorMsgLua.String())
	}

	if resultPathLua.Type() != lua.LTString || resultPathLua.String() == "" {
		// Lua script should have set finalResultPath as resultPath on success, but didn't.
		return "", fmt.Errorf("lua script finished without error but did not return a valid resultPath")
	}

	return resultPathLua.String(), nil
}
