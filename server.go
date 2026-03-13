package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// MCPServer handles MCP protocol requests.
type MCPServer struct {
	tika *TikaClient
}

func NewMCPServer(tika *TikaClient) *MCPServer {
	return &MCPServer{tika: tika}
}

// HandleRequest processes a single JSON-RPC request and returns a response.
func (s *MCPServer) HandleRequest(raw []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		resp := rpcError(nil, ErrParse, "Parse error: "+err.Error())
		return &resp
	}
	if req.JSONRPC != "2.0" {
		resp := rpcError(req.ID, ErrInvalidRequest, "Invalid JSON-RPC version, expected '2.0'")
		return &resp
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification - no response needed (return nil to skip)
		return nil
	case "notifications/initialized":
		return nil
	case "ping":
		resp := rpcResult(req.ID, map[string]interface{}{})
		return &resp
	case "tools/list":
		resp := rpcResult(req.ID, ListToolsResult{Tools: allTools()})
		return &resp
	case "tools/call":
		return s.handleToolCall(req)
	default:
		resp := rpcError(req.ID, ErrMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
		return &resp
	}
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: Implementation{
			Name:    "tika-mcp",
			Version: serverVersion,
		},
		Capabilities: ServerCaps{
			Tools: &ToolsCap{},
		},
		Instructions: "Apache Tika MCP server. Use tika_extract_text to extract plain text from documents, " +
			"tika_get_metadata for document properties, tika_detect_content_type for MIME detection, " +
			"and tika_detect_language for language identification. " +
			"All document inputs should be base64-encoded bytes.",
	}
	resp := rpcResult(req.ID, result)
	return &resp
}

func (s *MCPServer) handleToolCall(req JSONRPCRequest) *JSONRPCResponse {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		resp := rpcError(req.ID, ErrInvalidParams, "Invalid tool call params: "+err.Error())
		return &resp
	}

	result, err := s.dispatchTool(params.Name, params.Arguments)
	if err != nil {
		result = errorResult(fmt.Sprintf("Internal error dispatching tool %q: %v", params.Name, err))
	}

	resp := rpcResult(req.ID, result)
	return &resp
}

// dispatchTool routes a tool call to the appropriate handler.
func (s *MCPServer) dispatchTool(name string, args json.RawMessage) (*CallToolResult, error) {
	switch name {
	case "tika_extract_text":
		return s.toolExtractText(args)
	case "tika_extract_html":
		return s.toolExtractHTML(args)
	case "tika_extract_xml":
		return s.toolExtractXML(args)
	case "tika_get_metadata":
		return s.toolGetMetadata(args)
	case "tika_get_metadata_field":
		return s.toolGetMetadataField(args)
	case "tika_detect_content_type":
		return s.toolDetectContentType(args)
	case "tika_detect_language":
		return s.toolDetectLanguage(args)
	case "tika_extract_from_url":
		return s.toolExtractFromURL(args)
	case "tika_server_version":
		return s.toolServerVersion()
	case "tika_list_mime_types":
		return s.toolListMIMETypes()
	case "tika_list_parsers":
		return s.toolListParsers()
	case "tika_health_check":
		return s.toolHealthCheck()
	default:
		return errorResult(fmt.Sprintf("Unknown tool: %q. Use tools/list to see available tools.", name)), nil
	}
}

// --- Argument helpers ---

type docArgs struct {
	ContentBase64 string `json:"content_base64"`
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
}

func parseDocArgs(raw json.RawMessage) (*docArgs, []byte, error) {
	var a docArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if a.ContentBase64 == "" {
		return nil, nil, fmt.Errorf("content_base64 is required")
	}
	data, err := base64.StdEncoding.DecodeString(a.ContentBase64)
	if err != nil {
		// Try URL-safe base64
		data, err = base64.URLEncoding.DecodeString(a.ContentBase64)
		if err != nil {
			// Try raw (no padding)
			data, err = base64.RawStdEncoding.DecodeString(a.ContentBase64)
			if err != nil {
				return nil, nil, fmt.Errorf("content_base64 is not valid base64: %w", err)
			}
		}
	}
	return &a, data, nil
}

// --- Tool handlers ---

func (s *MCPServer) toolExtractText(args json.RawMessage) (*CallToolResult, error) {
	a, data, err := parseDocArgs(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	text, err := s.tika.ExtractText(data, a.Filename, a.ContentType)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika text extraction failed: %v", err)), nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return successResult("(no text content extracted - document may be empty, image-only, or password-protected)"), nil
	}
	return successResult(text), nil
}

func (s *MCPServer) toolExtractHTML(args json.RawMessage) (*CallToolResult, error) {
	a, data, err := parseDocArgs(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	html, err := s.tika.ExtractHTML(data, a.Filename, a.ContentType)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika HTML extraction failed: %v", err)), nil
	}
	return successResult(html), nil
}

