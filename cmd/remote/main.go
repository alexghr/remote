package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alexghr/remote/internal/process"
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

	tmuxPid := strings.Split(os.Getenv("TMUX"), ",")[1]
	pid, err := strconv.ParseInt(tmuxPid, 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	l := process.NewLinux()
	s, err := l.TakeSnapshot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("%v\n\n%v", panes, s.DescendantsOf(int(pid)))
}
