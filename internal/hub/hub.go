package hub

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/alexghr/remote/internal/jsonrpc"
	"github.com/alexghr/remote/internal/mtls"
)

type HubOptions struct {
	DataDir           string
	Addr              string
	HeartbeatInterval time.Duration
	PingTimeout       time.Duration
	MaxClientErrors   uint32
}

func withDefaults(opts HubOptions) HubOptions {
	o := opts
	if o.HeartbeatInterval == 0 {
		o.HeartbeatInterval = 10 * time.Second
	}

	if o.PingTimeout == 0 {
		o.PingTimeout = 250 * time.Millisecond
	}

	if o.MaxClientErrors == 0 {
		o.MaxClientErrors = 3
	}

	return o
}

type client struct {
	conn *mtls.Conn
	rpc  *jsonrpc.Client
	errs uint32
	mu   sync.Mutex
}

type Hub struct {
	opts    HubOptions
	tls     *mtls.Server
	logger  *slog.Logger
	clients map[string]*client
	mu      sync.Mutex
}

func New(opts HubOptions, logger *slog.Logger) (*Hub, error) {
	opts = withDefaults(opts)

	if opts.DataDir == "" {
		return nil, fmt.Errorf("empty data dir")
	}

	cert, err := mtls.LoadCertificate(filepath.Join(opts.DataDir, "server"), x509.ExtKeyUsageServerAuth)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	listener, err := mtls.NewServer(opts.Addr, cert)
	if err != nil {
		return nil, fmt.Errorf("create tls server: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	hub := Hub{
		opts:    opts,
		tls:     listener,
		logger:  logger,
		clients: make(map[string]*client),
	}

	return &hub, nil
}

func (h *Hub) Fingerprint() string {
	return h.tls.Fingerprint()
}

func (h *Hub) Addr() string {
	return h.tls.Addr()
}

func (h *Hub) Start(ctx context.Context) {
	h.logger.Info("Starting TLS server", "addr", h.tls.Addr())
	defer h.logger.Info("Stopping TLS server")

	go func() {
		<-ctx.Done()
		_ = h.tls.Close()
	}()

	go h.heartbeat(ctx)

	for {
		conn, err := h.tls.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			h.logger.Warn("Error accepting TLS connection", "err", err)
			continue
		}

		h.logger.Info("Client connected", "client", conn.Fingerprint)
		c, err := h.acceptClient(conn)
		if err != nil {
			h.logger.Warn("Refused client", "client", conn.Fingerprint, "err", err)
			_ = conn.Close()
			continue
		}

		go h.serveClient(ctx, c)
	}
}

func (h *Hub) acceptClient(conn *mtls.Conn) (*client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[conn.Fingerprint] != nil {
		return nil, fmt.Errorf("duplicate client: %s", conn.Fingerprint)
	}

	c := &client{
		conn: conn,
		rpc:  jsonrpc.NewClient(jsonrpc.NewConnTransport(conn), h.logger.WithGroup(conn.Fingerprint)),
	}
	h.clients[conn.Fingerprint] = c
	return c, nil
}

func (h *Hub) serveClient(ctx context.Context, c *client) {
	err := c.rpc.Start(ctx)
	if err != nil && ctx.Err() == nil {
		h.logger.Warn("Client RPC stopped", "client", c.conn.Fingerprint, "err", err)
	}

	h.removeClient(c)
}

func (h *Hub) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(h.opts.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			clients := func() []*client {
				h.mu.Lock()
				defer h.mu.Unlock()
				return slices.Collect(maps.Values(h.clients))
			}()

			for _, c := range clients {
				go func() {
					cctx, stop := context.WithTimeout(ctx, h.opts.PingTimeout)
					defer stop()
					h.pingClient(cctx, c)
				}()
			}
		}
	}
}

func (h *Hub) pingClient(ctx context.Context, client *client) {
	ok := client.mu.TryLock()

	if !ok {
		h.logger.Debug("Client busy", "client", client.conn.Fingerprint)
		return
	}

	defer client.mu.Unlock()
	err := ping(ctx, client.rpc)
	if err != nil {
		client.errs += 1
		if client.errs >= h.opts.MaxClientErrors {
			h.logger.Warn("Too many errors", "client", client.conn.Fingerprint, "errs", client.errs, "maxErrs", h.opts.MaxClientErrors)
			h.removeClient(client)
		} else {
			h.logger.Warn("Client failed ping", "client", client.conn.Fingerprint, "errs", client.errs, "maxErrs", h.opts.MaxClientErrors)
		}
	} else {
		client.errs = 0
	}
}

func (h *Hub) removeClient(c *client) {
	func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.clients[c.conn.Fingerprint] == c {
			// Invariant: no duplicate connections per client
			delete(h.clients, c.conn.Fingerprint)
		}
	}()
	err := c.conn.Close()
	h.logger.Info("Closed client", "client", c.conn.Fingerprint, "closeErr", err)
}

func ping(ctx context.Context, rpc *jsonrpc.Client) error {
	res, err := rpc.Call(ctx, "ping", nil)
	if err != nil {
		return err
	}

	var pong string
	if err := json.Unmarshal(res, &pong); err != nil {
		return fmt.Errorf("decode ping response: %w", err)
	}
	if pong != "pong" {
		return fmt.Errorf("unexpected ping response: %q", pong)
	}

	return nil
}
