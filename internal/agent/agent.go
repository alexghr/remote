package agent

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/alexghr/remote/internal/jsonrpc"
	"github.com/alexghr/remote/internal/mtls"
)

type AgentOptions struct {
	DataDir        string
	HubAddr        string
	HubFingerprint string
}

type Agent struct {
	opts        AgentOptions
	tls         *mtls.Client
	fingerprint string
	logger      *slog.Logger
}

func New(opts AgentOptions, logger *slog.Logger) (*Agent, error) {
	if opts.DataDir == "" {
		return nil, fmt.Errorf("empty data dir")
	}

	if opts.HubAddr == "" {
		return nil, fmt.Errorf("empty hub addr")
	}

	if opts.HubFingerprint == "" {
		return nil, fmt.Errorf("empty hub fingerprint")
	}

	cert, err := mtls.LoadCertificate(filepath.Join(opts.DataDir, "client"), x509.ExtKeyUsageClientAuth)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	fingerprint, err := mtls.X509CertificateFingerprint(cert.Leaf)
	if err != nil {
		return nil, fmt.Errorf("certificate fingerprint: %w", err)
	}

	client, err := mtls.NewClient(opts.HubAddr, cert, opts.HubFingerprint)
	if err != nil {
		return nil, fmt.Errorf("create tls client: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Agent{
		opts:        opts,
		tls:         client,
		fingerprint: fingerprint,
		logger:      logger,
	}, nil
}

func (a *Agent) Fingerprint() string {
	return a.fingerprint
}

func (a *Agent) Start(ctx context.Context) error {
	a.logger.Info("Connecting to hub", "addr", a.opts.HubAddr)
	conn, err := a.tls.Dial(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}

		return fmt.Errorf("dial hub: %w", err)
	}

	a.logger.Info("Connected to hub", "addr", a.opts.HubAddr)
	server := jsonrpc.NewServer(jsonrpc.NewConnTransport(conn), a.logger)
	a.register(server)

	if err := server.Start(ctx); err != nil {
		if ctx.Err() != nil {
			return nil
		}

		return fmt.Errorf("agent rpc: %w", err)
	}

	return nil
}

func (a *Agent) register(server *jsonrpc.Server) {
	server.Handle("ping", a.ping)
}

func (a *Agent) ping(ctx context.Context, params json.RawMessage) (any, error) {
	return "pong", nil
}
