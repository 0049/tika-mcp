package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// RunStdio runs the MCP server using stdio transport (newline-delimited JSON-RPC).
func (s *MCPServer) RunStdio() error {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer for large documents (up to 64MB)
	buf := make([]byte, 64*1024*1024)
	scanner.Buffer(buf, 64*1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.HandleRequest(line)
		if resp == nil {
			// Notification, no response
			continue
		}

		if err := encoder.Encode(resp); err != nil {
			log.Printf("Failed to encode response: %v", err)
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stdin scanner error: %w", err)
	}
	return nil
}

// RunHTTP runs the MCP server using HTTP transport (streamable HTTP / SSE).
// Each POST to /mcp carries a JSON-RPC request and returns a JSON-RPC response.
func (s *MCPServer) RunHTTP(addr string) error {
	mux := http.NewServeMux()

	// Streamable HTTP MCP endpoint
	mux.HandleFunc("/mcp", s.httpMCPHandler)
	mux.HandleFunc("/mcp/", s.httpMCPHandler)

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := s.tika.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"degraded","tika_error":%q}`, err.Error())
			return
		}
		fmt.Fprintf(w, `{"status":"ok","tika_url":%q}`, s.tika.baseURL)
	})

	// Root info
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"tika-mcp","version":%q,"endpoint":"/mcp","health":"/health"}`, serverVersion)
	})

	log.Printf("tika-mcp HTTP server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *MCPServer) httpMCPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// SSE endpoint for server-sent events (optional, for streaming clients)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprintf(w, "data: {\"jsonrpc\":\"2.0\",\"method\":\"server/ready\"}\n\n")
		return
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Limit request body to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	var raw json.RawMessage
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp := rpcError(nil, ErrParse, "Parse error: "+err.Error())
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Handle batch requests
	if len(raw) > 0 && raw[0] == '[' {
		var requests []json.RawMessage
		if err := json.Unmarshal(raw, &requests); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			resp := rpcError(nil, ErrParse, "Batch parse error: "+err.Error())
			json.NewEncoder(w).Encode(resp)
			return
		}
		var responses []interface{}
		for _, req := range requests {
			resp := s.HandleRequest(req)
			if resp != nil {
				responses = append(responses, resp)
			}
		}
		json.NewEncoder(w).Encode(responses)
		return
	}

	resp := s.HandleRequest(raw)
	if resp == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	json.NewEncoder(w).Encode(resp)
}
