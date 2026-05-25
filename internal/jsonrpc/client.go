package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

type wRes struct {
	res rpcResponse
	err error
}

type Client struct {
	t       JSONTransport
	id      uint32
	mu      sync.Mutex
	pending map[uint32]chan wRes
	logger  *slog.Logger
}

func NewClient(t JSONTransport, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	c := &Client{
		t:       t,
		pending: make(map[uint32]chan wRes),
		logger:  logger,
	}

	t.RegisterHandler(c.handleRes)

	return c
}

func (c *Client) Start(ctx context.Context) error {
	err := c.t.Start(ctx)
	if err != nil {
		c.fail(err)
	}

	if ctx.Err() != nil {
		return nil
	}

	return err
}

func (c *Client) Close() error {
	return c.t.Close()
}

func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("params encoding: %w", err)
	}

	id := c.nextId()
	encodedId, _ := json.Marshal(id)
	req := rpcRequest{
		ID:      encodedId,
		JSONRPC: "2.0",
		Method:  method,
		Params:  encodedParams,
	}

	encodedReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("req encoding: %w", err)
	}

	ch := c.trackResponse(id)
	defer c.untrackResponse(id)

	err = c.t.Send(ctx, encodedReq)
	if err != nil {
		return nil, fmt.Errorf("req sending: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}

		if r.res.Error != nil {
			return nil, r.res.Error
		}

		return r.res.Result, nil
	}
}

func (c *Client) handleRes(ctx context.Context, msg json.RawMessage) json.RawMessage {
	var res rpcResponse
	if err := json.Unmarshal(msg, &res); err != nil {
		c.logger.WarnContext(ctx, "failed to decode response", "err", err)
		return nil
	}

	var id uint32
	if err := json.Unmarshal(res.ID, &id); err != nil {
		c.logger.WarnContext(ctx, "failed to decode response ID", "err", err)
		return nil
	}

	ch := c.getTrackedResponse(id)
	if ch != nil {
		ch <- wRes{res, nil}
	}

	// we don't intent to write anything back to the channel
	return nil
}

func (c *Client) nextId() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.id
	c.id += 1
	return id
}

func (c *Client) trackResponse(id uint32) chan wRes {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan wRes, 1)
	c.pending[id] = ch

	return ch
}

func (c *Client) getTrackedResponse(id uint32) chan wRes {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pending[id]
}

func (c *Client) untrackResponse(id uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, id)
}

func (c *Client) fail(err error) {
	c.mu.Lock()
	pending := c.pending
	c.pending = make(map[uint32]chan wRes)
	c.mu.Unlock()

	for _, ch := range pending {
		ch <- wRes{rpcResponse{}, err}
	}

	clear(pending)
}
