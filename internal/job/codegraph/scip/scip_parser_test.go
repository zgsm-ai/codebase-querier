package scip

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// TestParseSCIPFileForGraph tests the SCIP index parsing function.
func TestParseSCIPFileForGraph(t *testing.T) {
	// Create test config
	config := &Config{
		Languages: []*LanguageConfig{{
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

	// Create test repository structure
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create a test TypeScript project
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"dependencies": {
			"typescript": "^4.0.0"
		}
	}`
	err = os.WriteFile(filepath.Join(testRepo, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create a test TypeScript file with some symbols
	tsFile := `
		export interface User {
			id: string;
			name: string;
		}

		export class UserService {
			private users: User[] = [];

			constructor() {
				this.users = [];
			}

			addUser(user: User): void {
				this.users.push(user);
			}

			getUser(id: string): User | undefined {
				return this.users.find(u => u.id === id);
			}
		}
	`
	err = os.WriteFile(filepath.Join(testRepo, "user.ts"), []byte(tsFile), 0644)
	require.NoError(t, err)

	// Generate SCIP index
	ctx := context.Background()
	indexPath, err := generator.Generate(ctx, testRepo)
	if err != nil {
		// Skip test if scip-typescript is not installed
		if strings.Contains(err.Error(), "scip-typescript command not found") {
			t.Skip("scip-typescript command not found, skipping test")
		}
		require.NoError(t, err)
	}
	require.FileExists(t, indexPath)

	tests := []struct {
		name        string
		scipPath    string
		setup       func() error
		wantErr     bool
		errContains string
		validate    func(t *testing.T, nodes map[string]*SymbolNode)
	}{
		{
			name:     "successful parse",
			scipPath: indexPath,
			setup: func() error {
				return nil
			},
			wantErr: false,
			validate: func(t *testing.T, nodes map[string]*SymbolNode) {
				assert.NotEmpty(t, nodes, "Should have parsed some symbol nodes")

				// Verify User interface
				userInterface, exists := nodes["User"]
				assert.True(t, exists, "Should find User interface")
				if exists {
					assert.Equal(t, "User", userInterface.SymbolName)
					assert.NotNil(t, userInterface.SymbolInfo)
				}

				// Verify UserService class
				userService, exists := nodes["UserService"]
				assert.True(t, exists, "Should find UserService class")
				if exists {
					assert.Equal(t, "UserService", userService.SymbolName)
					assert.NotNil(t, userService.SymbolInfo)
				}
			},
		},
		{
			name:     "non-existent file",
			scipPath: filepath.Join(tempDir, "non-existent.scip"),
			setup: func() error {
				return nil
			},
			wantErr:     true,
			errContains: "no such file",
		},
		{
			name:     "invalid SCIP file",
			scipPath: indexPath,
			setup: func() error {
				// Corrupt the SCIP file
				return os.WriteFile(indexPath, []byte("invalid scip content"), 0644)
			},
			wantErr:     true,
			errContains: "failed to parse SCIP file",
		},
		{
			name:     "empty SCIP file",
			scipPath: indexPath,
			setup: func() error {
				// Create an empty SCIP file
				return os.WriteFile(indexPath, []byte{}, 0644)
			},
			wantErr:     true,
			errContains: "empty SCIP file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			if tt.setup != nil {
				err := tt.setup()
				require.NoError(t, err)
			}

			// Parse SCIP file
			nodes, err := ParseSCIPFileForGraph(tt.scipPath)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, nodes)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, nodes)
				if tt.validate != nil {
					tt.validate(t, nodes)
				}
			}
		})
	}
}

func TestParseSCIPFileForGraph_Concurrent(t *testing.T) {
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

	// Create test repository and generate SCIP index
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create a simple TypeScript file
	tsFile := `export const hello = "world";`
	err = os.WriteFile(filepath.Join(testRepo, "index.ts"), []byte(tsFile), 0644)
	require.NoError(t, err)

	// Generate SCIP index
	ctx := context.Background()
	indexPath, err := generator.Generate(ctx, testRepo)
	if err != nil {
		// Skip test if scip-typescript is not installed
		if strings.Contains(err.Error(), "scip-typescript command not found") {
			t.Skip("scip-typescript command not found, skipping test")
		}
		require.NoError(t, err)
	}
	require.FileExists(t, indexPath)

	// Test concurrent parsing
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := ParseSCIPFileForGraph(indexPath)
			done <- err
		}()
	}

	// Wait for both goroutines to complete
	for i := 0; i < 2; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestParseSCIPFileForGraph_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "empty file",
			content:     "",
			expectError: true,
		},
		{
			name:        "invalid SCIP format",
			content:     "invalid content",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tempFile, err := os.CreateTemp("", "scip-test-*")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			// Write test content
			if tt.content != "" {
				_, err = tempFile.WriteString(tt.content)
				require.NoError(t, err)
			}
			tempFile.Close()

			// Parse the file
			graph, err := ParseSCIPFileForGraph(tempFile.Name())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, graph)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, graph)
			}
		})
	}
}

func TestParseSCIPFileForGraph_GoProject(t *testing.T) {
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

	// Create test repository structure
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

	// Generate SCIP index
	ctx := context.Background()
	indexPath, err := generator.Generate(ctx, testRepo)
	require.NoError(t, err)
	require.FileExists(t, indexPath)

	tests := []struct {
		name        string
		scipPath    string
		setup       func() error
		wantErr     bool
		errContains string
		validate    func(t *testing.T, nodes map[string]*SymbolNode)
	}{
		{
			name:     "successful parse",
			scipPath: indexPath,
			setup: func() error {
				return nil
			},
			wantErr: false,
			validate: func(t *testing.T, nodes map[string]*SymbolNode) {
				assert.NotEmpty(t, nodes, "Should have parsed some symbol nodes")

				// Verify User struct
				userStruct, exists := nodes["User"]
				assert.True(t, exists, "Should find User struct")
				if exists {
					assert.Equal(t, "User", userStruct.SymbolName)
					assert.Equal(t, "struct", userStruct.SymbolInfo.Symbol)
				}

				// Verify UserService struct
				userService, exists := nodes["UserService"]
				assert.True(t, exists, "Should find UserService struct")
				if exists {
					assert.Equal(t, "UserService", userService.SymbolName)
					assert.Equal(t, "struct", userService.SymbolInfo.Symbol)
				}

				// Verify methods
				addUserMethod, exists := nodes["AddUser"]
				assert.True(t, exists, "Should find AddUser method")
				if exists {
					assert.Equal(t, "AddUser", addUserMethod.SymbolName)
					assert.Equal(t, "method", addUserMethod.SymbolInfo.Symbol)
				}

				// Verify function
				newUserServiceFunc, exists := nodes["NewUserService"]
				assert.True(t, exists, "Should find NewUserService function")
				if exists {
					assert.Equal(t, "NewUserService", newUserServiceFunc.SymbolName)
					assert.Equal(t, "function", newUserServiceFunc.SymbolInfo.Symbol)
				}
			},
		},
		{
			name:     "non-existent file",
			scipPath: filepath.Join(tempDir, "non-existent.scip"),
			setup: func() error {
				return nil
			},
			wantErr:     true,
			errContains: "no such file",
		},
		{
			name:     "invalid SCIP file",
			scipPath: indexPath,
			setup: func() error {
				// Corrupt the SCIP file
				return os.WriteFile(indexPath, []byte("invalid scip content"), 0644)
			},
			wantErr:     true,
			errContains: "failed to parse SCIP file",
		},
		{
			name:     "empty SCIP file",
			scipPath: indexPath,
			setup: func() error {
				// Create an empty SCIP file
				return os.WriteFile(indexPath, []byte{}, 0644)
			},
			wantErr:     true,
			errContains: "empty SCIP file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			if tt.setup != nil {
				err := tt.setup()
				require.NoError(t, err)
			}

			// Parse SCIP file
			nodes, err := ParseSCIPFileForGraph(tt.scipPath)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, nodes)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, nodes)
				if tt.validate != nil {
					tt.validate(t, nodes)
				}
			}
		})
	}
}

func TestParseSCIPFileForGraph_GoProject_Concurrent(t *testing.T) {
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

	// Create test repository and generate SCIP index
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
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

	// Create go.mod
	err = os.WriteFile(filepath.Join(testRepo, "go.mod"), []byte(`
		module github.com/test/project

		go 1.21
	`), 0644)
	require.NoError(t, err)

	// Generate SCIP index
	ctx := context.Background()
	indexPath, err := generator.Generate(ctx, testRepo)
	require.NoError(t, err)
	require.FileExists(t, indexPath)

	// Test concurrent parsing
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := ParseSCIPFileForGraph(indexPath)
			done <- err
		}()
	}

	// Both parsing operations should succeed
	for i := 0; i < 2; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestParseSymbolNode(t *testing.T) {
	// Create a test symbol node
	node := &SymbolNode{
		SymbolName: "test.symbol",
		SymbolInfo: &scip.SymbolInformation{
			Symbol: "test.symbol",
		},
		DefinitionOcc: &scip.Occurrence{
			Range: []int32{1, 0, 1, 10},
		},
		ReferenceOccs: []*scip.Occurrence{
			{
				Range: []int32{2, 0, 2, 10},
			},
		},
		RelationshipsOut: map[types.NodeType][]string{
			types.NodeTypeReference: {"other.symbol"},
		},
	}

	// Test symbol node parsing
	assert.Equal(t, "test.symbol", node.SymbolName)
	assert.NotNil(t, node.SymbolInfo)
	assert.Equal(t, "test.symbol", node.SymbolInfo.Symbol)
	assert.NotNil(t, node.DefinitionOcc)
	assert.Equal(t, []int32{1, 0, 1, 10}, node.DefinitionOcc.Range)
	assert.Len(t, node.ReferenceOccs, 1)
	assert.Equal(t, []int32{2, 0, 2, 10}, node.ReferenceOccs[0].Range)
	assert.Contains(t, node.RelationshipsOut[types.NodeTypeReference], "other.symbol")
}

func TestSCIPIndexGenerator_Generate_Error(t *testing.T) {
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

	// Test with non-existent repository
	_, err := generator.Generate(context.Background(), "non-existent-repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codebase does not exist")
}

func TestSCIPIndexGenerator_Generate_UnsupportedLanguage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test repository structure
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create an unsupported file
	err = os.WriteFile(filepath.Join(testRepo, "unsupported.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	// Create generator
	generator := NewSCIPIndexGenerator(&Config{})

	// Test Generate
	ctx := context.Background()
	_, err = generator.Generate(ctx, testRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching language configuration found")
}
