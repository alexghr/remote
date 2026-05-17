package mtls

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type Certificate struct {
	cert tls.Certificate
	priv ed25519.PrivateKey
}

type Role int

const (
	RoleServer Role = iota
	RoleClient
)

const (
	certFile = "cert.pem"
	keyFile  = "key.pem"
)

func LoadCertificate(dir string, role Role) (*Certificate, error) {
	cert, err := loadCert(dir, role)

	if err == nil {
		return cert, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	cert, err = newCert(role)
	if err != nil {
		return nil, err
	}

	if err := cert.storeCert(dir); err != nil {
		return nil, err
	}

	return cert, nil
}

func (c *Certificate) Fingerprint() string {
	if c.cert.Leaf == nil {
		panic(fmt.Sprintf("certificate leaf is nil"))
	}

	return hashCert(c.cert.Leaf)
}

func (c *Certificate) ServerConfig() *tls.Config {
	return &tls.Config{
		MinVersion:       tls.VersionTLS13,
		Certificates:     []tls.Certificate{c.cert},
		ClientAuth:       tls.RequireAnyClientCert,
		VerifyConnection: c.verifyClientConn,
	}
}

func (c *Certificate) ClientConfig(hubCertPin string) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{c.cert},
		InsecureSkipVerify: true, // only safe because VerifyConnection pins the server cert
		VerifyConnection:   func(cs tls.ConnectionState) error { return c.verifyServerConn(hubCertPin, cs) },
	}
}

func (c *Certificate) verifyClientConn(cs tls.ConnectionState) error {
	return nil
}

func (c *Certificate) verifyServerConn(hubCertPin string, cs tls.ConnectionState) error {
	got := hashCert(cs.PeerCertificates[0])
	if hubCertPin != got {
		return fmt.Errorf("unexpected hub certificate pin: %s", got)
	}

	return nil
}

func (c *Certificate) storeCert(dir string) error {
	if c.priv == nil {
		return fmt.Errorf("missing private key")
	}

  // if the user has changed file perms, we respect those
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create parent folder: %w", err)
	}

	certFile := filepath.Join(dir, certFile)
	keyFile := filepath.Join(dir, keyFile)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.cert.Leaf.Raw,
	})

	err := os.WriteFile(certFile, certPEM, 0644)
	if err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(c.priv)
	if err != nil {
		return fmt.Errorf("marshal ed25519 private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})

	err = os.WriteFile(keyFile, keyPEM, 0600)
	if err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	return nil
}

func loadCert(dir string, role Role) (*Certificate, error) {
	certFile := filepath.Join(dir, certFile)
	keyFile := filepath.Join(dir, keyFile)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load x509 keypair: %w", err)
	}

	if cert.Leaf == nil {
		return nil, fmt.Errorf("loaded certificate has no leaf")
	}

	if !slices.Contains(cert.Leaf.ExtKeyUsage, roleToExtKeyUsage(role)) {
		return nil, fmt.Errorf("invalid key usage %v for role %v", cert.Leaf.ExtKeyUsage, role)
	}

	return &Certificate{
		cert: cert,
	}, nil
}

func roleToExtKeyUsage(role Role) x509.ExtKeyUsage {
	switch role {
	case RoleServer:
		return x509.ExtKeyUsageServerAuth
	case RoleClient:
		return x509.ExtKeyUsageClientAuth
	default:
		// reject unknown roles
		panic(fmt.Sprintf("unknown role: %v", role))
	}
}

func newCert(role Role) (*Certificate, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	extKeyUsage := []x509.ExtKeyUsage{roleToExtKeyUsage(role)}

	tmpl := &x509.Certificate{
		SerialNumber:          nil,
		Subject:               pkix.Name{CommonName: "x"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("create x509 certificate: %w", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse x509 certificate: %w", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
		Leaf:        leaf,
	}

	return &Certificate{
		cert: cert,
		priv: priv,
	}, nil
}

func hashCert(c *x509.Certificate) string {
	hash := sha256.Sum256(c.RawSubjectPublicKeyInfo)
	return "sha256:" + base64.RawURLEncoding.EncodeToString(hash[:])
}
