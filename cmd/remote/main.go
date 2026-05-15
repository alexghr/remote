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
	"github.com/alexghr/remote/internal/web"
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

	addr := os.Getenv("REMOTE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", api.New(c))
	mux.Handle("/", web.New())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Fprintf(os.Stderr, "listening on http://%s\n", addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
