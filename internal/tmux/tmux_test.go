package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type tempTmux struct {
	socket string
}

func newTempTmux(socket string) (*tempTmux, error) {
	cmd := exec.Command("tmux", "-S", socket, "new-session", "-d")
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("start test tmux: %w: %s", err, string(out))
	}

	return &tempTmux{socket}, nil
}

func (t *tempTmux) close() error {
	cmd := exec.Command("tmux", "-S", t.socket, "kill-server")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kill test tmux: %w: %s", err, string(out))
	}

	return nil
}

func TestListSessions(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "tmux.sock")
	tm, err := newTempTmux(socket)
	if err != nil {
		t.Fatalf("temp tmux unexpected error: %s", err)
	}

	defer tm.close()

	client, err := New("tmux", tm.socket)
	if err != nil {
		t.Fatalf("New unexpected error: %s", err)
	}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions unexpected error: %s", err)
	}

	expected := []Session{{ID: "$0", Name: "0"}}

	if !slices.Equal(sessions, expected) {
		t.Fatalf("ListSessions() = %+v, want %+v", sessions, expected)
	}
}

func TestListPanes(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "tmux.sock")
	tm, err := newTempTmux(socket)
	if err != nil {
		t.Fatalf("temp tmux unexpected error: %s", err)
	}

	defer tm.close()

	client, err := New("tmux", tm.socket)
	if err != nil {
		t.Fatalf("New unexpected error: %s", err)
	}

	panes, err := client.ListPanes(context.Background(), "0")
	if err != nil {
		t.Fatalf("ListPanes unexpected error: %s", err)
	}

	if len(panes) != 1 {
		t.Fatalf("ListPanes() returned %d panes, want 1: %+v", len(panes), panes)
	}

	pane := panes[0]
	if pane.Id != "%0" {
		t.Fatalf("ListPanes()[0].Id = %q, want %%0", pane.Id)
	}

	if pane.PID == "" {
		t.Fatal("ListPanes()[0].PID is empty")
	}

	if pane.Cmd == "" {
		t.Fatal("ListPanes()[0].Cmd is empty")
	}

	if pane.Cwd == "" {
		t.Fatal("ListPanes()[0].Cwd is empty")
	}
}

func TestWritelnAndCapture(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "tmux.sock")
	tm, err := newTempTmux(socket)
	if err != nil {
		t.Fatalf("temp tmux unexpected error: %s", err)
	}

	defer tm.close()

	client, err := New("tmux", tm.socket)
	if err != nil {
		t.Fatalf("New unexpected error: %s", err)
	}

	marker := "remote-writeln-capture"
	if err := client.Writeln(context.Background(), "%0", fmt.Sprintf("printf '%s\\n'", marker)); err != nil {
		t.Fatalf("Writeln unexpected error: %s", err)
	}

	out, err := client.Capture(context.Background(), "%0")
	if err != nil {
		t.Fatalf("Capture unexpected error: %s", err)
	}

	if !strings.Contains(out, marker) {
		t.Fatalf("Capture() does not contain %q:\n%s", marker, out)
	}
}
