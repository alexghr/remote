package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexghr/remote/internal/codex"
	"github.com/alexghr/remote/internal/tmux"
)

func TestListPanes(t *testing.T) {
	service := &fakeCodex{
		panes: []codex.CodexPane{{PaneID: "%12", PID: 1234}},
	}
	server := httptest.NewServer(New(service))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/codex/panes")
	if err != nil {
		t.Fatalf("GET /api/codex/panes: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestCapture(t *testing.T) {
	service := &fakeCodex{content: "hello"}
	server := httptest.NewServer(New(service))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/codex/panes/%2512/content")
	if err != nil {
		t.Fatalf("GET content: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if service.capturedPane != "%12" {
		t.Fatalf("captured pane = %q, want %%12", service.capturedPane)
	}
}

func TestPrompt(t *testing.T) {
	service := &fakeCodex{}
	server := httptest.NewServer(New(service))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/codex/panes/%2512/prompt", "application/json", strings.NewReader(`{"prompt":"continue"}`))
	if err != nil {
		t.Fatalf("POST prompt: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if service.promptedPane != "%12" {
		t.Fatalf("prompted pane = %q, want %%12", service.promptedPane)
	}

	if service.prompt != "continue" {
		t.Fatalf("prompt = %q, want continue", service.prompt)
	}
}

func TestPromptInvalidJSON(t *testing.T) {
	server := httptest.NewServer(New(&fakeCodex{}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/codex/panes/%2512/prompt", "application/json", strings.NewReader(`{`))
	if err != nil {
		t.Fatalf("POST prompt: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCodexErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "pane not found", err: codex.ErrPaneNotFound, want: http.StatusNotFound},
		{name: "bad prompt", err: codex.ErrBadPrompt, want: http.StatusBadRequest},
		{name: "internal", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeCodex{promptErr: tt.err}
			server := httptest.NewServer(New(service))
			defer server.Close()

			resp, err := http.Post(server.URL+"/api/codex/panes/%2512/prompt", "application/json", strings.NewReader(`{"prompt":"continue"}`))
			if err != nil {
				t.Fatalf("POST prompt: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

type fakeCodex struct {
	panes []codex.CodexPane

	content      string
	capturedPane tmux.PaneID
	captureErr   error

	promptedPane tmux.PaneID
	prompt       string
	promptErr    error
}

func (f *fakeCodex) ListPanes(ctx context.Context) ([]codex.CodexPane, error) {
	return f.panes, nil
}

func (f *fakeCodex) Capture(ctx context.Context, pane tmux.PaneID) (string, error) {
	f.capturedPane = pane
	return f.content, f.captureErr
}

func (f *fakeCodex) Prompt(ctx context.Context, pane tmux.PaneID, prompt string) error {
	f.promptedPane = pane
	f.prompt = prompt
	return f.promptErr
}
