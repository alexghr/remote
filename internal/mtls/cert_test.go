package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"testing"
)

func TestNewCert(t *testing.T) {
	dir := t.TempDir()
	c, err := LoadCertificate(dir, RoleServer)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	if c.cert.Leaf == nil {
		t.Fatalf("nil leaf")
	}

	pin := c.Fingerprint()
	if !strings.HasPrefix(pin, "sha256:") || len(pin[len("sha256:"):]) == 0 {
		t.Fatalf("invalid fingerprint: %s", pin)
	}
}

func TestLoadCert(t *testing.T) {
	dir := t.TempDir()
	c1, err := LoadCertificate(dir, RoleServer)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	c2, err := LoadCertificate(dir, RoleServer)
	if err != nil {
		t.Fatalf("LoadCertificate failed second time: %s", err)
	}

	c1Pin := c1.Fingerprint()
	c2Pin := c2.Fingerprint()
	if c1Pin != c2Pin {
		t.Fatalf("fingerprint mismatch: %s != %s", c1Pin, c2Pin)
	}
}

func TestLoadCertWrongRole(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadCertificate(dir, RoleServer)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	_, err = LoadCertificate(dir, RoleClient)
	if err == nil {
		t.Fatalf("LoadCertificate should have filed")
	}
}

func TestVerifyServerConnPin(t *testing.T) {
	hub, err := newCert(RoleServer)
	if err != nil {
		t.Fatalf("new hub cert: %s", err)
	}

	client, err := newCert(RoleClient)
	if err != nil {
		t.Fatalf("new client cert: %s", err)
	}

	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hub.cert.Leaf},
	}

	if err := client.verifyServerConn(hub.Fingerprint(), state); err != nil {
		t.Fatalf("verifyServerConn good pin: %s", err)
	}

	if err := client.verifyServerConn("sha256:not-the-pin", state); err == nil {
		t.Fatal("verifyServerConn bad pin succeeded")
	}
}

func TestVerifyClientConnAcceptsCertificate(t *testing.T) {
	server, err := newCert(RoleServer)
	if err != nil {
		t.Fatalf("new server cert: %s", err)
	}

	agent, err := newCert(RoleClient)
	if err != nil {
		t.Fatalf("new agent cert: %s", err)
	}

	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{agent.cert.Leaf},
	}

	if err := server.verifyClientConn(state); err != nil {
		t.Fatalf("verify client cert: %s", err)
	}
}
