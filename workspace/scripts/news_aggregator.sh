#!/bin/sh

# Define paths
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE="${WORKSPACE:-$WORKSPACE_DIR/news}"

mkdir -p "$WORKSPACE"
HISTORY_FILE="$WORKSPACE/reported.md"
touch "$HISTORY_FILE"

export WORKSPACE

if command -v go > /dev/null 2>&1; then
    cd "$SCRIPT_DIR" && go run news_aggregator.go
else
    echo "✨ Hiện tại không có tin tức mới nào. (Requires Go to fetch RSS)"
fi
