package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexghr/remote/internal/codex"
	"github.com/alexghr/remote/internal/tmux"
)

type CodexService interface {
	ListPanes(ctx context.Context) ([]codex.CodexPane, error)
	Capture(ctx context.Context, pane tmux.PaneID) (string, error)
	Prompt(ctx context.Context, pane tmux.PaneID, prompt string) error
	SendKeys(ctx context.Context, pane tmux.PaneID, keys []string) error
}

type Handler struct {
	codex CodexService
}

func New(codex CodexService) http.Handler {
	h := &Handler{codex: codex}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/codex/panes", h.listPanes)
	mux.HandleFunc("GET /api/codex/panes/{paneID}/content", h.capture)
	mux.HandleFunc("POST /api/codex/panes/{paneID}/prompt", h.prompt)
	mux.HandleFunc("POST /api/codex/panes/{paneID}/keys", h.sendKeys)

	return mux
}

func (h *Handler) listPanes(w http.ResponseWriter, r *http.Request) {
	panes, err := h.codex.ListPanes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list codex panes")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"panes": panes})
}

func (h *Handler) capture(w http.ResponseWriter, r *http.Request) {
	pane := tmux.PaneID(r.PathValue("paneID"))

	content, err := h.codex.Capture(r.Context(), pane)
	if err != nil {
		writeCodexError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"paneId":  pane,
		"content": content,
	})
}

type promptRequest struct {
	Prompt string `json:"prompt"`
}

func (h *Handler) prompt(w http.ResponseWriter, r *http.Request) {
	pane := tmux.PaneID(r.PathValue("paneID"))

	var req promptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.codex.Prompt(r.Context(), pane, req.Prompt); err != nil {
		writeCodexError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"paneId": pane,
		"sent":   true,
	})
}

type keyRequest struct {
	Keys []string `json:"keys"`
}

func (h *Handler) sendKeys(w http.ResponseWriter, r *http.Request) {
	pane := tmux.PaneID(r.PathValue("paneID"))

	var req keyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.codex.SendKeys(r.Context(), pane, req.Keys); err != nil {
		writeCodexError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"paneId": pane,
		"sent":   true,
	})
}

func writeCodexError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, codex.ErrPaneNotFound):
		writeError(w, http.StatusNotFound, "pane not found")
	case errors.Is(err, codex.ErrBadPrompt):
		writeError(w, http.StatusBadRequest, "bad prompt")
	case errors.Is(err, codex.ErrBadKeys):
		writeError(w, http.StatusBadRequest, "bad keys")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
