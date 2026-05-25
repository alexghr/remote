package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type JSONMessageHandler func(ctx context.Context, msg json.RawMessage) json.RawMessage

type JSONTransport interface {
	Start(ctx context.Context) error
	Close() error

	Send(ctx context.Context, msg json.RawMessage) error
	RegisterHandler(handler JSONMessageHandler)
}

type ConnTransport struct {
	conn    net.Conn
	writeMu sync.Mutex

	handler   JSONMessageHandler
	handlerMu sync.Mutex
}

func NewConnTransport(conn net.Conn) *ConnTransport {
	return &ConnTransport{conn: conn}
}

func (t *ConnTransport) RegisterHandler(handler JSONMessageHandler) {
	t.handlerMu.Lock()
	defer t.handlerMu.Unlock()
	t.handler = handler
}

func (t *ConnTransport) Send(ctx context.Context, msg json.RawMessage) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := t.conn.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("write deadline: %w", err)
		}
		defer t.conn.SetWriteDeadline(time.Time{})
	}

	if err := writeFrame(t.conn, msg); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (t *ConnTransport) Start(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			_ = t.conn.Close()
		case <-done:
		}
	}()

	defer close(done)
	defer t.conn.Close()

	for {
		payload, err := readFrame(t.conn)

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return err
		}

		go t.handle(ctx, json.RawMessage(payload))
	}
}

func (t *ConnTransport) Close() error {
	return t.conn.Close()
}

func (t *ConnTransport) handle(ctx context.Context, req json.RawMessage) {
	handler := t.getHandler()

	if handler == nil {
		return
	}

	res := handler(ctx, req)

	if res != nil {
		err := t.Send(ctx, res)
		if err != nil {
			t.Close()
		}
	}
}

func (t *ConnTransport) getHandler() JSONMessageHandler {
	t.handlerMu.Lock()
	defer t.handlerMu.Unlock()
	return t.handler
}
