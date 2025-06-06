package codebase

import (
	"context"
	"errors"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
)

// Store defines the interface for codebase storage operations
type Store interface {
	// Init initializes a new codebase for a client
	// Returns the initialized codebase or an error if initialization fails
	Init(ctx context.Context, clientId string, clientCodebasePath string) (*types.Codebase, error)

	// Add adds a file to the codebase
	// source: the source reader containing the file content
	// target: the target path where the file should be stored
	Add(ctx context.Context, codebasePath string, source io.Reader, target string) error

	// Unzip extracts a zip file into the codebase
	// source: the source reader containing the zip file
	// target: the target directory where files should be extracted
	Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error

	// Delete removes a file or directory from the codebase
	// path: the path to the file or directory to delete
	Delete(ctx context.Context, codebasePath string, path string) error

	// MkDirs creates directories in the codebase
	// path: the path where directories should be created
	MkDirs(ctx context.Context, codebasePath string, path string) error

	// Exists checks if a path exists in the codebase
	// Returns true if the path exists, false otherwise
	Exists(ctx context.Context, codebasePath string, path string) (bool, error)

	// Stat returns information about a file or directory
	// Returns FileInfo or an error if the path doesn't exist
	Stat(ctx context.Context, codebasePath string, path string) (*types.FileInfo, error)

	// List lists files and directories in a directory
	// dir: the directory to list
	// option: optional parameters for listing
	List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error)

	// Tree returns a tree structure of the codebase
	// dir: the root directory for the tree
	// option: optional parameters for tree construction
	Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error)

	// Read reads the content of a file
	// filePath: the path to the file to read
	// option: optional parameters for reading
	Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) ([]byte, error)

	// Open reads the content of a file
	// filePath: the path to the file to read
	// option: optional parameters for reading
	Open(ctx context.Context, codebasePath string, filePath string) (io.ReadSeekCloser, error)

	// Walk walks through the codebase and processes each file
	// dir: the root directory to start walking from
	// process: function to process each file
	Walk(ctx context.Context, codebasePath string, dir string, walkFn WalkFunc) error

	// BatchDelete deletes multiple files or directories
	// paths: list of paths to delete
	BatchDelete(ctx context.Context, codebasePath string, paths []string) error
	DeleteAll(ctx context.Context, codebasePath string) error
}

// WalkContext provides context information during directory traversal
type WalkContext struct {
	// Current file or directory being processed
	Path string
	// Relative path from the root directory
	RelativePath string
	// File information
	Info *types.FileInfo
	// Parent directory path
	ParentPath string
}

// WalkFunc is the type of the function called for each file or directory

type WalkFunc func(walkCtx *WalkContext, reader io.ReadCloser) error

// SkipDir is used as a return value from WalkFuncs to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var SkipDir = errors.New("skip this directory")