func (s *MCPServer) toolExtractXML(args json.RawMessage) (*CallToolResult, error) {
	a, data, err := parseDocArgs(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	xml, err := s.tika.ExtractXML(data, a.Filename, a.ContentType)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika XML extraction failed: %v", err)), nil
	}
	return successResult(xml), nil
}

func (s *MCPServer) toolGetMetadata(args json.RawMessage) (*CallToolResult, error) {
	a, data, err := parseDocArgs(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	meta, err := s.tika.GetMetadata(data, a.Filename, a.ContentType)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika metadata extraction failed: %v", err)), nil
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return errorResult("Failed to format metadata as JSON"), nil
	}
	return successResult(string(b)), nil
}

func (s *MCPServer) toolGetMetadataField(args json.RawMessage) (*CallToolResult, error) {
	var raw struct {
		ContentBase64 string `json:"content_base64"`
		Field         string `json:"field"`
		Filename      string `json:"filename"`
		ContentType   string `json:"content_type"`
	}
	if err := json.Unmarshal(args, &raw); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	if raw.ContentBase64 == "" {
		return errorResult("content_base64 is required"), nil
	}
	if raw.Field == "" {
		return errorResult("field is required"), nil
	}
	data, err := base64.StdEncoding.DecodeString(raw.ContentBase64)
	if err != nil {
		return errorResult("content_base64 is not valid base64: " + err.Error()), nil
	}
	value, err := s.tika.GetMetadataField(data, raw.Filename, raw.ContentType, raw.Field)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika metadata field %q failed: %v", raw.Field, err)), nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return successResult(fmt.Sprintf("(field %q not found in document metadata)", raw.Field)), nil
	}
	return successResult(fmt.Sprintf("%s: %s", raw.Field, value)), nil
}

func (s *MCPServer) toolDetectContentType(args json.RawMessage) (*CallToolResult, error) {
	var raw struct {
		ContentBase64 string `json:"content_base64"`
		Filename      string `json:"filename"`
	}
	if err := json.Unmarshal(args, &raw); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	if raw.ContentBase64 == "" {
		return errorResult("content_base64 is required"), nil
	}
	data, err := base64.StdEncoding.DecodeString(raw.ContentBase64)
	if err != nil {
		return errorResult("content_base64 is not valid base64: " + err.Error()), nil
	}
	result, err := s.tika.DetectContentType(data, raw.Filename)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika content-type detection failed: %v", err)), nil
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return successResult(string(b)), nil
}

func (s *MCPServer) toolDetectLanguage(args json.RawMessage) (*CallToolResult, error) {
	a, data, err := parseDocArgs(args)
	if err != nil {
		return errorResult(err.Error()), nil
	}
	result, err := s.tika.DetectLanguage(data, a.Filename, a.ContentType)
	if err != nil {
		return errorResult(fmt.Sprintf("Tika language detection failed: %v", err)), nil
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return successResult(string(b)), nil
}

func (s *MCPServer) toolExtractFromURL(args json.RawMessage) (*CallToolResult, error) {
	var raw struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &raw); err != nil {
		return errorResult("Invalid arguments: " + err.Error()), nil
	}
	if raw.URL == "" {
		return errorResult("url is required"), nil
	}
	text, err := s.tika.ExtractTextFromURL(raw.URL)
	if err != nil {
		return errorResult(fmt.Sprintf("URL extraction failed: %v", err)), nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return successResult("(no text content extracted from URL)"), nil
	}
	return successResult(text), nil
}

func (s *MCPServer) toolServerVersion() (*CallToolResult, error) {
	v, err := s.tika.GetVersion()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to get Tika version: %v", err)), nil
	}
	return successResult("Apache Tika version: " + v), nil
}

func (s *MCPServer) toolListMIMETypes() (*CallToolResult, error) {
	mimes, err := s.tika.GetMIMETypes()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to get MIME types: %v", err)), nil
	}
	b, _ := json.MarshalIndent(mimes, "", "  ")
	return successResult(string(b)), nil
}

func (s *MCPServer) toolListParsers() (*CallToolResult, error) {
	parsers, err := s.tika.GetParsers()
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to get parsers: %v", err)), nil
	}
	b, _ := json.MarshalIndent(parsers, "", "  ")
	return successResult(string(b)), nil
}

func (s *MCPServer) toolHealthCheck() (*CallToolResult, error) {
	err := s.tika.Ping()
	if err != nil {
		return &CallToolResult{
			Content: []ContentBlock{textContent(fmt.Sprintf(
				"❌ Tika server is NOT reachable at %s\nError: %v\n\nMake sure Apache Tika server is running:\n  java -jar tika-server-standard-*.jar --port 9998",
				s.tika.baseURL, err,
			))},
			IsError: true,
		}, nil
	}

	v, _ := s.tika.GetVersion()
	vStr := ""
	if v != "" {
		vStr = "\nVersion: " + v
	}

	return successResult(fmt.Sprintf("✅ Tika server is running\nURL: %s%s", s.tika.baseURL, vStr)), nil
}
