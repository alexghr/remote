package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"slices"
	"time"
)

type Conn struct {
	net.Conn
	Fingerprint string
}

type ServerConfig struct {
	HandshakeTimeout time.Duration
}

type Server struct {
	addr     string
	listener net.Listener
	config   ServerConfig
}

func NewServer(addr string, cert tls.Certificate) (*Server, error) {
	tlsConfig, err := serverTLSConfig(cert)
	if err != nil {
		return nil, fmt.Errorf("server tls config: %w", err)
	}

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("listen server: %w", err)
	}

	return &Server{
		addr:     addr,
		listener: listener,
		config: ServerConfig{
			HandshakeTimeout: 1 * time.Second,
		},
	}, nil
}

func (s *Server) Close() error {
	if s.listener == nil {
		return fmt.Errorf("listener is nil")
	}

	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("close server: %w", err)
	}

	return nil
}

func (s *Server) Accept(ctx context.Context) (*Conn, error) {
	conn, err := s.listener.Accept()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("accept conn: %w", err)
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("non-TLS conn: %T", conn)
	}

	handshakeCtx, cancel := context.WithTimeout(ctx, s.config.HandshakeTimeout)
	err = tlsConn.HandshakeContext(handshakeCtx)
	cancel()
	if err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("tls handshake: %w", err)
	}

	state := tlsConn.ConnectionState()
	fingerprint, err := X509CertificateFingerprint(state.PeerCertificates[0])
	if err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("client certificate fingerprint: %w", err)
	}

	return &Conn{
		Conn:        tlsConn,
		Fingerprint: fingerprint,
	}, nil
}

func serverTLSConfig(cert tls.Certificate) (*tls.Config, error) {
	if cert.Leaf == nil {
		return nil, fmt.Errorf("certificate leaf missing")
	}

	if !slices.Contains(cert.Leaf.ExtKeyUsage, x509.ExtKeyUsageServerAuth) {
		return nil, fmt.Errorf("invalid server cert key usage %v", cert.Leaf.ExtKeyUsage)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		VerifyConnection: func(cs tls.ConnectionState) error {
			cert := cs.PeerCertificates[0]
			if !slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
				return fmt.Errorf("invalid client cert key usage %v", cert.ExtKeyUsage)
			}
			return nil
		},
	}, nil
}
