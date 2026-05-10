package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexghr/remote/internal/codex"
	"github.com/alexghr/remote/internal/process"
	"github.com/alexghr/remote/internal/tmux"
)

func main() {
	l := process.NewLinux()

	t, err := tmux.New("tmux", strings.Split(os.Getenv("TMUX"), ",")[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c, err := codex.New(t, l)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p, err := c.ListPanes(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", p)
}
