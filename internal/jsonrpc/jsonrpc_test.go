package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestCallSuccess(t *testing.T) {
	client, server, cleanup := newPair(t)
	defer cleanup()

	server.Handle("hello", func(ctx context.Context, params json.RawMessage) (any, error) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		return map[string]string{"message": "hello " + req.Name}, nil
	})

	var got struct {
		Message string `json:"message"`
	}

	res, err := client.Call(t.Context(), "hello", map[string]string{"name": "remote"})
	if err != nil {
		t.Fatalf("call: %s", err)
	}

	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatalf("res decode: %s", err)
	}

	if got.Message != "hello remote" {
		t.Fatalf("message = %q, want hello remote", got.Message)
	}
}

func TestCallError(t *testing.T) {
	client, server, cleanup := newPair(t)
	defer cleanup()

	server.Handle("hello", func(ctx context.Context, _ json.RawMessage) (any, error) {
		return nil, fmt.Errorf("an error")
	})

	_, err := client.Call(t.Context(), "hello", map[string]string{"name": "remote"})
	if err == nil {
		t.Fatalf("call nil err")
	}

	if rpcError, ok := errors.AsType[*Error](err); ok {
		if rpcError.Message != "an error" {
			t.Fatalf("wrong error: %v", rpcError.Message)
		}
	} else {
		t.Fatalf("expected json rpc error: %v", err)
	}
}

func newPair(t *testing.T) (*Client, *Server, func()) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	server := NewServer(NewConnTransport(serverConn), nil)
	client := NewClient(NewConnTransport(clientConn), nil)

	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Go(func() {
		errs <- server.Start(ctx)
	})

	wg.Go(func() {
		errs <- client.Start(ctx)
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	cleanup := func() {
		cancel()
		_ = client.Close()
		_ = server.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out stopping server")
		}

		close(errs)
		for err := range errs {
			if err != nil {
				t.Errorf("jsonrpc start: %v", err)
			}
		}
	}

	return client, server, cleanup
}
