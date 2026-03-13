package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// TikaClient wraps Apache Tika REST API calls.
type TikaClient struct {
	baseURL    string
	httpClient *http.Client
}

// TikaMetadata holds key-value metadata returned by Tika.
type TikaMetadata map[string]interface{}

// TikaLanguageResult holds the detected language.
type TikaLanguageResult struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence,omitempty"`
	RawText    string  `json:"raw,omitempty"`
}

// TikaDetectResult holds content-type detection results.
type TikaDetectResult struct {
	ContentType string `json:"content_type"`
	MimeType    string `json:"mime_type"`
}

func NewTikaClient(baseURL string) *TikaClient {
	return &TikaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Ping checks if Tika server is alive.
func (c *TikaClient) Ping() error {
	resp, err := c.httpClient.Get(c.baseURL + "/tika")
	if err != nil {
		return fmt.Errorf("cannot reach Tika at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Tika returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// ExtractText extracts plain text from document bytes.
// endpoint: PUT /tika  Accept: text/plain
func (c *TikaClient) ExtractText(data []byte, filename, contentType string) (string, error) {
	return c.putContent("/tika", data, filename, contentType, "text/plain")
}

// ExtractHTML extracts HTML from document bytes.
// endpoint: PUT /tika  Accept: text/html
func (c *TikaClient) ExtractHTML(data []byte, filename, contentType string) (string, error) {
	return c.putContent("/tika", data, filename, contentType, "text/html")
}

// ExtractXML extracts XHTML from document bytes.
// endpoint: PUT /tika  Accept: application/xml
func (c *TikaClient) ExtractXML(data []byte, filename, contentType string) (string, error) {
	return c.putContent("/tika", data, filename, contentType, "application/xml")
}

// GetMetadata returns all metadata fields as a map.
// endpoint: PUT /meta  Accept: application/json
func (c *TikaClient) GetMetadata(data []byte, filename, contentType string) (TikaMetadata, error) {
	body, err := c.putContentRaw("/meta", data, filename, contentType, "application/json")
	if err != nil {
		return nil, err
	}
	var meta TikaMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}
	return meta, nil
}

// GetMetadataField returns a single metadata field value.
// endpoint: PUT /meta/<field>
func (c *TikaClient) GetMetadataField(data []byte, filename, contentType, field string) (string, error) {
	return c.putContent("/meta/"+field, data, filename, contentType, "text/plain")
}

// DetectContentType detects MIME type of document bytes.
// endpoint: PUT /detect/stream
func (c *TikaClient) DetectContentType(data []byte, filename string) (*TikaDetectResult, error) {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/detect/stream", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if filename != "" {
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tika detect request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tika returned HTTP %d: %s", resp.StatusCode, body)
	}
	ct := strings.TrimSpace(string(body))
	parts := strings.SplitN(ct, ";", 2)
	return &TikaDetectResult{
		ContentType: ct,
		MimeType:    strings.TrimSpace(parts[0]),
	}, nil
}

// DetectLanguage detects the language of text or document bytes.
// endpoint: PUT /language/stream
func (c *TikaClient) DetectLanguage(data []byte, filename, contentType string) (*TikaLanguageResult, error) {
	raw, err := c.putContent("/language/stream", data, filename, contentType, "text/plain")
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	return &TikaLanguageResult{
		Language: raw,
		RawText:  raw,
	}, nil
}

// GetVersion returns the Tika server version string.
func (c *TikaClient) GetVersion() (string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/version")
	if err != nil {
		return "", fmt.Errorf("cannot reach Tika: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetMIMETypes returns all MIME types supported by Tika.
// endpoint: GET /mime-types  Accept: application/json
func (c *TikaClient) GetMIMETypes() (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/mime-types", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tika mime-types request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse MIME types: %w", err)
	}
	return result, nil
}

// GetParsers returns all parsers available in Tika.
// endpoint: GET /parsers/details  Accept: application/json
func (c *TikaClient) GetParsers() (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/parsers/details", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tika parsers request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse parsers response: %w", err)
	}
	return result, nil
}

// ExtractTextFromURL fetches a remote URL and extracts text via Tika.
func (c *TikaClient) ExtractTextFromURL(docURL string) (string, error) {
	// Download the remote document first
	resp, err := c.httpClient.Get(docURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL %s: %w", docURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read URL content: %w", err)
	}
	ct := resp.Header.Get("Content-Type")
	return c.ExtractText(data, "", ct)
}

// putContent sends data to Tika endpoint and returns the response body as string.
func (c *TikaClient) putContent(endpoint string, data []byte, filename, inputContentType, accept string) (string, error) {
	body, err := c.putContentRaw(endpoint, data, filename, inputContentType, accept)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *TikaClient) putContentRaw(endpoint string, data []byte, filename, inputContentType, accept string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if inputContentType != "" {
		req.Header.Set("Content-Type", inputContentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if filename != "" {
		req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tika request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Tika response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Tika returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

// ExtractTextMultipart uses multipart upload (alternative method).
func (c *TikaClient) ExtractTextMultipart(data []byte, filename string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, bytes.NewReader(data)); err != nil {
		return "", err
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tika", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "text/plain")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Tika multipart request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
