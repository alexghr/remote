package mtls

import (
	"context"
	"crypto/x509"
	"testing"
	"time"
)

type acceptResult struct {
	conn *Conn
	err  error
}

func TestClientDialServerIntegration(t *testing.T) {
	serverCert, err := newCert(x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("new server cert: %s", err)
	}

	clientCert, err := newCert(x509.ExtKeyUsageClientAuth)
	if err != nil {
		t.Fatalf("new client cert: %s", err)
	}

	serverPin, err := X509CertificateFingerprint(serverCert.Leaf)
	if err != nil {
		t.Fatalf("server fingerprint: %s", err)
	}

	clientPin, err := X509CertificateFingerprint(clientCert.Leaf)
	if err != nil {
		t.Fatalf("client fingerprint: %s", err)
	}

	server, err := NewServer("127.0.0.1:0", serverCert)
	if err != nil {
		t.Fatalf("new server: %s", err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	accepted := make(chan acceptResult, 1)
	go func() {
		conn, err := server.Accept(ctx)
		accepted <- acceptResult{conn: conn, err: err}
	}()

	client, err := NewClient(server.listener.Addr().String(), clientCert, serverPin)
	if err != nil {
		t.Fatalf("new client: %s", err)
	}

	clientConn, err := client.Dial(ctx)
	if err != nil {
		t.Fatalf("client dial: %s", err)
	}
	defer clientConn.Close()

	result := waitAccept(t, ctx, accepted)
	if result.err != nil {
		t.Fatalf("accept: %s", result.err)
	}

	serverConn := result.conn
	defer serverConn.Close()

	if serverConn.Fingerprint != clientPin {
		t.Fatalf("server saw client fingerprint %q, want %q", serverConn.Fingerprint, clientPin)
	}
}

func TestClientDialRejectsWrongServerPin(t *testing.T) {
	serverCert, err := newCert(x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("new server cert: %s", err)
	}

	clientCert, err := newCert(x509.ExtKeyUsageClientAuth)
	if err != nil {
		t.Fatalf("new client cert: %s", err)
	}

	server, err := NewServer("127.0.0.1:0", serverCert)
	if err != nil {
		t.Fatalf("new server: %s", err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	accepted := make(chan acceptResult, 1)
	go func() {
		conn, err := server.Accept(ctx)
		accepted <- acceptResult{conn: conn, err: err}
	}()

	client, err := NewClient(server.listener.Addr().String(), clientCert, "sha256:not-the-pin")
	if err != nil {
		t.Fatalf("new client: %s", err)
	}

	conn, err := client.Dial(ctx)
	if err == nil {
		_ = conn.Close()
		t.Fatal("client dial succeeded with wrong server pin")
	}

	result := waitAccept(t, ctx, accepted)
	if result.err == nil {
		_ = result.conn.Close()
		t.Fatal("server accept succeeded after client rejected server pin")
	}
}

func waitAccept(t *testing.T, ctx context.Context, accepted <-chan acceptResult) acceptResult {
	t.Helper()

	select {
	case result := <-accepted:
		return result
	case <-ctx.Done():
		t.Fatalf("accept timed out: %s", ctx.Err())
		return acceptResult{}
	}
}
