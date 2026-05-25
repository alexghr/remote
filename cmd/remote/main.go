package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexghr/remote/internal/api"
	"github.com/alexghr/remote/internal/codex"
	"github.com/alexghr/remote/internal/jsonrpc"
	"github.com/alexghr/remote/internal/mtls"
	"github.com/alexghr/remote/internal/process"
	"github.com/alexghr/remote/internal/tmux"
	"github.com/alexghr/remote/internal/web"
)

const rpcAddr = "127.0.0.1:8443"

func main() {
	flags := parseFlags()

	switch {
	case flags.client:
		runClient(flags)
	case flags.server:
		runServer(flags)
	default:
		runLegacy()
	}
}

type flags struct {
	serverPin string
	client    bool
	server    bool
}

func parseFlags() flags {
	if len(os.Args) < 2 {
		return flags{}
	}

	switch os.Args[1] {
	case "client":
		fs := flag.NewFlagSet("client", flag.ExitOnError)
		serverPin := fs.String("server-pin", "", "the certificate fingerprint of the server")
		fs.Parse(os.Args[2:])
		return flags{client: true, serverPin: *serverPin}
	case "server":
		return flags{server: true}
	default:
		return flags{}
	}
}

func runClient(f flags) {
	cert, err := mtls.LoadCertificate(filepath.Join(configDir(), "client"), x509.ExtKeyUsageClientAuth)
	if err != nil {
		die(err)
	}

	client, err := mtls.NewClient(rpcAddr, cert, f.serverPin)
	if err != nil {
		die(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	conn, err := client.Dial(ctx)
	if err != nil {
		die(err)
	}

	server := jsonrpc.NewServer(jsonrpc.NewConnTransport(conn), nil)
	server.Handle("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		fmt.Fprintln(os.Stdout, "ping")
		return "pong", nil
	})

	fmt.Fprintf(os.Stderr, "connected to %s\n", rpcAddr)
	if err := server.Start(ctx); err != nil {
		die(err)
	}
}

func runServer(_ flags) {
	cert, err := mtls.LoadCertificate(filepath.Join(configDir(), "server"), x509.ExtKeyUsageServerAuth)
	if err != nil {
		die(err)
	}

	pin, err := mtls.X509CertificateFingerprint(cert.Leaf)
	if err != nil {
		die(err)
	}

	tlsServer, err := mtls.NewServer(rpcAddr, cert)
	if err != nil {
		die(err)
	}
	defer tlsServer.Close()

	fmt.Fprintf(os.Stdout, "Server certificate fingerprint: %s\n", pin)
	fmt.Fprintf(os.Stderr, "listening on %s\n", rpcAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = tlsServer.Close()
	}()

	for {
		conn, err := tlsServer.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Fprintf(os.Stderr, "accept: %s\n", err)
			continue
		}

		go heartbeat(ctx, conn)
	}
}

func heartbeat(ctx context.Context, conn *mtls.Conn) {
	client := jsonrpc.NewClient(jsonrpc.NewConnTransport(conn), nil)

	go client.Start(ctx)
	defer client.Close()

	fmt.Fprintf(os.Stderr, "client connected: %s\n", conn.Fingerprint)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	if err := pingClient(ctx, client); err != nil {
		fmt.Fprintf(os.Stderr, "heartbeat failed for %s: %s\n", conn.Fingerprint, err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := pingClient(ctx, client); err != nil {
				fmt.Fprintf(os.Stderr, "heartbeat failed for %s: %s\n", conn.Fingerprint, err)
				return
			}
		}
	}
}

func pingClient(ctx context.Context, client *jsonrpc.Client) error {
	res, err := client.Call(ctx, "ping", nil)
	if err != nil {
		return err
	}

	var pong string
	if err := json.Unmarshal(res, &pong); err != nil {
		return fmt.Errorf("decode ping response: %w", err)
	}
	if pong != "pong" {
		return fmt.Errorf("unexpected ping response: %q", pong)
	}

	fmt.Fprintln(os.Stdout, pong)
	return nil
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
