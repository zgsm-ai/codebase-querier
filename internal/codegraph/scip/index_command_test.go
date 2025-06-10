package scip

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestNewCommandExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		workDir      string
		indexTool    *IndexTool
		buildTool    *BuildTool
		placeHolders map[string]string
		wantErr      bool
		errContains  string
	}{
		{
			name:        "empty work dir",
			workDir:     "",
			wantErr:     true,
			errContains: "working dir is required",
		},
		{
			name:        "nil index tool",
			workDir:     "/tmp",
			wantErr:     true,
			errContains: "index commands are required",
		},
		{
			name:    "empty index commands",
			workDir: "/tmp",
			indexTool: &IndexTool{
				Name:     "test",
				Commands: []*Command{},
			},
			wantErr:     true,
			errContains: "index commands are required",
		},
		{
			name:    "valid config",
			workDir: "/tmp",
			indexTool: &IndexTool{
				Name: "test",
				Commands: []*Command{
					{
						Base: "test-cmd",
						Args: []string{"arg1", "arg2"},
					},
				},
			},
			placeHolders: map[string]string{
				"${WORK_DIR}": "/tmp",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			executor, err := newCommandExecutor(ctx, tt.workDir, tt.indexTool, tt.buildTool, "", tt.placeHolders)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, executor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, executor)
			}
		})
	}
}

func TestReplacePlaceHolder(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		placeHolders map[string]string
		expected     string
	}{
		{
			name:         "no placeholders",
			input:        "test string",
			placeHolders: map[string]string{},
			expected:     "test string",
		},
		{
			name:         "single placeholder",
			input:        "test ${PLACEHOLDER} string",
			placeHolders: map[string]string{"${PLACEHOLDER}": "value"},
			expected:     "test value string",
		},
		{
			name:  "multiple placeholders",
			input: "${START}test${MIDDLE}string${END}",
			placeHolders: map[string]string{
				"${START}":  "begin-",
				"${MIDDLE}": "-middle-",
				"${END}":    "-end",
			},
			expected: "begin-test-middle-string-end",
		},
		{
			name:         "placeholder not found",
			input:        "test ${NOT_FOUND} string",
			placeHolders: map[string]string{"${OTHER}": "value"},
			expected:     "test ${NOT_FOUND} string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replacePlaceHolder(tt.input, tt.placeHolders)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderCommand(t *testing.T) {
	tests := []struct {
		name         string
		command      *Command
		placeHolders map[string]string
		expected     *Command
	}{
		{
			name: "render all fields",
			command: &Command{
				Base: "${BASE}",
				Args: []string{"${ARG1}", "${ARG2}"},
				Env:  []string{"${ENV1}=value1", "${ENV2}=value2"},
			},
			placeHolders: map[string]string{
				"${BASE}": "test-cmd",
				"${ARG1}": "arg1",
				"${ARG2}": "arg2",
				"${ENV1}": "ENV1",
				"${ENV2}": "ENV2",
			},
			expected: &Command{
				Base: "test-cmd",
				Args: []string{"arg1", "arg2"},
				Env:  []string{"ENV1=value1", "ENV2=value2"},
			},
		},
		{
			name: "no placeholders",
			command: &Command{
				Base: "test-cmd",
				Args: []string{"arg1", "arg2"},
				Env:  []string{"ENV1=value1"},
			},
			placeHolders: map[string]string{},
			expected: &Command{
				Base: "test-cmd",
				Args: []string{"arg1", "arg2"},
				Env:  []string{"ENV1=value1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderCommand(tt.command, tt.placeHolders)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommandExecutor_Execute(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()

	// Determine script extension and content based on OS
	var scriptExt, successContent, failContent string
	if runtime.GOOS == "windows" {
		scriptExt = ".bat"
		successContent = "@echo off\nexit /b 0"
		failContent = "@echo off\nexit /b 1"
	} else {
		scriptExt = ".sh"
		successContent = "#!/bin/sh\nexit 0"
		failContent = "#!/bin/sh\nexit 1"
	}

	// Create a test script that will succeed
	successScript := filepath.Join(tmpDir, "success"+scriptExt)
	err := os.WriteFile(successScript, []byte(successContent), 0755)
	assert.NoError(t, err)

	// Create a test script that will fail
	failScript := filepath.Join(tmpDir, "fail"+scriptExt)
	err = os.WriteFile(failScript, []byte(failContent), 0755)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		buildCmds []*Command
		indexCmds []*Command
		wantErr   bool
	}{
		{
			name: "all commands succeed",
			buildCmds: []*Command{
				{
					Base: successScript,
					Args: []string{},
				},
			},
			indexCmds: []*Command{
				{
					Base: successScript,
					Args: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "build command fails",
			buildCmds: []*Command{
				{
					Base: failScript,
					Args: []string{},
				},
			},
			indexCmds: []*Command{
				{
					Base: successScript,
					Args: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "index command fails",
			buildCmds: []*Command{
				{
					Base: successScript,
					Args: []string{},
				},
			},
			indexCmds: []*Command{
				{
					Base: failScript,
					Args: []string{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			executor := &commandExecutor{
				workDir:   tmpDir,
				buildCmds: buildBuildCmds(&BuildTool{Commands: tt.buildCmds}, tmpDir, nil, nil),
				indexCmds: buildIndexCmds(&IndexTool{Commands: tt.indexCmds}, tmpDir, nil, nil),
				logger:    logx.WithContext(ctx),
			}

			err := executor.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
