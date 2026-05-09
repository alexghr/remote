package codex

import "github.com/alexghr/remote/internal/tmux"

type Codex struct {
	tmux *tmux.Tmux
}

func New(tmux *tmux.Tmux) (*Codex, error) {
	return &Codex{
		tmux: tmux,
	}, nil
}
