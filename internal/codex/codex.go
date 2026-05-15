package codex

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alexghr/remote/internal/process"
	"github.com/alexghr/remote/internal/tmux"
)

type Codex struct {
	tmux *tmux.Tmux
	proc process.ProcessMonitor
}

func New(tmux *tmux.Tmux, proc process.ProcessMonitor) (*Codex, error) {
	return &Codex{
		tmux: tmux,
		proc: proc,
	}, nil
}

type CodexPane struct {
	PaneID tmux.PaneID `json:"paneId"`
	PID    int         `json:"pid"`
}

func (c *Codex) ListPanes(ctx context.Context) ([]CodexPane, error) {
	return c.listPanes(ctx)
}

func (c *Codex) Capture(ctx context.Context, pane tmux.PaneID) (string, error) {
	if err := c.validatePane(ctx, pane); err != nil {
		return "", err
	}

	out, err := c.tmux.Capture(ctx, pane)
	if err != nil {
		return "", fmt.Errorf("codex pane capture: %w", err)
	}

	return out, nil
}

func (c *Codex) Prompt(ctx context.Context, pane tmux.PaneID, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return ErrBadPrompt
	}

	if err := c.validatePane(ctx, pane); err != nil {
		return err
	}

	if err := c.tmux.Writeln(ctx, pane, prompt); err != nil {
		return fmt.Errorf("codex prompt: %w", err)
	}

	return nil
}

func (c *Codex) SendKeys(ctx context.Context, pane tmux.PaneID, keys []string) error {
	if len(keys) == 0 {
		return ErrBadKeys
	}

	for _, key := range keys {
		if !allowedKey(key) {
			return ErrBadKeys
		}
	}

	if err := c.validatePane(ctx, pane); err != nil {
		return err
	}

	if err := c.tmux.SendKeys(ctx, pane, keys...); err != nil {
		return fmt.Errorf("codex send keys: %w", err)
	}

	return nil
}

var ErrPaneNotFound = errors.New("pane not found")
var ErrBadPrompt = errors.New("bad prompt")
var ErrBadKeys = errors.New("bad keys")

func allowedKey(key string) bool {
	switch key {
	case "Up", "Down", "Enter", "Escape", "Tab":
		return true
	default:
		return false
	}
}

func (c *Codex) validatePane(ctx context.Context, pane tmux.PaneID) error {
	panes, err := c.listPanes(ctx)
	if err != nil {
		return fmt.Errorf("validate codex pane: %w", err)
	}

	for _, p := range panes {
		if p.PaneID == pane {
			return nil
		}
	}

	return ErrPaneNotFound
}

func (c *Codex) listPanes(ctx context.Context) ([]CodexPane, error) {
	panes, err := c.tmux.ListPanes(ctx)
	if err != nil {
		return nil, fmt.Errorf("tmux panes: %w", err)
	}

	snapshot, err := c.proc.TakeSnapshot()
	if err != nil {
		return nil, fmt.Errorf("process snapshot: %w", err)
	}

	codexPanes := make([]CodexPane, 0)
	for _, pane := range panes {
		codexProc, ok := findCodexProcess(snapshot, pane)
		if !ok {
			continue
		}

		codexPanes = append(codexPanes, CodexPane{
			PaneID: pane.ID,
			PID:    codexProc.PID,
		})
	}

	return codexPanes, nil
}

func findCodexProcess(snapshot *process.Snapshot, pane tmux.Pane) (process.Process, bool) {
	candidates := make([]process.Process, 0, 1)

	if root, ok := snapshot.Find(pane.PID); ok {
		candidates = append(candidates, root)
	}

	candidates = append(candidates, snapshot.DescendantsOf(pane.PID)...)

	for _, proc := range candidates {
		if proc.Comm == "codex" {
			return proc, true
		}
	}

	for _, proc := range candidates {
		if len(proc.Args) == 0 {
			continue
		}

		if filepath.Base(proc.Args[0]) == "codex" {
			return proc, true
		}

		for _, arg := range proc.Args[1:] {
			if filepath.Base(arg) == "codex" {
				return proc, true
			}
		}
	}

	return process.Process{}, false
}
