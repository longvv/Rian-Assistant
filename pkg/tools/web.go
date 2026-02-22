package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// ─── Simple LRU fetch cache ───────────────────────────────────────────────────
// Caches web_fetch results by URL in-memory to avoid redundant HTTP calls
// during a single response chain. Holds at most fetchCacheMaxSize entries.

const fetchCacheMaxSize = 64

type fetchCacheEntry struct {
	result string
	at     time.Time
}

type fetchCache struct {
	mu   sync.Mutex
	keys []string // insertion-order ring for eviction
	data map[string]fetchCacheEntry
	ttl  time.Duration
}

func newFetchCache(ttl time.Duration) *fetchCache {
	return &fetchCache{
		keys: make([]string, 0, fetchCacheMaxSize),
		data: make(map[string]fetchCacheEntry, fetchCacheMaxSize),
		ttl:  ttl,
	}
}

func (c *fetchCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok {
		return "", false
	}
	if c.ttl > 0 && time.Since(e.at) > c.ttl {
		delete(c.data, key)
		return "", false
	}
	return e.result, true
}

func (c *fetchCache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.data[key]; !exists {
		if len(c.keys) >= fetchCacheMaxSize {
			// Evict oldest
			evict := c.keys[0]
			c.keys = c.keys[1:]
			delete(c.data, evict)
		}
		c.keys = append(c.keys, key)
	}
	c.data[key] = fetchCacheEntry{result: value, at: time.Now()}
}

// globalFetchCache is shared across all WebFetchTool instances within a process.
// TTL of 10 minutes: stale enough to avoid hammering, fresh enough not to mislead.
var globalFetchCache = newFetchCache(10 * time.Minute)

type SearchProvider interface {
	Search(ctx context.Context, query string, count int) (string, error)
}

type DuckDuckGoSearchProvider struct{}

func (p *DuckDuckGoSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return p.extractResults(string(body), count, query)
}

func (p *DuckDuckGoSearchProvider) extractResults(html string, count int, query string) (string, error) {
	// Simple regex based extraction for DDG HTML
	// Strategy: Find all result containers or key anchors directly

	// Try finding the result links directly first, as they are the most critical
	// Pattern: <a class="result__a" href="...">Title</a>
	// The previous regex was a bit strict. Let's make it more flexible for attributes order/content
	reLink := regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	matches := reLink.FindAllStringSubmatch(html, count+5)

	if len(matches) == 0 {
		return fmt.Sprintf("No results found or extraction failed. Query: %s", query), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Results for: %s (via DuckDuckGo)", query))

	// Pre-compile snippet regex to run inside the loop
	// We'll search for snippets relative to the link position or just globally if needed
	// But simple global search for snippets might mismatch order.
	// Since we only have the raw HTML string, let's just extract snippets globally and assume order matches (risky but simple for regex)
	// Or better: Let's assume the snippet follows the link in the HTML

	// A better regex approach: iterate through text and find matches in order
	// But for now, let's grab all snippets too
	reSnippet := regexp.MustCompile(`<a class="result__snippet[^"]*".*?>([\s\S]*?)</a>`)
	snippetMatches := reSnippet.FindAllStringSubmatch(html, count+5)

	maxItems := min(len(matches), count)

	for i := 0; i < maxItems; i++ {
		urlStr := matches[i][1]
		title := stripTags(matches[i][2])
		title = strings.TrimSpace(title)

		// URL decoding if needed
		if strings.Contains(urlStr, "uddg=") {
			if u, err := url.QueryUnescape(urlStr); err == nil {
				idx := strings.Index(u, "uddg=")
				if idx != -1 {
					urlStr = u[idx+5:]
				}
			}
		}

		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, title, urlStr))

		// Attempt to attach snippet if available and index aligns
		if i < len(snippetMatches) {
			snippet := stripTags(snippetMatches[i][1])
			snippet = strings.TrimSpace(snippet)
			if snippet != "" {
				lines = append(lines, fmt.Sprintf("   %s", snippet))
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

func stripTags(content string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(content, "")
}

// ─── SearXNG Provider ────────────────────────────────────────────────────────

var defaultSearXNGInstances = []string{
	"https://searx.be",
	"https://searx.fyi",
	"https://searxng.site",
	"https://search.mdosch.de",
	"https://searx.sp-codes.de",
}

// SearXNGSearchProvider queries SearXNG instances over their JSON API.
// If a custom baseURL is provided, it uses only that.
// Otherwise, it automatically rotates through a list of public instances.
type SearXNGSearchProvider struct {
	baseURLs []string
}

func (p *SearXNGSearchProvider) Search(ctx context.Context, query string, count int) (string, error) {
	var lastErr error

	for _, baseURL := range p.baseURLs {
		searchURL := fmt.Sprintf("%s/search?q=%s&format=json&engines=google,bing,duckduckgo,wikipedia",
			strings.TrimRight(baseURL, "/"), url.QueryEscape(query))

		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}
		req.Header.Set("User-Agent", userAgent)

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request to %s failed: %w", baseURL, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response from %s: %w", baseURL, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("SearXNG instance %s returned status %d", baseURL, resp.StatusCode)
			continue
		}

		var searchResp struct {
			Results []struct {
				Title   string  `json:"title"`
				URL     string  `json:"url"`
				Content string  `json:"content"`
				Score   float64 `json:"score"`
				Engine  string  `json:"engine"`
			} `json:"results"`
		}

		if err := json.Unmarshal(body, &searchResp); err != nil {
			lastErr = fmt.Errorf("failed to parse SearXNG response from %s: %w", baseURL, err)
			continue
		}

		results := searchResp.Results
		if len(results) == 0 {
			return fmt.Sprintf("No results for: %s", query), nil
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Results for: %s (via SearXNG %s — multi-engine)", query, baseURL))
		maxItems := count
		if maxItems > len(results) {
			maxItems = len(results)
		}
		for i := 0; i < maxItems; i++ {
			r := results[i]
			lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, r.Title, r.URL))
			if r.Content != "" {
				lines = append(lines, fmt.Sprintf("   %s", r.Content))
			}
		}
		return strings.Join(lines, "\n"), nil
	}

	return "", fmt.Errorf("all SearXNG instances failed. Last error: %w", lastErr)
}

