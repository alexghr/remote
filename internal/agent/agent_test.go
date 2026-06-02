package agent

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

func TestAgentAnswersPing(t *testing.T) {
	tmp := t.TempDir()
	logger := slog.Default()

	cert, err := mtls.LoadCertificate(filepath.Join(tmp, "server"), x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	pin, err := mtls.X509CertificateFingerprint(cert.Leaf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	hub, err := mtls.NewServer("127.0.0.1:0", cert)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer hub.Close()

	agent, err := New(AgentOptions{
		DataDir:        filepath.Join(tmp, "agent"),
		HubAddr:        hub.Addr(),
		HubFingerprint: pin,
	}, logger)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if agent.Fingerprint() == "" {
		t.Fatal("agent fingerprint is empty")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	go agent.Start(ctx)

	conn, err := hub.Accept(ctx)
	if err != nil {
		t.Fatalf("accept agent: %s", err)
	}
	defer conn.Close()

	rpc := jsonrpc.NewClient(jsonrpc.NewConnTransport(conn), logger)
	go rpc.Start(ctx)

	res, err := rpc.Call(ctx, "ping", nil)
	if err != nil {
		t.Fatalf("ping: %s", err)
	}

	var pong string
	if err := json.Unmarshal(res, &pong); err != nil {
		t.Fatalf("decode ping response: %s", err)
	}

	if pong != "pong" {
		t.Fatalf("ping response = %q, want pong", pong)
	}

	cancel()
	_ = rpc.Close()
}
