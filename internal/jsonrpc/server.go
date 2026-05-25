package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
)

type RPCHandler func(ctx context.Context, params json.RawMessage) (any, error)

type Server struct {
	t        JSONTransport
	mu       sync.Mutex
	handlers map[string]RPCHandler
	logger   *slog.Logger
}

func NewServer(t JSONTransport, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := Server{
		t:        t,
		handlers: make(map[string]RPCHandler),
		logger:   logger,
	}

	t.RegisterHandler(s.handleReq)

	return &s
}

func (s *Server) Start(ctx context.Context) error {
	err := s.t.Start(ctx)
	if ctx.Err() != nil {
		return nil
	}

	return err
}

func (s *Server) Close() error {
	return s.t.Close()
}

func (s *Server) handleReq(ctx context.Context, msg json.RawMessage) json.RawMessage {
	var req rpcRequest

	if err := json.Unmarshal(msg, &req); err != nil {
		s.logger.Warn("received undecodable request", "err", err)
		return nil // can't send back a response if we don't even have an request ID
	}

	noReply := len(req.ID) == 0

	if req.JSONRPC != version || req.Method == "" {
		if noReply {
			return nil
		}
		return wrapErr(req.ID, ErrInvalidRequest)
	}

	handler := s.getHandler(req.Method)
	if handler == nil {
		if noReply {
			return nil
		}
		return wrapErr(req.ID, ErrMethodNotFound)
	}

	res, err := handler(ctx, req.Params)
	if noReply {
		return nil
	}

	if err != nil {
		return wrapErr(req.ID, err)
	}

	resJson, err := json.Marshal(res)
	if err != nil {
		return wrapErr(req.ID, err)
	}

	json, err := json.Marshal(rpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resJson,
	})

	if err != nil {
		return wrapErr(req.ID, err)
	}

	return json
}

func (s *Server) Handle(method string, handler RPCHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *Server) getHandler(method string) RPCHandler {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.handlers[method]
}

func wrapErr(id json.RawMessage, err error) json.RawMessage {
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		rpcErr = &Error{Code: CodeInternalError, Message: err.Error()}
	}

	res, mErr := json.Marshal(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcErr,
	})

	if mErr != nil {
		res, _ = json.Marshal(rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &Error{Code: CodeInternalError, Message: mErr.Error()},
		})
	}

	return res
}
