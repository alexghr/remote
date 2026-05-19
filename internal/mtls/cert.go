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

const (
	certFile = "cert.pem"
	keyFile  = "key.pem"
)

func LoadCertificate(dir string, usage x509.ExtKeyUsage) (tls.Certificate, error) {
	cert, err := loadCert(dir, usage)

	if err == nil {
		return cert, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return tls.Certificate{}, err
	}

	cert, err = newCert(usage)
	if err != nil {
		return tls.Certificate{}, err
	}

	if err := storeCert(dir, cert); err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func X509CertificateFingerprint(cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", fmt.Errorf("certificate is nil")
	}

	hash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return "sha256:" + base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

func storeCert(dir string, cert tls.Certificate) error {
	if cert.PrivateKey == nil {
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
		Bytes: cert.Leaf.Raw,
	})

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return fmt.Errorf("marshal ed25519 private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})

	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	return nil
}

func loadCert(dir string, usage x509.ExtKeyUsage) (tls.Certificate, error) {
	certFile := filepath.Join(dir, certFile)
	keyFile := filepath.Join(dir, keyFile)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load x509 keypair: %w", err)
	}

	if cert.Leaf == nil {
		return tls.Certificate{}, fmt.Errorf("certificate leaf missing")
	}

	if !slices.Contains(cert.Leaf.ExtKeyUsage, usage) {
		return tls.Certificate{}, fmt.Errorf("invalid key usage %v, want %v", cert.Leaf.ExtKeyUsage, usage)
	}

	return cert, nil
}

func newCert(usage x509.ExtKeyUsage) (tls.Certificate, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	tmpl := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "x"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{usage},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create x509 certificate: %w", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse x509 certificate: %w", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
		Leaf:        leaf,
	}

	return cert, nil
}

