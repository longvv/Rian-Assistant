#!/bin/bash

# Define paths
WORKSPACE="${WORKSPACE:-/home/picoclaw/.picoclaw/workspace/news}"
mkdir -p "$WORKSPACE"
HISTORY_FILE="$WORKSPACE/reported.txt"
touch "$HISTORY_FILE"

# Make sure history file doesn't grow indefinitely (keep last 1000 lines)
tail -n 1000 "$HISTORY_FILE" > "$HISTORY_FILE.tmp" && mv "$HISTORY_FILE.tmp" "$HISTORY_FILE"

if command -v python3 > /dev/null 2>&1; then
    python3 -c '
import urllib.request, xml.etree.ElementTree as ET, os, ssl

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

HISTORY_FILE = os.environ.get("WORKSPACE", "/home/picoclaw/.picoclaw/workspace/news") + "/reported.txt"
try:
    with open(HISTORY_FILE, "r", encoding="utf-8") as f:
        history = set(line.strip() for line in f if line.strip())
except FileNotFoundError:
    history = set()

feeds = [
    ("VnExpress", "https://vnexpress.net/rss/tin-moi-nhat.rss"),
    ("Tuổi Trẻ", "https://tuoitre.vn/rss/tin-moi-nhat.rss"),
    ("Thanh Niên", "https://thanhnien.vn/rss/home.rss"),
    ("VietnamNet", "https://vietnamnet.vn/rss/home.rss"),
    ("Dân Trí", "https://dantri.com.vn/rss/home.rss"),
    ("Tiền Phong", "https://tienphong.vn/rss/home.rss"),
    ("BBC", "https://feeds.bbci.co.uk/news/world/rss.xml"),
    ("TechCrunch", "https://techcrunch.com/feed/"),
    ("The Verge", "https://www.theverge.com/rss/index.xml"),
    ("The New York Times", "https://www.nytimes.com/svc/collections/v1/publish/https://www.nytimes.com/section/world/rss.xml"),
    ("The Wall Street Journal", "https://www.wsj.com/rss/Section/World"),
    ("Reuters", "https://www.reuters.com/world/rss"),
    ("Associated Press", "https://apnews.com/rss/APTopNews"),
    ("The Guardian", "https://www.theguardian.com/world/rss"),
    ("The Economist", "https://www.economist.com/world/rss"),
    ("Bloomberg", "https://www.bloomberg.com/world/rss"),
    ("Financial Times", "https://www.ft.com/world/rss")
]

new_items = []
new_links = set()

for name, url in feeds:
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
        with urllib.request.urlopen(req, timeout=10, context=ctx) as response:
            xml_data = response.read()
            root = ET.fromstring(xml_data)
            count = 0
            for item in root.findall(".//item"):
                if count >= 3: break
                title_elem = item.find("title")
                link_elem = item.find("link")
                if title_elem is not None and link_elem is not None:
                    title, link = title_elem.text, link_elem.text
                    if link and link not in history and link not in new_links:
                        new_items.append(f"- **{name}**: [{title}]({link})")
                        new_links.add(link)
                count += 1
    except Exception as e:
        continue

if not new_items:
    print("Không có tin tức mới.")
else:
    print("🗞️ **TIN TỨC MỚI (24 giờ qua)**\n")
    for item in new_items:
        print(item)
    with open(HISTORY_FILE, "a", encoding="utf-8") as f:
        for link in new_links:
            f.write(link + "\n")
'
else
    echo "Không có tin tức mới. (Requires python3 to fetch RSS)"
fi
