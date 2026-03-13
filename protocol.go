package main

import "encoding/json"

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP protocol types

type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    ClientCaps     `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    ServerCaps     `json:"capabilities"`
	ServerInfo      Implementation `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientCaps struct {
	Roots    *RootsCap    `json:"roots,omitempty"`
	Sampling *SamplingCap `json:"sampling,omitempty"`
}

type RootsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCap struct{}

type ServerCaps struct {
	Tools     *ToolsCap     `json:"tools,omitempty"`
	Resources *ResourcesCap `json:"resources,omitempty"`
	Prompts   *PromptsCap   `json:"prompts,omitempty"`
}

type ToolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCap struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Tools
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content  []ContentBlock `json:"content"`
	IsError  bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func textContent(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

func errorResult(msg string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{textContent(msg)},
		IsError: true,
	}
}

func successResult(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{textContent(text)},
	}
}

// JSON-RPC error codes
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

func rpcError(id json.RawMessage, code int, msg string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: msg},
	}
}

func rpcResult(id json.RawMessage, result interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
