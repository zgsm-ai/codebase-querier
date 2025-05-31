package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandExecutor tests the basic command execution functionality
func TestCommandExecutor(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executor := NewCommandExecutor(tempDir)

	// Test simple command execution
	output, err := executor.ExecuteCommand(context.Background(), "echo 'test'")
	if err != nil {
		t.Errorf("ExecuteCommand failed: %v", err)
	}
	if output != "test\n" {
		t.Errorf("Unexpected command output. Got: %q, Want: %q", output, "test\n")
	}

	// Test command with error
	_, err = executor.ExecuteCommand(context.Background(), "nonexistent-command")
	if err == nil {
		t.Error("Expected error for nonexistent command, got nil")
	}
}

// TestBuildCommandString tests the command string building functionality
func TestBuildCommandString(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	executor := NewCommandExecutor(tempDir)

	cmd := Command{
		Base: "test-command",
		Args: []CommandArg{
			"--source",
			"__sourcePath__",
			"--output",
			"__outputPath__",
			"--build-args",
			"__buildArgs__",
		},
	}

	// Test without build args
	expected := fmt.Sprintf("test-command --source %s --output %s --build-args ", tempDir, executor.outputPath)
	result := executor.BuildCommandString(cmd, "")
	if result != expected {
		t.Errorf("BuildCommandString without build args failed. Got: %q, Want: %q", result, expected)
	}

	// Test with build args
	buildArgs := "build arg1 arg2"
	expected = fmt.Sprintf("test-command --source %s --output %s --build-args %s", tempDir, executor.outputPath, buildArgs)
	result = executor.BuildCommandString(cmd, buildArgs)
	if result != expected {
		t.Errorf("BuildCommandString with build args failed. Got: %q, Want: %q", result, expected)
	}
}

// TestFindDetectionFile tests the file detection functionality
func TestFindDetectionFile(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{
		"go.mod",
		"package.json",
		"pom.xml",
		"subdir/build.gradle",
	}

	for _, file := range testFiles {
		dir := filepath.Dir(file)
		if dir != "." {
			if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test cases
	testCases := []struct {
		name           string
		detectionFiles []string
		expected       string
		shouldFind     bool
	}{
		{
			name:           "Find go.mod",
			detectionFiles: []string{"go.mod"},
			expected:       filepath.Join(tempDir, "go.mod"),
			shouldFind:     true,
		},
		{
			name:           "Find nested build.gradle",
			detectionFiles: []string{"**/build.gradle"},
			expected:       filepath.Join(tempDir, "subdir", "build.gradle"),
			shouldFind:     true,
		},
		{
			name:           "Find multiple files",
			detectionFiles: []string{"package.json", "pom.xml"},
			expected:       filepath.Join(tempDir, "package.json"),
			shouldFind:     true,
		},
		{
			name:           "Find nonexistent file",
			detectionFiles: []string{"nonexistent.txt"},
			expected:       "",
			shouldFind:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found, err := findDetectionFile(tempDir, tc.detectionFiles)
			if err != nil {
				t.Errorf("findDetectionFile failed: %v", err)
				return
			}

			if tc.shouldFind {
				if found != tc.expected {
					t.Errorf("Expected to find %s, got %s", tc.expected, found)
				}
			} else {
				if found != "" {
					t.Errorf("Expected no file to be found, got %s", found)
				}
			}
		})
	}
}

// TestSCIPIndexGeneration_GoProject tests the SCIP index generation for a Go project.
// It uses the current codebase as the test project.
func TestSCIPIndexGeneration_GoProject(t *testing.T) {
	// Get the directory containing the test file.
	_, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get current file information")
	}

	codebasePath := "/home/zk/projects/codebase-indexer"

	fmt.Printf("Testing with codebase path: %s\n", codebasePath)

	generator := &SCIPIndexGenerator{}

	// Run the index generation process
	indexPath, err := generator.Generate(context.Background(), codebasePath)

	// Check the results
	if err != nil {
		t.Errorf("SCIP index generation failed: %v", err)
		return
	}

	// Verify the output path
	expectedOutputPath := filepath.Join(codebasePath, ".codebase_index", "index.scip")
	if indexPath != expectedOutputPath {
		t.Errorf("Unexpected index path. Got: %s, Want: %s", indexPath, expectedOutputPath)
	}

	// Optional: Check if the file actually exists (requires the command to succeed and create the file)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("Generated index file does not exist at: %s", indexPath)
	}

	fmt.Printf("Successfully generated SCIP index at: %s\n", indexPath)
}

