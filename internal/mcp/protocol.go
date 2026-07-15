// Package mcp implements a narrow Model Context Protocol server exposing a
// reviewed subset of symrelate's contact-management contract to MCP
// clients over stdio JSON-RPC 2.0. See docs/MCP.md for the tool catalogue
// and docs/CLI_CONTRACT.md for the shared JSON versioning rules.
//
// Transport hygiene mirrors internal/cli: stdout carries protocol frames
// only (one JSON-RPC message per line), diagnostics go to stderr, and
// every error message is passed through security.Redact before it can
// reach a client, so a contact-point value can never leak through an
// error path even if a call site forgot to keep it out.
package mcp

import "encoding/json"

// ProtocolVersion is the MCP protocol revision this server implements.
// initialize always responds with this value regardless of what the
// client requested — see docs/MCP.md for the compatibility note.
const ProtocolVersion = "2024-11-05"

// rpcRequest is a JSON-RPC 2.0 request or notification (Notifications
// omit ID and receive no response).
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response. Exactly one of Result/Error is
// set.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes (https://www.jsonrpc.org/specification).
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Application-defined error codes, reserved range per the JSON-RPC spec
// (-32000 to -32099), mapped from errs.Kind — see errorToRPC.
const (
	codeNotFound    = -32001
	codeConflict    = -32002
	codePermission  = -32003
	codeUnavailable = -32004
)

func newResult(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func newError(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}
