package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	tikaURL = flag.String("tika-url", getEnvOrDefault("TIKA_URL", "http://localhost:9998"), "Apache Tika server URL")
	transport = flag.String("transport", getEnvOrDefault("MCP_TRANSPORT", "stdio"), "Transport mode: stdio or http")
	httpAddr  = flag.String("addr", getEnvOrDefault("MCP_ADDR", ":8080"), "HTTP listen address (for http transport)")
	version   = flag.Bool("version", false, "Print version and exit")
)

const serverVersion = "1.0.0"

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("tika-mcp version %s\n", serverVersion)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr)

	tika := NewTikaClient(*tikaURL)
	server := NewMCPServer(tika)

	switch *transport {
	case "stdio":
		log.Printf("Starting tika-mcp %s (stdio transport, Tika URL: %s)", serverVersion, *tikaURL)
		if err := server.RunStdio(); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
	case "http":
		log.Printf("Starting tika-mcp %s (HTTP transport on %s, Tika URL: %s)", serverVersion, *httpAddr, *tikaURL)
		if err := server.RunHTTP(*httpAddr); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown transport: %s. Use 'stdio' or 'http'\n", *transport)
		os.Exit(1)
	}
}
