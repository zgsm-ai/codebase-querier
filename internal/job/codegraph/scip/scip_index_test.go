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

// TestSCIPIndexGeneration_GoProject tests the SCIP index generation for a Go project.
// It uses the current codebase as the test project.
func TestSCIPIndexGeneration_GoProject(t *testing.T) {
	// Get the directory containing the test file.
	_, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get current file information")
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	codebasePath := filepath.Join(homeDir, "projects", "codebase-indexer")
	fmt.Printf("Testing with codebase path: %s\n", codebasePath)

	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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
			errContains: "codebase does not exist",
		},
		{
			name:     "unsupported language",
			repoPath: testRepo,
			setup: func() error {
				// Remove go.mod to make it an unsupported language
				return os.Remove(filepath.Join(testRepo, "go.mod"))
			},
			wantErr:     true,
			errContains: "exit status 1",
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
			errContains: "exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			if tt.setup != nil {
				err := tt.setup()
				require.NoError(t, err)
			}

			// Generate index
			indexPath, err := generator.Generate(context.Background(), tt.repoPath)

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

				// Verify the output path is correct
				expectedPath := filepath.Join(tt.repoPath, ".codebase_index", "index.scip")
				assert.Equal(t, expectedPath, indexPath)
				assert.FileExists(t, indexPath)

				// Verify the index file contains expected content
				fileInfo, err := os.Stat(indexPath)
				if err != nil {
					t.Logf("Failed to stat index file: %v", err)
					return
				}
				assert.Greater(t, fileInfo.Size(), int64(0), "Index file should not be empty")
			}
		})
	}
}

func TestSCIPIndexGenerator_ConcurrentGeneration(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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
	for _, repo := range []string{repo1, repo2} {
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
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Generate index
	_, err = generator.Generate(ctx, testRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestLanguageDetection(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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

	// Generate index
	indexPath, err := generator.Generate(context.Background(), testRepo)
	if err != nil {
		t.Errorf("failed to generate index: %v", err)
		return
	}

	// Verify the index file exists
	assert.FileExists(t, indexPath)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func TestCommandGeneration(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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

	// Generate index
	indexPath, err := generator.Generate(context.Background(), testRepo)
	if err != nil {
		t.Errorf("failed to generate index: %v", err)
		return
	}

	// Verify the index file exists
	assert.FileExists(t, indexPath)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func TestSCIPIndexGenerator(t *testing.T) {
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

	// Create config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

	// Generate index
	indexPath, err := generator.Generate(context.Background(), testRepo)
	if err != nil {
		// If scip-go is not installed, skip the test
		if err.Error() == "scip-go command not found" {
			t.Skip("scip-go command not found, skipping test")
		}
		t.Errorf("failed to generate index: %v", err)
		return
	}

	// Verify the index file exists
	assert.FileExists(t, indexPath)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func TestSCIPIndexGenerator_Concurrent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test repositories
	repos := []string{
		filepath.Join(tempDir, "repo1"),
		filepath.Join(tempDir, "repo2"),
		filepath.Join(tempDir, "repo3"),
	}

	for _, repo := range repos {
		err = os.MkdirAll(repo, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(repo, "go.mod"), []byte(`
			module github.com/test/project

			go 1.21
		`), 0644)
		require.NoError(t, err)
	}

	// Create config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

	// Generate indices concurrently
	err = generator.GenerateConcurrent(context.Background(), repos)
	if err != nil {
		// If scip-go is not installed, skip the test
		if err.Error() == "scip-go command not found" {
			t.Skip("scip-go command not found, skipping test")
		}
		t.Errorf("failed to generate indices: %v", err)
		return
	}

	// Verify all index files exist
	for _, repo := range repos {
		indexPath := filepath.Join(repo, ".codebase_index", "index.scip")
		assert.FileExists(t, indexPath)

		fileInfo, err := os.Stat(indexPath)
		require.NoError(t, err)
		assert.Greater(t, fileInfo.Size(), int64(0))
	}
}

func TestSCIPIndexGenerator_GenerateConcurrent(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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

	// Generate index
	indexPath, err := generator.Generate(context.Background(), testRepo)
	if err != nil {
		t.Errorf("failed to generate index: %v", err)
		return
	}

	// Verify the index file exists
	assert.FileExists(t, indexPath)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}

func TestSCIPIndexGenerator_ValidateCodebase(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{
			{
				Name:           "go",
				DetectionFiles: []string{"go.mod"},
				Tools: []*ToolConfig{
					{
						Name: "scip-go",
						Commands: []*Command{
							{
								Base: "scip-go",
								Args: []string{
									"--project-root",
									"__sourcePath__",
									"--output",
									"__outputPath__/index.scip",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create generator
	generator := NewSCIPIndexGenerator(config)

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

	// Generate index
	indexPath, err := generator.Generate(context.Background(), testRepo)
	if err != nil {
		t.Errorf("failed to generate index: %v", err)
		return
	}

	// Verify the index file exists
	assert.FileExists(t, indexPath)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}
