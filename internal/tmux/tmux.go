package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

var sep = "\x1f"

type Tmux struct {
	bin    string
	socket string
}

func New(bin, socket string) (*Tmux, error) {
	return &Tmux{bin, socket}, nil
}

type Pane struct {
	Id, Title, PID, Cmd, Cwd string
}

func (t *Tmux) ListPanes(ctx context.Context) ([]Pane, error) {
	format := []string{"#{pane_id}", "#{pane_title}", "#{pid}", "#{pane_current_command}", "#{pane_current_path}"}
	out, err := runCmd(t.bin, t.withDefaults("list-panes", "-F", strings.Join(format, sep))...)
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
		panes = append(panes, Pane{Id: parts[0], Title: parts[1], PID: parts[2], Cmd: parts[3], Cwd: parts[4]})
	}

	return panes, nil
}

type Session struct {
	Id, Name string
}

func (t *Tmux) ListSessions(ctx context.Context) ([]Session, error) {
	format := []string{"#{session_id}", "#{session_name}"}
	out, err := runCmd(t.bin, t.withDefaults("list-sessions", "-F", strings.Join(format, sep))...)
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

func (t *Tmux) withDefaults(args ...string) []string {
	finalArgs := make([]string, 0, len(args)+2)
	finalArgs = append(finalArgs, "-S", t.socket)
	finalArgs = append(finalArgs, args...)

	return finalArgs
}

func runCmd(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cmd run: %w", err)
	}

	return string(out), nil
}
