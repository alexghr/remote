package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"slices"
)

type Client struct {
	addr   string
	config *tls.Config
}

func NewClient(addr string, cert tls.Certificate, pin string) (*Client, error) {
	config, err := clientTLSConfig(cert, pin)
	if err != nil {
		return nil, fmt.Errorf("client tls config: %w", err)
	}

	return &Client{
		addr:   addr,
		config: config,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	dialer := tls.Dialer{Config: c.config}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	defer conn.Close()
	return nil
}

func clientTLSConfig(cert tls.Certificate, hubCertPin string) (*tls.Config, error) {
	if !slices.Contains(cert.Leaf.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
		return nil, fmt.Errorf("invalid client cert key usage %v", cert.Leaf.ExtKeyUsage)
	}

	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // only safe because VerifyConnection pins the server cert
		VerifyConnection: func(cs tls.ConnectionState) error {
			cert := cs.PeerCertificates[0]

			got, err := X509CertificateFingerprint(cert)
			if err != nil {
				return err
			}
			if hubCertPin != got {
				return fmt.Errorf("unexpected hub certificate pin: %s", got)
			}

			if !slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
				return fmt.Errorf("invalid server cert key usage %v", cert.ExtKeyUsage)
			}

			return nil
		},
	}, nil
}
