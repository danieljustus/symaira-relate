package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/danieljustus/symaira-relate/internal/app"
)

func testApp(t *testing.T) *app.App {
	t.Helper()
	a, err := app.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("app.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

// call sends one JSON-RPC request line through Server.Run and returns the
// decoded response. Run is driven over an in-memory pipe so this exercises
// the exact stdio framing symrelate mcp uses in production.
func call(t *testing.T, s *Server, id int, method string, params any) map[string]any {
	t.Helper()

	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	reqLine, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var stdout, stderr bytes.Buffer
	stdin := bytes.NewReader(append(reqLine, '\n'))
	if err := s.Run(context.Background(), stdin, &stdout, &stderr); err != nil {
		t.Fatalf("Run() error = %v (stderr=%s)", err, stderr.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, stdout.String())
	}
	return resp
}

func TestServer_StdoutIsProtocolClean(t *testing.T) {
	s := New(testApp(t))
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n")
	if err := s.Run(context.Background(), stdin, &stdout, &stderr); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	scanner := bufio.NewScanner(&stdout)
	lines := 0
	for scanner.Scan() {
		lines++
		var v map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Errorf("stdout line %d is not valid JSON: %v (%s)", lines, err, scanner.Text())
		}
	}
	if lines != 1 {
		t.Errorf("expected exactly 1 stdout line, got %d", lines)
	}
	if stderr.Len() == 0 {
		t.Error("expected a diagnostic line on stderr")
	}
}

func TestServer_Initialize(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "initialize", nil)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in %v", resp)
	}
	if result["protocolVersion"] != ProtocolVersion {
		t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], ProtocolVersion)
	}
	serverInfo, _ := result["serverInfo"].(map[string]any)
	if serverInfo["name"] != "symrelate" {
		t.Errorf("serverInfo.name = %v, want symrelate", serverInfo["name"])
	}
}

func TestServer_ToolsList_SnakeCaseNames(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "tools/list", nil)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in %v", resp)
	}
	tools, _ := result["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}
	want := map[string]bool{
		"contact_search": false, "contact_get": false, "contact_create": false, "contact_update": false,
		"organization_search": false, "organization_get": false, "followup_list": false, "timeline_get": false,
	}
	for _, raw := range tools {
		tl := raw.(map[string]any)
		name, _ := tl["name"].(string)
		if name != strings.ToLower(name) || strings.Contains(name, "-") {
			t.Errorf("tool name %q is not snake_case", name)
		}
		if _, ok := want[name]; ok {
			want[name] = true
		}
		if _, ok := tl["inputSchema"].(map[string]any); !ok {
			t.Errorf("tool %q missing inputSchema", name)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected tool %q not found in tools/list", name)
		}
	}
}

func TestServer_ToolsCall_ContactCreateAndGet(t *testing.T) {
	s := New(testApp(t))

	createResp := call(t, s, 1, "tools/call", map[string]any{
		"name":      "contact_create",
		"arguments": map[string]any{"display_name": "Ada Lovelace", "email": "ada@example.com"},
	})
	createResult, ok := createResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in %v", createResp)
	}
	if createResult["isError"] != false {
		t.Fatalf("contact_create reported an error: %v", createResult)
	}
	structured, ok := createResult["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structuredContent in %v", createResult)
	}
	id, _ := structured["ID"].(string)
	if id == "" {
		t.Fatalf("created contact missing ID: %v", structured)
	}

	getResp := call(t, s, 2, "tools/call", map[string]any{
		"name":      "contact_get",
		"arguments": map[string]any{"id": id},
	})
	getResult := getResp["result"].(map[string]any)
	got := getResult["structuredContent"].(map[string]any)
	if got["DisplayName"] != "Ada Lovelace" {
		t.Errorf("DisplayName = %v, want Ada Lovelace", got["DisplayName"])
	}
}

// An unknown tool name is a protocol-level mistake by the caller (it can
// never succeed by retrying with different arguments), so it is reported
// as a JSON-RPC error — unlike a valid tool that fails during execution
// (not-found, validation, ...), which is reported in-band via isError so
// a client can distinguish "the tool ran and failed" from "the call
// itself was invalid".
func TestServer_ToolsCall_UnknownTool_IsRPCError(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "tools/call", map[string]any{"name": "does_not_exist", "arguments": map[string]any{}})
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected an RPC error for an unknown tool, got %v", resp)
	}
	if code, _ := errObj["code"].(float64); int(code) != codeInvalidParams {
		t.Errorf("code = %v, want %d", errObj["code"], codeInvalidParams)
	}
}

func TestServer_ToolsCall_ContactGet_NotFound_IsInBandError(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "tools/call", map[string]any{
		"name":      "contact_get",
		"arguments": map[string]any{"id": "does-not-exist"},
	})
	result := resp["result"].(map[string]any)
	if result["isError"] != true {
		t.Errorf("expected isError=true for a not-found contact, got %v", result)
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "bogus/method", nil)
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected an error response, got %v", resp)
	}
	if code, _ := errObj["code"].(float64); int(code) != codeMethodNotFound {
		t.Errorf("code = %v, want %d", errObj["code"], codeMethodNotFound)
	}
}

func TestServer_MalformedJSON_ReturnsParseError_NotCrash(t *testing.T) {
	s := New(testApp(t))
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("{not json\n")
	if err := s.Run(context.Background(), stdin, &stdout, &stderr); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, stdout.String())
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected a parse error response, got %v", resp)
	}
	if code, _ := errObj["code"].(float64); int(code) != codeParseError {
		t.Errorf("code = %v, want %d", errObj["code"], codeParseError)
	}
}

func TestServer_Notification_GetsNoResponse(t *testing.T) {
	s := New(testApp(t))
	var stdout, stderr bytes.Buffer
	// No "id" field: this is a notification per JSON-RPC 2.0 and must not
	// produce a response line.
	stdin := strings.NewReader(`{"jsonrpc":"2.0","method":"ping"}` + "\n")
	if err := s.Run(context.Background(), stdin, &stdout, &stderr); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no stdout output for a notification, got %q", stdout.String())
	}
}

func TestServer_UnknownFieldInArguments_IsRejected(t *testing.T) {
	resp := call(t, New(testApp(t)), 1, "tools/call", map[string]any{
		"name":      "contact_create",
		"arguments": map[string]any{"display_name": "X", "not_a_real_field": "y"},
	})
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in %v", resp)
	}
	if result["isError"] != true {
		t.Errorf("expected isError=true for an unknown argument field, got %v", result)
	}
}
