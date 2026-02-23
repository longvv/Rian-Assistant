package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Global pre-compiled regex to save massive CPU cycles per record
var (
	reHTML = regexp.MustCompile(`(?i)<[^>]*>`)
	reEnt  = regexp.MustCompile(`(?i)&([a-z0-9]+|#[0-9]{1,6}|#x[0-9a-f]{1,6});`)
	reWS   = regexp.MustCompile(`\s+`)
	reLink = regexp.MustCompile(`\]\((https?://[^\)]+)\)`)
)

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func generateSummary(apiKey string, itemsBySource map[string][]NewsItem) string {
	if apiKey == "" {
		return ""
	}

	var promptBuilder strings.Builder
	promptBuilder.WriteString("Đây là danh sách các tin tức vừa được thu thập. ")
	promptBuilder.WriteString("Hãy viết một đoạn TÓM TẮT NGẮN GỌN (khoảng 3-5 câu) bằng tiếng Việt về tình hình tin tức nổi bật trong nước và quốc tế để người đọc có cái nhìn tổng quan. ")
	promptBuilder.WriteString("Không liệt kê chi tiết, chỉ nếu khái quát xu hướng, sự kiện chính.\n\n")

	count := 0
	for source, items := range itemsBySource {
		for _, item := range items {
			if count >= 20 { // Limit to try and keep within context
				break
			}
			promptBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", source, item.Title))
			count++
		}
		if count >= 20 {
			break
		}
	}

	reqBody := OpenRouterRequest{
		Model: "openrouter/nvidia/nemotron-3-nano-30b-a3b:free",
		Messages: []Message{
			{Role: "user", Content: promptBuilder.String()},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("⚠️ Lỗi tạo JSON request gửi AI: %v\n", err)
		return ""
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("⚠️ Lỗi tạo HTTP request gửi AI: %v\n", err)
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/picoclaw/picoclaw")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠️ Lỗi gọi AI: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️ AI trả về HTTP %d\n", resp.StatusCode)
		return ""
	}

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var orResp OpenRouterResponse
	if err := json.Unmarshal(bodyText, &orResp); err != nil {
		return ""
	}

	if len(orResp.Choices) > 0 {
		return orResp.Choices[0].Message.Content
	}

	return ""
}

type RssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"encoded"`
	PubDate     string `xml:"pubDate"`
}

type RssFeed struct {
	Channel struct {
		Items []RssItem `xml:"item"`
	} `xml:"channel"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

type AtomEntry struct {
	Title   string   `xml:"title"`
	Link    AtomLink `xml:"link"`
	Summary string   `xml:"summary"`
	Updated string   `xml:"updated"`
}

type AtomFeed struct {
	Entries []AtomEntry `xml:"entry"`
}

type NewsItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
}

