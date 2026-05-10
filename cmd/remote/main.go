package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/alexghr/remote/internal/api"
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

	addr := "127.0.0.1:8080"
	server := &http.Server{
		Addr:    addr,
		Handler: api.New(c),
	}

	fmt.Fprintf(os.Stderr, "listening on http://%s\n", addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
