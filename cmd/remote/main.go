package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/alexghr/remote/internal/agent"
	"github.com/alexghr/remote/internal/api"
	"github.com/alexghr/remote/internal/codex"
	"github.com/alexghr/remote/internal/hub"
	"github.com/alexghr/remote/internal/process"
	"github.com/alexghr/remote/internal/tmux"
	"github.com/alexghr/remote/internal/web"
)

const rpcAddr = "127.0.0.1:8443"

func main() {
	flags := parseFlags()

	switch {
	case flags.agent:
		runAgent(flags)
	case flags.hub:
		runHub(flags)
	default:
		runLegacy()
	}
}

type flags struct {
	hubPin string
	agent  bool
	hub    bool
}

func parseFlags() flags {
	if len(os.Args) < 2 {
		return flags{}
	}

	switch os.Args[1] {
	case "agent":
		fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
		hubPin := fs.String("hub-pin", "", "the certificate fingerprint of the hub")
		fs.StringVar(hubPin, "server-pin", "", "alias for --hub-pin")
		fs.Parse(os.Args[2:])
		return flags{agent: true, hubPin: *hubPin}
	case "hub":
		return flags{hub: true}
	default:
		return flags{}
	}
}

func runAgent(f flags) {
	a, err := agent.New(agent.AgentOptions{
		DataDir:        configDir(),
		HubAddr:        rpcAddr,
		HubFingerprint: f.hubPin,
	}, nil)
	if err != nil {
		die(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Fprintf(os.Stdout, "Agent certificate fingerprint: %s\n", a.Fingerprint())
	if err := a.Start(ctx); err != nil {
		die(err)
	}
}

func runHub(_ flags) {
	h, err := hub.New(hub.HubOptions{
		DataDir: configDir(),
		Addr:    rpcAddr,
	}, nil)
	if err != nil {
		die(err)
	}

	fmt.Fprintf(os.Stdout, "Hub certificate fingerprint: %s\n", h.Fingerprint())
	fmt.Fprintf(os.Stderr, "listening on %s\n", rpcAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	h.Start(ctx)
}

func configDir() string {
	dataDir := os.Getenv("HOME")
	if dataDir == "" {
		// TODO non-Unix paths?
		dataDir = "/var/lib/remote"
	} else {
		dataDir = filepath.Join(dataDir, ".remote")
	}

	return dataDir
}

func runLegacy() {
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

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