func TestSCIPIndexGenerator_Generate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test repository structure
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create a test Go module
	err = os.WriteFile(filepath.Join(testRepo, "go.mod"), []byte(`
		module github.com/test/project

		go 1.21
	`), 0644)
	require.NoError(t, err)

	// Create a test Go file with some symbols
	goFile := `
		package main

		import (
			"fmt"
			"time"
		)

		// User represents a user in the system
		type User struct {
			ID        string    ` + "`json:\"id\"`" + `
			Name      string    ` + "`json:\"name\"`" + `
			CreatedAt time.Time ` + "`json:\"created_at\"`" + `
		}

		// UserService handles user-related operations
		type UserService struct {
			users map[string]*User
		}

		// NewUserService creates a new UserService instance
		func NewUserService() *UserService {
			return &UserService{
				users: make(map[string]*User),
			}
		}

		// AddUser adds a new user to the service
		func (s *UserService) AddUser(user *User) error {
			if user.ID == "" {
				return fmt.Errorf("user ID cannot be empty")
			}
			s.users[user.ID] = user
			return nil
		}

		// GetUser retrieves a user by ID
		func (s *UserService) GetUser(id string) (*User, error) {
			user, exists := s.users[id]
			if !exists {
				return nil, fmt.Errorf("user not found: %s", id)
			}
			return user, nil
		}

		func main() {
			service := NewUserService()
			user := &User{
				ID:        "1",
				Name:      "Test User",
				CreatedAt: time.Now(),
			}
			if err := service.AddUser(user); err != nil {
				fmt.Printf("Error adding user: %v\n", err)
			}
		}
	`
	err = os.WriteFile(filepath.Join(testRepo, "main.go"), []byte(goFile), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		repoPath    string
		setup       func() error
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful index generation",
			repoPath: testRepo,
			setup: func() error {
				return nil
			},
			wantErr: false,
		},
		{
			name:     "non-existent repository",
			repoPath: filepath.Join(tempDir, "non-existent"),
			setup: func() error {
				return nil
			},
			wantErr:     true,
			errContains: "codebase path does not exist",
		},
		{
			name:     "unsupported language",
			repoPath: testRepo,
			setup: func() error {
				// Remove go.mod to make it an unsupported language
				return os.Remove(filepath.Join(testRepo, "go.mod"))
			},
			wantErr:     true,
			errContains: "no matching language configuration found",
		},
		{
			name:     "invalid Go module",
			repoPath: testRepo,
			setup: func() error {
				// Create an invalid go.mod
				invalidMod := `invalid go.mod content`
				return os.WriteFile(filepath.Join(testRepo, "go.mod"), []byte(invalidMod), 0644)
			},
			wantErr:     true,
			errContains: "command execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			if tt.setup != nil {
				err := tt.setup()
				require.NoError(t, err)
			}

			// Create generator
			generator, err := NewSCIPIndexGenerator()
			require.NoError(t, err)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Generate index
			indexPath, err := generator.Generate(ctx, tt.repoPath)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, indexPath)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, indexPath)
				assert.FileExists(t, indexPath)

				// Verify the index file contains expected content
				// This is a basic check - more detailed verification is done in scip_parser_test.go
				fileInfo, err := os.Stat(indexPath)
				assert.NoError(t, err)
				assert.Greater(t, fileInfo.Size(), int64(0), "Index file should not be empty")
			}
		})
	}
}

func TestSCIPIndexGenerator_ConcurrentGeneration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create two test repositories
	repo1 := filepath.Join(tempDir, "repo1")
	repo2 := filepath.Join(tempDir, "repo2")
	for _, repo := range []string{repo1, repo2} {
		err = os.MkdirAll(repo, 0755)
		require.NoError(t, err)

		// Create go.mod for Go detection
		goMod := `
			module github.com/test/project

			go 1.21
		`
		err = os.WriteFile(filepath.Join(repo, "go.mod"), []byte(goMod), 0644)
		require.NoError(t, err)

		// Create a simple Go file
		goFile := `
			package main

			import "fmt"

			func main() {
				fmt.Println("Hello, World!")
			}
		`
		err = os.WriteFile(filepath.Join(repo, "main.go"), []byte(goFile), 0644)
		require.NoError(t, err)
	}

	// Create generator
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Try to generate index for the same repository concurrently
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := generator.Generate(ctx, repo1)
			done <- err
		}()
	}

	// Check that one of the generations fails due to concurrent processing
	errors := []error{<-done, <-done}
	assert.True(t, (errors[0] != nil) != (errors[1] != nil), "Expected exactly one generation to fail")

	// Generate index for different repositories concurrently
	done = make(chan error, 2)
	for i, repo := range []string{repo1, repo2} {
		go func(r string) {
			_, err := generator.Generate(ctx, r)
			done <- err
		}(repo)
	}

	// Both generations should succeed
	for i := 0; i < 2; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestSCIPIndexGenerator_Timeout(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test repository
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create go.mod for Go detection
	goMod := `
		module github.com/test/project

		go 1.21
	`
	err = os.WriteFile(filepath.Join(testRepo, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)

	// Create a simple Go file
	goFile := `
		package main

		import "fmt"

		func main() {
			fmt.Println("Hello, World!")
		}
	`
	err = os.WriteFile(filepath.Join(testRepo, "main.go"), []byte(goFile), 0644)
	require.NoError(t, err)

	// Create generator
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Generate index
	_, err = generator.Generate(ctx, testRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