func cleanHTML(s string) string {
	s = strings.ReplaceAll(s, "<![CDATA[", "")
	s = strings.ReplaceAll(s, "]]>", "")

	s = reHTML.ReplaceAllString(s, "")
	s = reEnt.ReplaceAllString(s, "")
	s = reWS.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

func getLink(item RssItem) string {
	l := strings.TrimSpace(item.Link)
	if l != "" {
		return l
	}
	return ""
}

func getDesc(item RssItem) string {
	if item.Description != "" {
		return cleanHTML(item.Description)
	}
	if item.Content != "" {
		return cleanHTML(item.Content)
	}
	return ""
}

type FeedSource struct {
	Name string
	URL  string
}

func fetchFeed(f FeedSource, client *http.Client, history, newLinks map[string]bool, newItemsBySource map[string][]NewsItem, mu *sync.Mutex, bg *sync.WaitGroup) {
	defer bg.Done()
	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var rssFeed RssFeed
	var atomFeed AtomFeed
	var localItems []NewsItem

	// Read body bytes for Unmarshaling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	isAtom := false
	if err := xml.Unmarshal(body, &rssFeed); err != nil || len(rssFeed.Channel.Items) == 0 {
		if err := xml.Unmarshal(body, &atomFeed); err == nil && len(atomFeed.Entries) > 0 {
			isAtom = true
		}
	}

	if isAtom {
		for _, entry := range atomFeed.Entries {
			title := cleanHTML(entry.Title)
			link := strings.TrimSpace(entry.Link.Href)
			desc := cleanHTML(entry.Summary)
			pubdate := cleanHTML(entry.Updated)

			if pubdate == "" {
				pubdate = "Unknown Date"
			}

			mu.Lock()
			if title != "" && link != "" && !history[link] && !newLinks[link] {
				localItems = append(localItems, NewsItem{
					Title:       title,
					Link:        link,
					Description: desc,
					PubDate:     pubdate,
				})
				newLinks[link] = true
			}
			mu.Unlock()
		}
	} else {
		for _, item := range rssFeed.Channel.Items {
			title := cleanHTML(item.Title)
			link := getLink(item)
			desc := getDesc(item)
			pubdate := cleanHTML(item.PubDate)

			if pubdate == "" {
				pubdate = "Unknown Date"
			}

			mu.Lock()
			if title != "" && link != "" && !history[link] && !newLinks[link] {
				localItems = append(localItems, NewsItem{
					Title:       title,
					Link:        link,
					Description: desc,
					PubDate:     pubdate,
				})
				newLinks[link] = true
			}
			mu.Unlock()
		}
	}

	if len(localItems) > 0 {
		mu.Lock()
		newItemsBySource[f.Name] = append(newItemsBySource[f.Name], localItems...)
		mu.Unlock()
	}
}

func main() {
	// Extreme Performance HTTP Transport Configuration
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Max TCP connection wait time
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100, // Reuse connections widely globally
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second, // Hard exit if response hangs infinitely
	}

	// Calculate workspace dir
	scriptDir, err := os.Getwd()
	if err != nil {
		scriptDir = "."
	}
	fallbackWorkspace := filepath.Join(scriptDir, "..", "news")
	workspace := os.Getenv("WORKSPACE")
	if workspace == "" {
		workspace = fallbackWorkspace
	}

	historyFile := filepath.Join(workspace, "reported.md")
	reportsDir := filepath.Join(workspace, "reports")
	os.MkdirAll(reportsDir, 0755)

	// Read history
	history := make(map[string]bool)
	if content, err := os.ReadFile(historyFile); err == nil {
		matches := reLink.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				history[match[1]] = true
			}
		}
	}

	feeds := []FeedSource{
		{"VnExpress", "https://vnexpress.net/rss/tin-moi-nhat.rss"},
		{"Tuổi Trẻ", "https://tuoitre.vn/rss/tin-moi-nhat.rss"},
		{"Thanh Niên", "https://thanhnien.vn/rss/home.rss"},
		{"VietnamNet", "https://vietnamnet.vn/rss/home.rss"},
		{"Dân Trí", "https://dantri.com.vn/rss/home.rss"},
		{"Tiền Phong", "https://tienphong.vn/rss/home.rss"},
		{"BBC", "https://feeds.bbci.co.uk/news/world/rss.xml"},
		{"TechCrunch", "https://techcrunch.com/feed/"},
		{"The Verge", "https://www.theverge.com/rss/index.xml"},
		{"The New York Times", "https://www.nytimes.com/svc/collections/v1/publish/https://www.nytimes.com/section/world/rss.xml"},
		{"The Wall Street Journal", "https://www.wsj.com/rss/Section/World"},
		{"Reuters", "https://www.reuters.com/world/rss"},
		{"Associated Press", "https://apnews.com/rss/APTopNews"},
		{"The Guardian", "https://www.theguardian.com/world/rss"},
		{"The Economist", "https://www.economist.com/world/rss"},
		{"Bloomberg", "https://www.bloomberg.com/world/rss"},
		{"Financial Times", "https://www.ft.com/world/rss"},
	}

	var bg sync.WaitGroup
	var mu sync.Mutex

	newItemsBySource := make(map[string][]NewsItem)
	newLinks := make(map[string]bool)

	// Dispatch fetchers concurrently
	for _, feed := range feeds {
		bg.Add(1)
		go fetchFeed(feed, client, history, newLinks, newItemsBySource, &mu, &bg)
	}

	bg.Wait()

	hasNew := false
	for _, items := range newItemsBySource {
		if len(items) > 0 {
			hasNew = true
			break
		}
	}

	if !hasNew {
		fmt.Println("✨ Hiện tại không có tin tức mới nào từ các nguồn.")
		return
	}

	now := time.Now()
	timestamp := now.Format("20060102_150405")
	humanTime := now.Format("2006-01-02 15:04:05")
	reportFilename := fmt.Sprintf("news_report_%s.md", timestamp)
	reportPath := filepath.Join(reportsDir, reportFilename)

	var reportLines []string
	reportLines = append(reportLines, "# 🌟 BẢN TIN TỔNG HỢP MỚI NHẤT 🌟\n")
	reportLines = append(reportLines, fmt.Sprintf("_Cập nhật lúc: %s_\n", humanTime))

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	summary := generateSummary(apiKey, newItemsBySource)
	if summary != "" {
		reportLines = append(reportLines, "## 🤖 AI Tóm Tắt Nhanh\n")
		reportLines = append(reportLines, fmt.Sprintf("> *%s*\n\n", summary))
	}

	reportLines = append(reportLines, "Dưới đây là các tin tức mới được biên tập từ các nguồn:\n")
	reportLines = append(reportLines, "---\n")

	var historyLines []string
	historyLines = append(historyLines, fmt.Sprintf("\n# 📝 TỔNG HỢP LÚC %s\n\n", humanTime))
	if summary != "" {
		historyLines = append(historyLines, "## 🤖 AI Tóm Tắt Nhanh\n")
		historyLines = append(historyLines, fmt.Sprintf("\n> *%s*\n\n---\n\n", summary))
	}

	for _, feed := range feeds {
		items := newItemsBySource[feed.Name]
		if len(items) == 0 {
			continue
		}

		reportLines = append(reportLines, fmt.Sprintf("## 📰 **%s**\n\n", feed.Name))
		historyLines = append(historyLines, fmt.Sprintf("## 🌐 %s\n\n", feed.Name))

		for _, item := range items {
			reportLines = append(reportLines, fmt.Sprintf("### 🔹 [%s](%s)\n", item.Title, item.Link))
			reportLines = append(reportLines, fmt.Sprintf("**🗓 Ngày đăng:** %s\n\n", item.PubDate))
			if item.Description != "" {
				reportLines = append(reportLines, fmt.Sprintf("> *%s*\n\n", item.Description))
			}
			reportLines = append(reportLines, "---\n\n")

			historyDesc := ""
			if item.Description != "" {
				historyDesc = "\n> " + item.Description
			}
			historyLines = append(historyLines, fmt.Sprintf("- **[%s](%s)**\n🗓 _%s_%s\n\n---\n\n", item.Title, item.Link, item.PubDate, historyDesc))
		}
	}

	markdownContent := strings.Join(reportLines, "\n")
	err = os.WriteFile(reportPath, []byte(markdownContent), 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write report: %v\n", err)
		return
	}

	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to open history file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(strings.Join(historyLines, "")); err != nil {
		fmt.Printf("❌ Failed to append history: %v\n", err)
		return
	}

	fmt.Printf("✅ Đã tạo báo cáo tin tức tại: %s\n", reportPath)
	fmt.Printf("📝 Lịch sử được cập nhật tại: %s\n\n", historyFile)
	fmt.Println(markdownContent)
}
