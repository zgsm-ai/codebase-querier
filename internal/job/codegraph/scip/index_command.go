package scip

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandExecutor handles command execution for SCIP indexing
type CommandExecutor struct {
	workDir string
	// build
	buildCmds []*exec.Cmd
	indexCmds []*exec.Cmd
	logger    logx.Logger
}

// NewCommandExecutor creates a new CommandExecutor
func NewCommandExecutor(ctx context.Context,
	workDir string, indexTool *IndexTool,
	buildTool *BuildTool, placeHolders map[string]string) (*CommandExecutor, error) {
	if workDir == "" {
		return nil, fmt.Errorf("working dir is required")
	}
	if indexTool == nil || len(indexTool.Commands) == 0 {
		return nil, fmt.Errorf("index commands are required")
	}
	logger := logx.WithContext(ctx)
	indexFileLogger, err := newFileLogger(indexLogDir(workDir))
	if err != nil {
		logger.Errorf("failed to create index log writer: %v", err)
	}
	var logWriter io.Writer
	if indexFileLogger != nil {
		logWriter = indexFileLogger.Writer()
	}
	return &CommandExecutor{
		workDir:   workDir,
		buildCmds: buildBuildCmds(buildTool, workDir, logWriter, placeHolders),
		indexCmds: buildIndexCmds(indexTool, workDir, logWriter, placeHolders),
		logger:    logger,
	}, nil
}

func buildBuildCmds(buildTool *BuildTool, workDir string, logFileWriter io.Writer, placeHolders map[string]string) []*exec.Cmd {
	if logFileWriter == nil {
		logFileWriter = os.Stdout
	}
	var buildCmds []*exec.Cmd
	if buildTool != nil && len(buildTool.Commands) > 0 {
		for _, v := range buildTool.Commands {
			renderedCmd := renderCommand(v, placeHolders)
			buildCmds = append(buildCmds, &exec.Cmd{
				Path:   renderedCmd.Base,
				Args:   renderedCmd.Args,
				Dir:    workDir,
				Env:    renderedCmd.Env,
				Stdout: logFileWriter,
				Stderr: logFileWriter,
			})
		}
	}
	return buildCmds
}

func buildIndexCmds(indexTool *IndexTool, workDir string, logFileWriter io.Writer, placeHolders map[string]string) []*exec.Cmd {
	if logFileWriter == nil {
		logFileWriter = os.Stdout
	}
	var indexCmds []*exec.Cmd
	for _, v := range indexTool.Commands {
		renderedCmd := renderCommand(v, placeHolders)
		indexCmds = append(indexCmds, &exec.Cmd{
			Path:   renderedCmd.Base,
			Args:   renderedCmd.Args,
			Dir:    workDir,
			Env:    renderedCmd.Env,
			Stdout: logFileWriter,
			Stderr: logFileWriter,
		})
	}
	return indexCmds
}

func renderCommand(v *Command, placeHolders map[string]string) *Command {
	v.Base = replacePlaceHolder(v.Base, placeHolders)
	for i, arg := range v.Args {
		v.Args[i] = replacePlaceHolder(arg, placeHolders)
	}

	for i, env := range v.Env {
		v.Env[i] = replacePlaceHolder(env, placeHolders)
	}
	return v
}

func replacePlaceHolder(base string, placeHolders map[string]string) string {
	for key, val := range placeHolders {
		base = strings.ReplaceAll(base, key, val)
	}
	return base
}

// Execute executes a command string
func (e *CommandExecutor) Execute() error {

	e.logger.Debugf("[%s] start to execute command", e.workDir)

	var err error

	for _, cmd := range e.buildCmds {
		if curErr := cmd.Run(); curErr != nil {
			e.logger.Errorf("[%s] buld command execution failed: %v, err: %s", e.workDir, cmd, err)
			err = errors.Join(err, curErr)
		} else {
			e.logger.Debugf("[%s] buld command execution successfully: %v", e.workDir, cmd)
		}
	}

	for _, cmd := range e.indexCmds {
		if curErr := cmd.Run(); curErr != nil {
			e.logger.Errorf("[%s] index command execution failed: %v, err: %s", e.workDir, cmd, err)
			err = errors.Join(err, curErr)
		} else {
			e.logger.Debugf("[%s] index command execution successfully: %v", e.workDir, cmd)
		}
	}

	e.logger.Debugf("[%s] command executed end", e.workDir)
	return err
}

func indexLogDir(baseDir string) string {
	return filepath.Join(baseDir, types.CodebaseIndexDir, "logs")
}
