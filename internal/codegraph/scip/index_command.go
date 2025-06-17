package scip

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandExecutor handles command execution for SCIP indexing
type CommandExecutor struct {
	workDir string
	// build
	BuildCmds          []*exec.Cmd
	IndexCmds          []*exec.Cmd
	indexFileLogWriter io.Writer
}

// newCommandExecutor creates a new CommandExecutor
func newCommandExecutor(ctx context.Context,
	workDir string,
	indexTool *config.IndexTool,
	buildTool *config.BuildTool,
	logDir string,
	placeHolders map[string]string) (*CommandExecutor, error) {
	if workDir == "" {
		return nil, fmt.Errorf("working dir is required")
	}
	if indexTool == nil || len(indexTool.Commands) == 0 {
		return nil, fmt.Errorf("index commands are required")
	}
	indexFileLogger, err := newFileLogWriter(logDir, logFileNamePrefix(workDir))
	if err != nil {
		logx.Errorf("failed to create index log writer: %v", err)
	}
	var logWriter io.Writer
	if indexFileLogger != nil {
		logWriter = indexFileLogger
	} else {
		logWriter = os.Stdout
	}
	return &CommandExecutor{
		workDir:            workDir,
		BuildCmds:          buildBuildCmds(buildTool, workDir, logWriter, placeHolders),
		IndexCmds:          buildIndexCmds(indexTool, workDir, logWriter, placeHolders),
		indexFileLogWriter: logWriter,
	}, nil
}

func logFileNamePrefix(workDir string) string {
	return strings.ReplaceAll(workDir, "/", "_")
}

func buildBuildCmds(buildTool *config.BuildTool, workDir string, logFileWriter io.Writer, placeHolders map[string]string) []*exec.Cmd {
	if logFileWriter == nil {
		logFileWriter = os.Stdout
	}
	var buildCmds []*exec.Cmd
	if buildTool != nil && len(buildTool.Commands) > 0 {
		for _, v := range buildTool.Commands {
			renderedCmd := renderCommand(v, placeHolders)
			cmd := exec.Command(renderedCmd.Base, renderedCmd.Args...)
			cmd.Dir = workDir
			cmd.Env = renderedCmd.Env
			cmd.Stdout = logFileWriter
			cmd.Stderr = logFileWriter
			buildCmds = append(buildCmds, cmd)
		}
	}
	return buildCmds
}

func buildIndexCmds(indexTool *config.IndexTool, workDir string, logFileWriter io.Writer, placeHolders map[string]string) []*exec.Cmd {
	if logFileWriter == nil {
		logFileWriter = os.Stdout
	}
	var indexCmds []*exec.Cmd
	for _, v := range indexTool.Commands {
		renderedCmd := renderCommand(v, placeHolders)
		cmd := exec.Command(renderedCmd.Base, renderedCmd.Args...)
		cmd.Dir = workDir
		cmd.Env = renderedCmd.Env
		cmd.Stdout = logFileWriter
		cmd.Stderr = logFileWriter
		indexCmds = append(indexCmds, cmd)
	}
	return indexCmds
}

func renderCommand(v *config.Command, placeHolders map[string]string) *config.Command {
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
func (e *CommandExecutor) Execute(ctx context.Context) error {
	start := time.Now()

	tracer.WithTrace(ctx).Debugf("[%s] start to execute index command", e.workDir)
	indexLogInfo(e.indexFileLogWriter, "[%s] start to execute index commands", e.workDir)

	var err error

	for _, cmd := range e.BuildCmds {
		indexLogInfo(e.indexFileLogWriter, "[%s] start to execute build command: %v", e.workDir, cmd)
		if curErr := cmd.Run(); curErr != nil {
			tracer.WithTrace(ctx).Errorf("[%s] build command execution failed: %v, err: %s", e.workDir, cmd, curErr)
			indexLogInfo(e.indexFileLogWriter, "[%s] build command execution failed:%v", e.workDir, curErr)
			err = errors.Join(err, curErr)
		} else {
			tracer.WithTrace(ctx).Debugf("[%s] build command execution successfully: %v", e.workDir, cmd)
			indexLogInfo(e.indexFileLogWriter, "[%s] build command execution successfully", e.workDir)
		}
	}

	for _, cmd := range e.IndexCmds {
		indexLogInfo(e.indexFileLogWriter, "[%s] start to execute index command: %v", e.workDir, cmd)
		if curErr := cmd.Run(); curErr != nil {
			tracer.WithTrace(ctx).Errorf("[%s] index command execution failed: %v, err: %v", e.workDir, cmd, curErr)
			indexLogInfo(e.indexFileLogWriter, "[%s] build command execution failed: %v", e.workDir, curErr)
			err = errors.Join(err, curErr)
		} else {
			tracer.WithTrace(ctx).Debugf("[%s] index command execution successfully: %v", e.workDir, cmd)
			indexLogInfo(e.indexFileLogWriter, "[%s] index command execution successfully", e.workDir)
		}
	}

	tracer.WithTrace(ctx).Debugf("[%s] index commands executed end, cost: %d ms", e.workDir, time.Since(start).Milliseconds())
	indexLogInfo(e.indexFileLogWriter,
		"[%s] index commands executed end, cost: %d ms\n", e.workDir, time.Since(start).Milliseconds())
	return err
}

func (e *CommandExecutor) Close() error {
	if e.indexFileLogWriter == nil {
		logx.Errorf("index command file log write is nil ,not close.")
	}
	if writer, ok := e.indexFileLogWriter.(io.WriteCloser); ok {
		if err := writer.Close(); err != nil {
			logx.Errorf("failed to close index log writer: %v", err)
		}
	}
	return nil
}
