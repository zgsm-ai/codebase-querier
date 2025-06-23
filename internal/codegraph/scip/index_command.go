package scip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// CommandExecutor handles command execution for SCIP indexing
type CommandExecutor struct {
	workDir string
	// build
	BuildCmds       []*exec.Cmd
	IndexCmds       []*exec.Cmd
	cmdLoggerWriter io.Writer
}

// newCommandExecutor creates a new CommandExecutor
func newCommandExecutor(
	cmdLogger *tracer.CmdLogger,
	workDir string,
	indexTool *config.IndexTool,
	buildTool *config.BuildTool, placeHolders map[string]string) (*CommandExecutor, error) {
	if workDir == types.EmptyString {
		return nil, fmt.Errorf(" scip index generator working dir is required")
	}
	if indexTool == nil || len(indexTool.Commands) == 0 {
		return nil, fmt.Errorf("[%s] scip index generator index commands are required", workDir)
	}
	hostname, err := os.Hostname()
	if err != nil {
		logx.Errorf("[%s] scip index generator failed to get hostname:%v", workDir, err)
		hostname = uuid.New().String()
	}
	var writer io.Writer
	if cmdLogger == nil {
		logx.Errorf("[%s] scip index generator cmdLogger is nil, use stdout.", workDir)
		writer = os.Stdout
	} else {
		writer, err = cmdLogger.GetWriter(hostname)
		if err != nil {
			logx.Errorf("[%s] scip index generator failed to get cmdLogger writer, use stdout. err: %v", workDir, err)
			writer = os.Stdout
		}
	}

	return &CommandExecutor{
		cmdLoggerWriter: writer,
		workDir:         workDir,
		BuildCmds:       buildBuildCmds(buildTool, workDir, writer, placeHolders),
		IndexCmds:       buildIndexCmds(indexTool, workDir, writer, placeHolders),
	}, nil
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

	tracer.WithTrace(ctx).Infof("[%s] scip index generator start to execute index commands.", e.workDir)
	e.cmdLoggerWriter.Write([]byte("-----------------------------------"))
	_, logErr := e.cmdLoggerWriter.Write([]byte(fmt.Sprintf("[%s] start to execute index command", e.workDir)))
	if logErr != nil {
		tracer.WithTrace(ctx).Errorf("[%s] scip index generator command write log err:%v", logErr)
	}

	var err error

	for _, cmd := range e.BuildCmds {
		tracer.WithTrace(ctx).Infof("[%s] scip index generator start to execute build command: %v", e.workDir, cmd)
		if curErr := cmd.Run(); curErr != nil {
			tracer.WithTrace(ctx).Errorf("[%s] scip index generator build command execution failed: %v, err: %s", e.workDir, cmd, curErr)
			err = errors.Join(err, curErr)
		} else {
			tracer.WithTrace(ctx).Infof("[%s] scip index generator build command execution successfully: %v", e.workDir, cmd)
		}
	}

	for _, cmd := range e.IndexCmds {
		tracer.WithTrace(ctx).Infof("[%s] scip index generator start to execute index command: %v", e.workDir, cmd)
		if curErr := cmd.Run(); curErr != nil {
			tracer.WithTrace(ctx).Errorf("[%s] scip index generator index command execution failed: %v, err: %v", e.workDir, cmd, curErr)
			err = errors.Join(err, curErr)
		} else {
			tracer.WithTrace(ctx).Infof("[%s] scip index generator index command execution successfully: %v", e.workDir, cmd)
		}
	}

	tracer.WithTrace(ctx).Infof("[%s] scip index generator command executed end, cost: %d ms", e.workDir, time.Since(start).Milliseconds())

	_, logErr = e.cmdLoggerWriter.Write([]byte(fmt.Sprintf("[%s] index commands executed end, cost: %d ms",
		e.workDir, time.Since(start).Milliseconds())))
	if logErr != nil {
		tracer.WithTrace(ctx).Errorf("[%s] scip index generator command write log err:%v", err)
	}

	return err
}

func (e *CommandExecutor) Close() error {
	logx.Infof("[%s] scip index generator  closed successfully.", e.workDir)
	return nil
}