type WebSearchTool struct {
	provider   SearchProvider
	maxResults int
}

type WebSearchToolOptions struct {
	DuckDuckGoMaxResults int
	DuckDuckGoEnabled    bool
	// SearXNG — self-hosted meta-search engine (docker run -p 8080:8080 searxng/searxng)
	SearXNGURL        string
	SearXNGMaxResults int
	SearXNGEnabled    bool
}

func NewWebSearchTool(opts WebSearchToolOptions) *WebSearchTool {
	var provider SearchProvider
	maxResults := 5

	// Priority: SearXNG > DuckDuckGo
	if opts.SearXNGEnabled {
		urls := defaultSearXNGInstances
		if opts.SearXNGURL != "" {
			urls = []string{opts.SearXNGURL}
		}
		provider = &SearXNGSearchProvider{baseURLs: urls}
		if opts.SearXNGMaxResults > 0 {
			maxResults = opts.SearXNGMaxResults
		}
	} else if opts.DuckDuckGoEnabled {
		provider = &DuckDuckGoSearchProvider{}
		if opts.DuckDuckGoMaxResults > 0 {
			maxResults = opts.DuckDuckGoMaxResults
		}
	} else {
		return nil
	}

	return &WebSearchTool{
		provider:   provider,
		maxResults: maxResults,
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for current information. Returns titles, URLs, and snippets from search results."
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results (1-10)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	query, ok := args["query"].(string)
	if !ok {
		return ErrorResult("query is required")
	}

	count := t.maxResults
	if c, ok := args["count"].(float64); ok {
		if int(c) > 0 && int(c) <= 10 {
			count = int(c)
		}
	}

	result, err := t.provider.Search(ctx, query, count)
	if err != nil {
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}

type WebFetchTool struct {
	maxChars          int
	jinaReaderEnabled bool
}

func NewWebFetchTool(maxChars int) *WebFetchTool {
	if maxChars <= 0 {
		maxChars = 50000
	}
	return &WebFetchTool{
		maxChars:          maxChars,
		jinaReaderEnabled: true, // enabled by default — no API key needed
	}
}

// SetJinaReaderEnabled enables or disables the Jina AI Reader extraction mode.
func (t *WebFetchTool) SetJinaReaderEnabled(v bool) {
	t.jinaReaderEnabled = v
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and extract readable content as clean markdown text. Use this for documentation, news articles, blog posts, or any web content. Results are cached within a session to avoid redundant requests."
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"maxChars": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum characters to extract",
				"minimum":     100.0,
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	urlStr, ok := args["url"].(string)
	if !ok {
		return ErrorResult("url is required")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrorResult("only http/https URLs are allowed")
	}

	if parsedURL.Host == "" {
		return ErrorResult("missing domain in URL")
	}

	maxChars := t.maxChars
	if mc, ok := args["maxChars"].(float64); ok {
		if int(mc) > 100 {
			maxChars = int(mc)
		}
	}

	// ── LRU cache check ──────────────────────────────────────────────────────
	cacheKey := fmt.Sprintf("%s|%d", urlStr, maxChars)
	if cached, hit := globalFetchCache.get(cacheKey); hit {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("[cache hit] %s\n\n%s", urlStr, cached),
			ForUser: fmt.Sprintf("[cache hit] %s", urlStr),
		}
	}

	// ── Strategy 1: Jina AI Reader (clean markdown, no API key) ──────────────
	// Jina Reader converts any URL to clean markdown using readability algorithms.
	// It handles JS-rendered content, removes nav/ads, and formats code blocks.
	if t.jinaReaderEnabled {
		jinaURL := "https://r.jina.ai/" + urlStr
		jinaReq, err := http.NewRequestWithContext(ctx, "GET", jinaURL, nil)
		if err == nil {
			jinaReq.Header.Set("User-Agent", userAgent)
			jinaReq.Header.Set("Accept", "text/markdown, text/plain")
			jinaClient := &http.Client{Timeout: 20 * time.Second}
			jinaResp, jinaErr := jinaClient.Do(jinaReq)
			if jinaErr == nil {
				defer jinaResp.Body.Close()
				jinaBody, readErr := io.ReadAll(jinaResp.Body)
				if readErr == nil && jinaResp.StatusCode == http.StatusOK && len(jinaBody) > 200 {
					text := string(jinaBody)
					truncated := len(text) > maxChars
					if truncated {
						text = text[:maxChars]
					}
					globalFetchCache.set(cacheKey, text)
					return &ToolResult{
						ForLLM:  fmt.Sprintf("Fetched %s (extractor: jina-reader, truncated: %v)\n\nContent:\n%s", urlStr, truncated, text),
						ForUser: fmt.Sprintf("Fetched %s via Jina Reader (%d chars)", urlStr, len(text)),
					}
				}
			}
		}
		// Jina failed or returned thin content — fall through to direct fetch
	}

	// ── Strategy 2: Direct HTTP fetch with content-type aware extraction ──────
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			TLSHandshakeTimeout: 15 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	contentType := resp.Header.Get("Content-Type")

	var text, extractor string

	switch {
	case strings.Contains(contentType, "application/json"):
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			formatted, _ := json.MarshalIndent(jsonData, "", "  ")
			text = string(formatted)
		} else {
			text = string(body)
		}
		extractor = "json"
	case strings.Contains(contentType, "text/html"),
		len(body) > 0 && (strings.HasPrefix(string(body), "<!DOCTYPE") || strings.HasPrefix(strings.ToLower(string(body)), "<html")):
		text = t.extractText(string(body))
		extractor = "html-regex"
	default:
		text = string(body)
		extractor = "raw"
	}

	truncated := len(text) > maxChars
	if truncated {
		text = text[:maxChars]
	}

	globalFetchCache.set(cacheKey, text)

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Fetched %d bytes from %s (extractor: %s, truncated: %v)\n\nContent:\n%s", len(text), urlStr, extractor, truncated, text),
		ForUser: fmt.Sprintf("Fetched %s (%s, %d chars)", urlStr, extractor, len(text)),
	}
}

func (t *WebFetchTool) extractText(htmlContent string) string {
	re := regexp.MustCompile(`<script[\s\S]*?</script>`)
	result := re.ReplaceAllLiteralString(htmlContent, "")
	re = regexp.MustCompile(`<style[\s\S]*?</style>`)
	result = re.ReplaceAllLiteralString(result, "")
	re = regexp.MustCompile(`<[^>]+>`)
	result = re.ReplaceAllLiteralString(result, "")

	result = strings.TrimSpace(result)

	re = regexp.MustCompile(`[^\S\n]+`)
	result = re.ReplaceAllString(result, " ")
	re = regexp.MustCompile(`\n{3,}`)
	result = re.ReplaceAllString(result, "\n\n")

	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}
