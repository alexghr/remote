package tmux

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"slices"
	"testing"
)

type tempTmux struct {
	socket string
}

func newTempTmux() (*tempTmux, error) {
	id := rand.Int()
	socket := fmt.Sprintf("tmux_test_%d", id)
	cmd := exec.Command("tmux", "-L", socket, "new")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("start test tmux: %w", err)
	}

	return &tempTmux{socket}, nil
}

func (t *tempTmux) close() error {
	cmd := exec.Command("tmux", "-L", t.socket, "kill-session")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kill test tmux: %w", err)
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

	expected := []Session{Session{Id: "%0", Name: "0"}}

	if !slices.Equal(sessions, expected) {
		t.Fatalf("ListSessions() = %+v, want %+v", sessions, expected)
	}
}
