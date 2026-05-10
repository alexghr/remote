package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexghr/remote/internal/tmux"
)

func main() {
	t, err := tmux.New("tmux", strings.Split(os.Getenv("TMUX"), ",")[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ss, err := t.ListSessions(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	panes, err := t.ListPanes(context.Background(), ss[0].ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", panes)
}
