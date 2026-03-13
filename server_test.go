package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockTika creates a test HTTP server that mimics Apache Tika responses.
func mockTika(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/tika", func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		switch {
		case strings.Contains(accept, "text/html"):
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><p>Hello World</p></body></html>")
		case strings.Contains(accept, "application/xml"):
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, "<?xml version=\"1.0\"?><html><body><p>Hello World</p></body></html>")
		default:
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "Hello World")
		}
	})

	mux.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(r.URL.Path, "/meta/", 2)
		if len(parts) == 2 && parts[1] != "" {
			// Single field
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "Test Author")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Author":"Test Author","Content-Type":"application/pdf","title":"Test Doc"}`)
	})

	mux.HandleFunc("/detect/stream", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "application/pdf")
	})

	mux.HandleFunc("/language/stream", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "en")
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Apache Tika 2.9.1")
	})

	mux.HandleFunc("/mime-types", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"application/pdf":{"alias":["application/x-pdf"]}}`)
	})

	mux.HandleFunc("/parsers/details", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"DefaultParser","children":[]}`)
	})

	return httptest.NewServer(mux)
}

func newTestServer(t *testing.T) (*MCPServer, *httptest.Server) {
	mock := mockTika(t)
	tika := NewTikaClient(mock.URL)
	return NewMCPServer(tika), mock
}

func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func callTool(t *testing.T, s *MCPServer, toolName string, args map[string]interface{}) *CallToolResult {
	argsJSON, _ := json.Marshal(args)
	paramsJSON, _ := json.Marshal(CallToolParams{Name: toolName, Arguments: argsJSON})
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  paramsJSON,
	}
	raw, _ := json.Marshal(req)
	resp := s.HandleRequest(raw)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
	resultJSON, _ := json.Marshal(resp.Result)
	var result CallToolResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	return &result
}

func TestInitialize(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	params, _ := json.Marshal(InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      Implementation{Name: "test-client", Version: "1.0"},
	})
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  params,
	}
	raw, _ := json.Marshal(req)
	resp := s.HandleRequest(raw)

	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestToolsList(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/list",
	}
	raw, _ := json.Marshal(req)
	resp := s.HandleRequest(raw)

	resultJSON, _ := json.Marshal(resp.Result)
	var result ListToolsResult
	json.Unmarshal(resultJSON, &result)

	if len(result.Tools) == 0 {
		t.Fatal("expected tools, got none")
	}
	t.Logf("Found %d tools", len(result.Tools))
}

func TestExtractText(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_extract_text", map[string]interface{}{
		"content_base64": b64("fake pdf content"),
		"filename":       "test.pdf",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].Text, "Hello World") {
		t.Fatalf("unexpected text: %v", result.Content)
	}
}

func TestExtractHTML(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_extract_html", map[string]interface{}{
		"content_base64": b64("fake content"),
	})

	if result.IsError {
		t.Fatalf("expected success: %v", result.Content)
	}
	if !strings.Contains(result.Content[0].Text, "<html>") {
		t.Fatalf("expected HTML, got: %v", result.Content[0].Text)
	}
}

func TestGetMetadata(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_get_metadata", map[string]interface{}{
		"content_base64": b64("fake content"),
	})

	if result.IsError {
		t.Fatalf("expected success: %v", result.Content)
	}
	if !strings.Contains(result.Content[0].Text, "Author") {
		t.Fatalf("expected metadata with Author, got: %v", result.Content[0].Text)
	}
}

func TestDetectContentType(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_detect_content_type", map[string]interface{}{
		"content_base64": b64("fake pdf"),
		"filename":       "doc.pdf",
	})

	if result.IsError {
		t.Fatalf("expected success: %v", result.Content)
	}
	if !strings.Contains(result.Content[0].Text, "application/pdf") {
		t.Fatalf("expected PDF content type, got: %v", result.Content[0].Text)
	}
}

func TestDetectLanguage(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_detect_language", map[string]interface{}{
		"content_base64": b64("Hello World this is a test"),
	})

	if result.IsError {
		t.Fatalf("expected success: %v", result.Content)
	}
	if !strings.Contains(result.Content[0].Text, "en") {
		t.Fatalf("expected English, got: %v", result.Content[0].Text)
	}
}

func TestHealthCheck(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_health_check", map[string]interface{}{})
	if result.IsError {
		t.Fatalf("health check failed: %v", result.Content)
	}
	if !strings.Contains(result.Content[0].Text, "running") {
		t.Fatalf("unexpected health response: %v", result.Content[0].Text)
	}
}

func TestUnknownTool(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_nonexistent_tool", map[string]interface{}{})
	if !result.IsError {
		t.Fatal("expected error for unknown tool")
	}
}

func TestMissingBase64(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	result := callTool(t, s, "tika_extract_text", map[string]interface{}{})
	if !result.IsError {
		t.Fatal("expected error for missing content_base64")
	}
}

func TestInvalidJSONRPC(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	resp := s.HandleRequest([]byte(`{not valid json`))
	if resp == nil || resp.Error == nil {
		t.Fatal("expected error response for invalid JSON")
	}
	if resp.Error.Code != ErrParse {
		t.Fatalf("expected parse error code, got %d", resp.Error.Code)
	}
}

func TestMethodNotFound(t *testing.T) {
	s, mock := newTestServer(t)
	defer mock.Close()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "nonexistent/method",
	}
	raw, _ := json.Marshal(req)
	resp := s.HandleRequest(raw)

	if resp.Error == nil || resp.Error.Code != ErrMethodNotFound {
		t.Fatalf("expected method-not-found error, got: %v", resp)
	}
}

var _ = fmt.Sprintf // ensure fmt is used
