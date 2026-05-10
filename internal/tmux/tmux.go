package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var sep = "\x1f"

type Tmux struct {
	bin    string
	socket string
	delay  time.Duration
}

func New(bin, socket string) (*Tmux, error) {
	return &Tmux{bin: bin, socket: socket, delay: 150 * time.Millisecond}, nil
}

type PaneId string

type Pane struct {
	Id                   PaneId
	Title, PID, Cmd, Cwd string
}

func (t *Tmux) ListPanes(ctx context.Context) ([]Pane, error) {
	format := []string{"#{pane_id}", "#{pane_title}", "#{pane_pid}", "#{pane_current_command}", "#{pane_current_path}"}
	out, err := t.run(ctx, "list-panes", "-F", strings.Join(format, sep))
	if err != nil {
		return nil, fmt.Errorf("list-panes: %w", err)
	}

	panes := make([]Pane, 0)
	for line := range strings.Lines(strings.TrimSpace(string(out))) {
		if line == "" {
			continue
		}
		parts := strings.Split(strings.TrimSpace(line), sep)
		if len(parts) != len(format) {
			return nil, fmt.Errorf("parse pane: %s", line)
		}
		panes = append(panes, Pane{Id: PaneId(parts[0]), Title: parts[1], PID: parts[2], Cmd: parts[3], Cwd: parts[4]})
	}

	return panes, nil
}

type Session struct {
	Id, Name string
}

func (t *Tmux) ListSessions(ctx context.Context) ([]Session, error) {
	format := []string{"#{session_id}", "#{session_name}"}
	out, err := t.run(ctx, "list-sessions", "-F", strings.Join(format, sep))
	if err != nil {
		return nil, fmt.Errorf("list-sessions: %w", err)
	}

	sessions := make([]Session, 0)
	for line := range strings.Lines(strings.TrimSpace(string(out))) {
		if line == "" {
			continue
		}
		parts := strings.Split(strings.TrimSpace(line), sep)
		if len(parts) != len(format) {
			return nil, fmt.Errorf("parse session: %s", line)
		}
		sessions = append(sessions, Session{Id: parts[0], Name: parts[1]})
	}

	return sessions, nil
}

func (t *Tmux) Writeln(ctx context.Context, pane PaneId, keys string) error {
	if err := t.sendKeys(ctx, pane, true, keys); err != nil {
		return fmt.Errorf("write literal: %w", err)
	}

	select {
	case <-time.After(t.delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := t.sendKeys(ctx, pane, false, "Enter"); err != nil {
		return fmt.Errorf("write <Enter>: %w", err)
	}

	return nil
}

func (t *Tmux) sendKeys(ctx context.Context, pane PaneId, literal bool, keys string) error {
	args := []string{"-t", string(pane)}

	if literal {
		args = append(args, "-l")
	}

	args = append(args, keys)

	_, err := t.run(ctx, "send-keys", args...)

	if err != nil {
		return fmt.Errorf("send-keys: %w", err)
	}
	return nil
}

type runError struct {
	op     string
	args   []string
	stderr string
	err    error
}

func (e *runError) Error() string {
	return fmt.Sprintf("tmux %s: %s: stderr: %s", e.op, e.err, e.stderr)
}

func (e *runError) Unwrap() error {
	return e.err
}

func (t *Tmux) run(ctx context.Context, op string, args ...string) (string, error) {
	cmdArgs := append([]string{"-S", t.socket, op}, args...)
	cmd := exec.CommandContext(ctx, t.bin, cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &runError{op: op, args: cmdArgs, stderr: stderr.String(), err: err}
	}

	return stdout.String(), nil
}
