package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/danieljustus/symaira-relate/internal/app"
	"github.com/danieljustus/symaira-relate/internal/domain/security"
	"github.com/danieljustus/symaira-relate/internal/errs"
	"github.com/danieljustus/symaira-relate/internal/version"
)

// maxLineBytes bounds a single incoming JSON-RPC message. MCP tool calls
// are small structured requests, never bulk payloads — a generous but
// finite bound keeps a malformed or hostile client from growing an
// unbounded buffer in this process.
const maxLineBytes = 4 << 20 // 4 MiB

// Server dispatches JSON-RPC 2.0 requests read from stdin to the
// registered tools, over the App service boundary shared with the CLI.
type Server struct {
	app   *app.App
	tools map[string]tool
}

// New wires a Server against an already-open App. The caller owns the
// App's lifecycle (Open/Close) exactly as CLI commands do.
func New(a *app.App) *Server {
	s := &Server{app: a, tools: map[string]tool{}}
	registerTools(s)
	return s
}

// Run reads newline-delimited JSON-RPC 2.0 messages from r until EOF or
// ctx is cancelled, dispatches each to the matching method, and writes
// one JSON-RPC response per line to w. Diagnostics go to stderrW only —
// w carries protocol frames exclusively, so it stays safe to use as a
// transport even if a caller pipes stdout somewhere unexpected.
func (s *Server) Run(ctx context.Context, r io.Reader, w io.Writer, stderrW io.Writer) error {
	fmt.Fprintln(stderrW, "symrelate mcp: listening on stdio")

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		resp, isNotification := s.handleLine(ctx, line)
		if isNotification {
			continue
		}
		if err := writeResponse(w, resp); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderrW, "symrelate mcp: read error: %v\n", err)
		return err
	}
	return nil
}

func writeResponse(w io.Writer, resp rpcResponse) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

// handleLine parses and dispatches one JSON-RPC message. isNotification
// is true when the request carried no id and therefore must never
// receive a response (JSON-RPC 2.0 notification semantics).
func (s *Server) handleLine(ctx context.Context, line []byte) (resp rpcResponse, isNotification bool) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return newError(nil, codeParseError, "invalid JSON"), false
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return newError(req.ID, codeInvalidRequest, "not a valid JSON-RPC 2.0 request"), len(req.ID) == 0
	}
	if len(req.ID) == 0 {
		// Notification: process for side effects (none of our methods
		// currently have any) but never write a response.
		return rpcResponse{}, true
	}

	result, err := s.dispatch(ctx, req.Method, req.Params)
	if err != nil {
		return errorToResponse(req.ID, err), false
	}
	return newResult(req.ID, result), false
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return s.handleToolsList(), nil
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	default:
		return nil, &rpcMethodError{method: method}
	}
}

type rpcMethodError struct{ method string }

func (e *rpcMethodError) Error() string { return "method not found: " + e.method }

func (s *Server) handleInitialize() any {
	return map[string]any{
		"protocolVersion": ProtocolVersion,
		"serverInfo": map[string]any{
			"name":    version.Tool,
			"version": version.Version,
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	}
}

// errorToResponse maps an internal error to a JSON-RPC error response.
// The message is redacted (see docs/PRIVACY.md) so a contact-point value
// that reached an error path — a duplicate email in a conflict error, for
// example — can never leak through the MCP transport.
func errorToResponse(id json.RawMessage, err error) rpcResponse {
	if methodErr, ok := err.(*rpcMethodError); ok {
		return newError(id, codeMethodNotFound, methodErr.Error())
	}

	code := codeInternalError
	switch errs.KindOf(err) {
	case errs.KindNotFound:
		code = codeNotFound
	case errs.KindConflict:
		code = codeConflict
	case errs.KindInvalid:
		code = codeInvalidParams
	case errs.KindPermission:
		code = codePermission
	case errs.KindUnavailable:
		code = codeUnavailable
	case errs.KindInternal, "":
		code = codeInternalError
	}
	return newError(id, code, security.Redact(err.Error()))
}
