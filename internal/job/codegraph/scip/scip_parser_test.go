package scip

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSCIPFileForGraph tests the SCIP index parsing function.
func TestParseSCIPFileForGraph(t *testing.T) {
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
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

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
		validate    func(t *testing.T, nodes []SymbolNode)
	}{
		{
			name:     "successful parse",
			scipPath: indexPath,
			setup: func() error {
				return nil
			},
			wantErr: false,
			validate: func(t *testing.T, nodes []SymbolNode) {
				assert.NotEmpty(t, nodes, "Should have parsed some symbol nodes")
				
				// Verify User interface
				var userInterface *SymbolNode
				for _, node := range nodes {
					if node.Name == "User" && node.Kind == "interface" {
						userInterface = &node
						break
					}
				}
				assert.NotNil(t, userInterface, "Should find User interface")
				if userInterface != nil {
					assert.Equal(t, "interface", userInterface.Kind)
					assert.Contains(t, userInterface.Location, "user.ts")
				}

				// Verify UserService class
				var userService *SymbolNode
				for _, node := range nodes {
					if node.Name == "UserService" && node.Kind == "class" {
						userService = &node
						break
					}
				}
				assert.NotNil(t, userService, "Should find UserService class")
				if userService != nil {
					assert.Equal(t, "class", userService.Kind)
					assert.Contains(t, userService.Location, "user.ts")
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
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

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

func TestParseSCIPFileForGraph_EdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		content     []byte
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty file",
			content:     []byte{},
			wantErr:     true,
			errContains: "empty SCIP file",
		},
		{
			name:        "invalid protobuf",
			content:     []byte("invalid protobuf content"),
			wantErr:     true,
			errContains: "failed to parse SCIP file",
		},
		{
			name:        "truncated protobuf",
			content:     []byte{0x0A, 0x0B}, // Minimal protobuf header
			wantErr:     true,
			errContains: "failed to parse SCIP file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test SCIP file
			scipPath := filepath.Join(tempDir, "test.scip")
			err := os.WriteFile(scipPath, tt.content, 0644)
			require.NoError(t, err)

			// Parse SCIP file
			nodes, err := ParseSCIPFileForGraph(scipPath)

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
			}
		})
	}
}

func TestParseSCIPFileForGraph_GoProject(t *testing.T) {
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
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

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
		validate    func(t *testing.T, nodes []SymbolNode)
	}{
		{
			name:     "successful parse",
			scipPath: indexPath,
			setup: func() error {
				return nil
			},
			wantErr: false,
			validate: func(t *testing.T, nodes []SymbolNode) {
				assert.NotEmpty(t, nodes, "Should have parsed some symbol nodes")
				
				// Verify User struct
				var userStruct *SymbolNode
				for _, node := range nodes {
					if node.Name == "User" && node.Kind == "struct" {
						userStruct = &node
						break
					}
				}
				assert.NotNil(t, userStruct, "Should find User struct")
				if userStruct != nil {
					assert.Equal(t, "struct", userStruct.Kind)
					assert.Contains(t, userStruct.Location, "main.go")
				}

				// Verify UserService struct
				var userService *SymbolNode
				for _, node := range nodes {
					if node.Name == "UserService" && node.Kind == "struct" {
						userService = &node
						break
					}
				}
				assert.NotNil(t, userService, "Should find UserService struct")
				if userService != nil {
					assert.Equal(t, "struct", userService.Kind)
					assert.Contains(t, userService.Location, "main.go")
				}

				// Verify methods
				var addUserMethod *SymbolNode
				for _, node := range nodes {
					if node.Name == "AddUser" && node.Kind == "method" {
						addUserMethod = &node
						break
					}
				}
				assert.NotNil(t, addUserMethod, "Should find AddUser method")
				if addUserMethod != nil {
					assert.Equal(t, "method", addUserMethod.Kind)
					assert.Contains(t, addUserMethod.Location, "main.go")
				}

				// Verify function
				var newUserServiceFunc *SymbolNode
				for _, node := range nodes {
					if node.Name == "NewUserService" && node.Kind == "function" {
						newUserServiceFunc = &node
						break
					}
				}
				assert.NotNil(t, newUserServiceFunc, "Should find NewUserService function")
				if newUserServiceFunc != nil {
					assert.Equal(t, "function", newUserServiceFunc.Kind)
					assert.Contains(t, newUserServiceFunc.Location, "main.go")
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
	generator, err := NewSCIPIndexGenerator()
	require.NoError(t, err)

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
