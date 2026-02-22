#!/bin/sh
# News Aggregator Script
# Tổng hợp tin tức Việt Nam và quốc tế, gửi vào 8:00 sáng hàng ngày
# POSIX sh-compatible — works with busybox/Alpine environments (no bash required)

# Working directory
WORKSPACE="/home/picoclaw/.picoclaw/workspace"
cd "$WORKSPACE" || { echo "ERROR: Cannot cd to $WORKSPACE" >&2; exit 1; }

# Output file
NEWS_FILE="daily-news-$(date +%Y%m%d).md"

# Write header
printf '# TIN TỨC HÔM NAY - %s\n\n' "$(date +'%d/%m/%Y')" > "$NEWS_FILE"

# ---------------------------------------------------------------------------
# fetch_news: fetch an RSS/XML feed and extract <title> elements
#   $1 = section heading (display name)
#   $2 = RSS feed URL
#   $3 = max items (default 10 if not given)
# ---------------------------------------------------------------------------
fetch_news() {
    section_name="$1"
    url="$2"
    max_items="${3:-10}"

    printf '## %s\n\n' "$section_name" >> "$NEWS_FILE"

    # Fetch the RSS feed (XML).  Extract text between <title> tags.
    # - Strip CDATA wrappers:  <title><![CDATA[...]]></title>
    # - Strip plain tags:       <title>...</title>
    # Pipe the result line-by-line via POSIX-compatible construct.
    content=$(curl -s -A "Mozilla/5.0 (compatible; NewsBot/1.0)" "$url" \
        | grep -o '<title>[^<]*</title>' \
        | sed 's/<title><!\[CDATA\[//g; s/\]\]><\/title>//g; s/<\/title>//g; s/<title>//g' \
        | grep -v '^\s*$' \
        | head -"$max_items")

    if [ -z "$content" ]; then
        printf '- _(Không tải được tin tức từ nguồn này)_\n\n' >> "$NEWS_FILE"
        return
    fi

    # Iterate line-by-line — POSIX compatible (no <<< here-string)
    echo "$content" | while IFS= read -r line; do
        if [ -n "$line" ]; then
            printf -- '- **%s**\n  [Đọc thêm](%s)\n\n' "$line" "$url" >> "$NEWS_FILE"
        fi
    done

    printf '\n' >> "$NEWS_FILE"
}

# ---------------------------------------------------------------------------
# Vietnamese news sources
# Use category-specific RSS feeds (the generic /rss URL is an HTML page)
# ---------------------------------------------------------------------------
fetch_news "VnExpress - Tin mới nhất"  "https://vnexpress.net/rss/tin-moi-nhat.rss"  10
fetch_news "Tuổi Trẻ Online"           "https://tuoitre.vn/rss/tin-moi-nhat.rss"     10
fetch_news "Thanh Niên"                "https://thanhnien.vn/rss/home.rss"            10

# ---------------------------------------------------------------------------
# International news sources
# ---------------------------------------------------------------------------
fetch_news "BBC News"  "https://feeds.bbci.co.uk/news/world/rss.xml"  10
fetch_news "Reuters"   "https://feeds.reuters.com/reuters/topNews"     10

# ---------------------------------------------------------------------------
# Deliver the news file
# ---------------------------------------------------------------------------
if [ -f "$NEWS_FILE" ]; then
    if command -v telegram-send > /dev/null 2>&1; then
        telegram-send --file "$NEWS_FILE"
    else
        echo "News file created: $NEWS_FILE"
        echo "Tip: install telegram-send to enable automatic delivery."
    fi
else
    echo "ERROR: News file was not created." >&2
    exit 1
fi
