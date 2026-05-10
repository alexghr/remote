package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

type tempTmux struct {
	socket string
}

func newTempTmux() (*tempTmux, error) {
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("tmux_test_%d.sock", os.Getpid()))
	cmd := exec.Command("tmux", "-S", socket, "new-session", "-d")
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("start test tmux: %w: %s", err, string(out))
	}

	return &tempTmux{socket}, nil
}

func (t *tempTmux) close() error {
	cmd := exec.Command("tmux", "-S", t.socket, "kill-session")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kill test tmux: %w: %s", err, string(out))
	}

	return nil
}

func TestListSessions(t *testing.T) {
	tm, err := newTempTmux()
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

	expected := []Session{{Id: "$0", Name: "0"}}

	if !slices.Equal(sessions, expected) {
		t.Fatalf("ListSessions() = %+v, want %+v", sessions, expected)
	}
}
