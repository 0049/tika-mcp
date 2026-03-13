package main

import "encoding/json"

// rawSchema converts a Go map to json.RawMessage for tool InputSchema.
func rawSchema(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// allTools returns the complete list of MCP tools exposed by tika-mcp.
func allTools() []Tool {
	return []Tool{
		{
			Name: "tika_extract_text",
			Description: "Extract plain text content from a document. " +
				"Accepts base64-encoded file content. " +
				"Supports PDF, DOCX, XLSX, PPTX, HTML, XML, TXT, ODT, RTF, images (with OCR), and 1000+ other formats. " +
				"Returns the extracted plain text.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content to extract text from",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional original filename (helps Tika select the right parser, e.g. 'report.pdf')",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type of the document (e.g. 'application/pdf', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document')",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_extract_html",
			Description: "Extract content from a document as annotated HTML. " +
				"The HTML preserves structural information like headings, paragraphs, tables, and lists. " +
				"Accepts base64-encoded file content. Returns HTML string.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional original filename",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type hint",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_extract_xml",
			Description: "Extract content from a document as XHTML XML. " +
				"Returns well-formed XHTML with structural markup. " +
				"Accepts base64-encoded file content.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional original filename",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type hint",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_get_metadata",
			Description: "Extract all metadata fields from a document as JSON. " +
				"Returns author, creation date, modification date, title, language, page count, " +
				"dimensions, and dozens of other document properties depending on the format. " +
				"Accepts base64-encoded file content.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional original filename",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type hint",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_get_metadata_field",
			Description: "Extract a specific metadata field from a document. " +
				"Useful when you need a single property like 'Author', 'title', 'Creation-Date', " +
				"'Content-Type', 'xmpTPg:NPages' (page count), 'dc:creator', etc. " +
				"Accepts base64-encoded file content.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content",
					},
					"field": map[string]interface{}{
						"type":        "string",
						"description": "Metadata field name to retrieve, e.g. 'Author', 'title', 'Creation-Date', 'Content-Type'",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional original filename",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type hint",
					},
				},
				"required": []string{"content_base64", "field"},
			}),
		},
		{
			Name: "tika_detect_content_type",
			Description: "Detect the MIME content type of a document. " +
				"Returns the detected MIME type string (e.g. 'application/pdf', 'image/jpeg'). " +
				"More reliable than file extension guessing. " +
				"Accepts base64-encoded file content.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document content",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional filename hint (e.g. 'document.pdf')",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_detect_language",
			Description: "Detect the natural language of text or document content. " +
				"Returns the ISO 639-1 language code (e.g. 'en', 'de', 'fr', 'es', 'ja'). " +
				"Accepts base64-encoded document content or plain text bytes.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content_base64": map[string]interface{}{
						"type":        "string",
						"description": "Base64-encoded document or text content",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional filename hint",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Optional MIME type hint (use 'text/plain' for raw text)",
					},
				},
				"required": []string{"content_base64"},
			}),
		},
		{
			Name: "tika_extract_from_url",
			Description: "Fetch a remote document from a URL and extract its plain text content using Tika. " +
				"The URL must be publicly accessible. " +
				"Supports all document formats Tika can parse.",
			InputSchema: rawSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Publicly accessible URL of the document to fetch and extract (e.g. https://example.com/doc.pdf)",
					},
				},
				"required": []string{"url"},
			}),
		},
		{
			Name:        "tika_server_version",
			Description: "Get the version of the running Apache Tika server. Returns the version string.",
			InputSchema: rawSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "tika_list_mime_types",
			Description: "List all MIME types supported by the Tika server with their parser and alias information. Returns a JSON object mapping MIME type strings to their metadata.",
			InputSchema: rawSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "tika_list_parsers",
			Description: "List all document parsers available in the Tika server with their supported MIME types. Useful to check if a specific format is supported before processing.",
			InputSchema: rawSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
		{
			Name:        "tika_health_check",
			Description: "Check if the Apache Tika server is running and reachable. Returns server status and URL.",
			InputSchema: rawSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}),
		},
	}
}
