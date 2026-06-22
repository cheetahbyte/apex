package webfetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/cheetahbyte/apex/internal/tools"
)

type WebfetchTool struct{}

const (
	maxResponseBytes = 2 * 1024 * 1024
	browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

type args struct {
	URL string `json:"url"`
}

func (WebfetchTool) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        "web_fetch",
		Description: "Fetches the content of a web page and returns markdown",
		ReadOnly:    true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The HTTP or HTTPS URL of the web page to fetch",
				},
			},
			"required":             []string{"url"},
			"additionalProperties": false,
		},
	}
}

func (WebfetchTool) Execute(ctx context.Context, input json.RawMessage) (tools.ToolResult, error) {
	var parsedArgs args
	if err := json.Unmarshal(input, &parsedArgs); err != nil {
		return tools.ToolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	rawURL := parsedArgs.URL
	if strings.TrimSpace(rawURL) == "" {
		return tools.ToolResult{}, fmt.Errorf("missing url")
	}
	rawURL = strings.TrimSpace(rawURL)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return tools.ToolResult{}, fmt.Errorf("unsupported url scheme %q", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := httpClient.Do(req)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return tools.ToolResult{}, fmt.Errorf("failed to fetch: status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to read body: %w", err)
	}
	if len(body) > maxResponseBytes {
		return tools.ToolResult{}, fmt.Errorf("response too large: limit %d bytes", maxResponseBytes)
	}

	markdown, err := htmltomarkdown.ConvertString(string(body))
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to convert html to markdown: %w", err)
	}

	return tools.ToolResult{Content: markdown}, nil
}
