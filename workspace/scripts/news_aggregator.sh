#!/bin/bash

# Define paths
WORKSPACE="${WORKSPACE:-/home/picoclaw/.picoclaw/workspace/news}"
mkdir -p "$WORKSPACE"
HISTORY_FILE="$WORKSPACE/reported.txt"
touch "$HISTORY_FILE"

# Make sure history file doesn't grow indefinitely (keep last 5000 lines because we fetch all items now)
tail -n 5000 "$HISTORY_FILE" > "$HISTORY_FILE.tmp" && mv "$HISTORY_FILE.tmp" "$HISTORY_FILE"

export WORKSPACE

if command -v python3 > /dev/null 2>&1; then
    python3 -c '
import urllib.request
import xml.etree.ElementTree as ET
import os
import ssl
import re
from collections import defaultdict

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

WORKSPACE = os.environ.get("WORKSPACE", "/home/picoclaw/.picoclaw/workspace/news")
HISTORY_FILE = os.path.join(WORKSPACE, "reported.txt")
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

def clean_html(raw_html):
    # Lọc bỏ data dư thừa và thẻ HTML
    cleanr_cdata = re.compile(r"<!\[CDATA\[(.*?)\]\]>", re.DOTALL)
    text = re.sub(cleanr_cdata, r"\1", str(raw_html))
    cleanr_html = re.compile(r"<.*?>|&([a-z0-9]+|#[0-9]{1,6}|#x[0-9a-f]{1,6});")
    text = re.sub(cleanr_html, "", text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()

new_items_by_source = defaultdict(list)
new_links = set()

for name, url in feeds:
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"})
        with urllib.request.urlopen(req, timeout=15, context=ctx) as response:
            xml_data = response.read()
            root = ET.fromstring(xml_data)
            
            # Quét tất cả item/entry (rss hoặc atom)
            items = root.findall(".//item")
            if not items:
                items = root.findall(".//{http://www.w3.org/2005/Atom}entry")
            if not items:
                items = root.findall(".//entry")
                
            for item in items:
                title_elem = item.find("title")
                if title_elem is None:
                    title_elem = item.find("{http://www.w3.org/2005/Atom}title")
                title = title_elem.text.strip() if title_elem is not None and title_elem.text else ""
                
                link_elem = item.find("link")
                if link_elem is None:
                    link_elem = item.find("{http://www.w3.org/2005/Atom}link")
                
                link = ""
                if link_elem is not None:
                    if link_elem.text and link_elem.text.strip() and not link_elem.text.strip().startswith("\n"):
                        link = link_elem.text.strip()
                    elif link_elem.get("href"):
                        link = link_elem.get("href").strip()
                
                # Fetch description / summary để làm tóm tắt
                desc_elem = item.find("description")
                if desc_elem is None:
                    desc_elem = item.find("summary")
                if desc_elem is None:
                    desc_elem = item.find("{http://www.w3.org/2005/Atom}summary")
                
                desc = clean_html(desc_elem.text) if desc_elem is not None and desc_elem.text else ""
                if len(desc) > 250:
                    desc = desc[:247] + "..."
                
                if title and link and link not in history and link not in new_links:
                    new_items_by_source[name].append((title, link, desc))
                    new_links.add(link)
    except Exception as e:
        continue

if not new_items_by_source:
    print("✨ Hiện tại không có tin tức mới nào từ các nguồn.")
else:
    print("🌟 **BẢN TIN TỔNG HỢP MỚI NHẤT** 🌟\n")
    print("Dưới đây là tất cả các tin tức mới được cập nhật từ các nguồn:\n")
    print("---")
    
    for source, items in new_items_by_source.items():
        print(f"\n### 📰 **{source}**")
        for title, link, desc in items:
            print(f"🔹 **[{title}]({link})**")
            if desc:
                print(f"   📝 _{desc}_")
            print() # Dòng trống phân chia các tin
        
    with open(HISTORY_FILE, "a", encoding="utf-8") as f:
        for link in new_links:
            f.write(link + "\n")
'
else
    echo "✨ Hiện tại không có tin tức mới nào. (Requires python3 to fetch RSS)"
fi
