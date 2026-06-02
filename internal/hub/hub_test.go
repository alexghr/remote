package hub

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexghr/remote/internal/jsonrpc"
	"github.com/alexghr/remote/internal/mtls"
)

func TestHubAceptsConn(t *testing.T) {
	tmp := t.TempDir()
	logger := slog.Default()
	heartbeatInterval := 100 * time.Millisecond
	hub, err := New(HubOptions{
		DataDir:           filepath.Join(tmp, "server"),
		Addr:              "127.0.0.1:0",
		HeartbeatInterval: heartbeatInterval,
	}, logger)

	if err != nil {
		t.Fatalf("unexptected err: %s", err)
	}

	go hub.Start(t.Context())

	cert, err := mtls.LoadCertificate(filepath.Join(tmp, "client"), x509.ExtKeyUsageClientAuth)
	if err != nil {
		t.Fatalf("unexptected err: %s", err)
	}

	client, err := mtls.NewClient(hub.Addr(), cert, hub.Fingerprint())
	if err != nil {
		t.Fatalf("unexptected err: %s", err)
	}

	conn, err := client.Dial(t.Context())
	if err != nil {
		t.Fatalf("unexptected err: %s", err)
	}

	server := jsonrpc.NewServer(jsonrpc.NewConnTransport(conn), logger)

	ch := make(chan struct{})
	server.Handle("ping", func(ctx context.Context, params json.RawMessage) (any, error) {
		defer close(ch)
		return "pong", nil
	})

	go server.Start(t.Context())

	select {
	case <-time.After(3 * heartbeatInterval):
		t.Fatalf("timeout")
	case <-ch:
	}
}
