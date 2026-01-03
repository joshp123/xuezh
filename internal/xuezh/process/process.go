package process

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type ToolMissingError struct {
	Tool string
}

func (e ToolMissingError) Error() string {
	return fmt.Sprintf("Required tool not found on PATH: %s", e.Tool)
}

type ProcessFailedError struct {
	Cmd        []string
	ReturnCode int
	Stdout     string
	Stderr     string
}

func (e ProcessFailedError) Error() string {
	return fmt.Sprintf("Process failed with code %d: %s", e.ReturnCode, strings.Join(e.Cmd, " "))
}

type ProcessResult struct {
	Stdout     string
	Stderr     string
	ReturnCode int
}

func EnsureTool(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", ToolMissingError{Tool: name}
	}
	return path, nil
}

func RunChecked(cmd []string) (ProcessResult, error) {
	if len(cmd) == 0 {
		return ProcessResult{}, errors.New("empty command")
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	stdout, err := c.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return ProcessResult{}, ProcessFailedError{Cmd: cmd, ReturnCode: exitErr.ExitCode(), Stdout: string(stdout), Stderr: string(exitErr.Stderr)}
		}
		return ProcessResult{}, err
	}
	return ProcessResult{Stdout: string(stdout), Stderr: "", ReturnCode: 0}, nil
}
