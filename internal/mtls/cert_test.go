package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"testing"
)

func TestNewCert(t *testing.T) {
	dir := t.TempDir()
	c, err := LoadCertificate(dir, x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	if c.Leaf == nil {
		t.Fatalf("nil leaf")
	}

	pin, err := X509CertificateFingerprint(c.Leaf)
	if err != nil {
		t.Fatalf("fingerprint: %s", err)
	}
	if !strings.HasPrefix(pin, "sha256:") || len(pin[len("sha256:"):]) == 0 {
		t.Fatalf("invalid fingerprint: %s", pin)
	}
}

func TestLoadCert(t *testing.T) {
	dir := t.TempDir()
	c1, err := LoadCertificate(dir, x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	c2, err := LoadCertificate(dir, x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("LoadCertificate failed second time: %s", err)
	}

	c1Pin, err := X509CertificateFingerprint(c1.Leaf)
	if err != nil {
		t.Fatalf("c1 fingerprint: %s", err)
	}

	c2Pin, err := X509CertificateFingerprint(c2.Leaf)
	if err != nil {
		t.Fatalf("c2 fingerprint: %s", err)
	}

	if c1Pin != c2Pin {
		t.Fatalf("fingerprint mismatch: %s != %s", c1Pin, c2Pin)
	}
}

func TestLoadCertWrongRole(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadCertificate(dir, x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("LoadCertificate failed: %s", err)
	}

	_, err = LoadCertificate(dir, x509.ExtKeyUsageClientAuth)
	if err == nil {
		t.Fatalf("LoadCertificate should have filed")
	}
}

func TestVerifyServerConnPin(t *testing.T) {
	hub, err := newCert(x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("new hub cert: %s", err)
	}

	client, err := newCert(x509.ExtKeyUsageClientAuth)
	if err != nil {
		t.Fatalf("new client cert: %s", err)
	}

	pin, err := X509CertificateFingerprint(hub.Leaf)
	if err != nil {
		t.Fatalf("hub fingerprint: %s", err)
	}

	config, err := clientTLSConfig(client, pin)
	if err != nil {
		t.Fatalf("client tls config: %s", err)
	}

	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hub.Leaf},
	}

	if err := config.VerifyConnection(state); err != nil {
		t.Fatalf("verify server conn good pin: %s", err)
	}

	badConfig, err := clientTLSConfig(client, "sha256:not-the-pin")
	if err != nil {
		t.Fatalf("bad client tls config: %s", err)
	}

	if err := badConfig.VerifyConnection(state); err == nil {
		t.Fatal("verify server conn bad pin succeeded")
	}
}

func TestVerifyClientConnAcceptsCertificate(t *testing.T) {
	server, err := newCert(x509.ExtKeyUsageServerAuth)
	if err != nil {
		t.Fatalf("new server cert: %s", err)
	}

	agent, err := newCert(x509.ExtKeyUsageClientAuth)
	if err != nil {
		t.Fatalf("new agent cert: %s", err)
	}

	config, err := serverTLSConfig(server)
	if err != nil {
		t.Fatalf("server tls config: %s", err)
	}

	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{agent.Leaf},
	}

	if err := config.VerifyConnection(state); err != nil {
		t.Fatalf("verify client cert: %s", err)
	}
}
