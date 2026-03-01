package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
)

// SettingsGetter is the interface used by Runner to look up the configured
// claude command. Implementations should return the value of the
// "claude_command" setting (e.g. "claude" or a full path).
type SettingsGetter interface {
	GetSetting(key string) (string, error)
}

// RunResult holds the outcome of a completed Claude CLI invocation.
type RunResult struct {
	Output   string
	Stderr   string
	ExitCode int
	Err      error
}

// Runner manages spawning and tracking claude -p subprocesses.
type Runner struct {
	settings SettingsGetter

	mu      sync.Mutex
	procs   map[int64]*exec.Cmd // active processes keyed by task ID
}

// NewRunner creates a Runner that will use sg to resolve the claude command.
func NewRunner(sg SettingsGetter) *Runner {
	return &Runner{
		settings: sg,
		procs:    make(map[int64]*exec.Cmd),
	}
}

// Run spawns `claude -p` with the given prompt, streams parsed events through
// onEvent, and returns a RunResult when the process exits.
//
// The process is killed if ctx is cancelled. taskID is used as the key for
// Kill(); pass 0 if kill support is not needed.
func (r *Runner) Run(ctx context.Context, taskID int64, repoPath string, prompt string, onEvent func(Event)) RunResult {
	command := "claude"
	if r.settings != nil {
		if v, err := r.settings.GetSetting("claude_command"); err == nil && v != "" {
			command = v
		}
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
		"--max-turns", "200",
		prompt,
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = repoPath
	// Use a process group so we can kill the entire tree on cancellation.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult{Err: fmt.Errorf("stdout pipe: %w", err)}
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return RunResult{Err: fmt.Errorf("start claude process: %w", err)}
	}

	// Track the running process so Kill can reach it.
	if taskID != 0 {
		r.mu.Lock()
		r.procs[taskID] = cmd
		r.mu.Unlock()

		defer func() {
			r.mu.Lock()
			delete(r.procs, taskID)
			r.mu.Unlock()
		}()
	}

	var fullOutput string

	scanner := bufio.NewScanner(stdout)
	// Allow for large JSON lines (default 64 KiB may be too small).
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		ev, parseErr := ParseEvent(line)
		if parseErr != nil {
			continue
		}
		if ev == nil {
			// empty line
			continue
		}
		fullOutput += ev.Content
		if onEvent != nil {
			onEvent(*ev)
		}
	}

	waitErr := cmd.Wait()
	stderrStr := stderrBuf.String()

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RunResult{Output: fullOutput, Stderr: stderrStr, ExitCode: -1, Err: fmt.Errorf("wait: %w", waitErr)}
		}
	}

	return RunResult{
		Output:   fullOutput,
		Stderr:   stderrStr,
		ExitCode: exitCode,
	}
}

// Kill terminates the process associated with the given task ID (if any).
// It sends SIGKILL to the entire process group so child processes are also
// cleaned up.
func (r *Runner) Kill(taskID int64) {
	r.mu.Lock()
	cmd, ok := r.procs[taskID]
	r.mu.Unlock()
	if !ok || cmd.Process == nil {
		return
	}
	// Kill the process group (negative PID).
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
