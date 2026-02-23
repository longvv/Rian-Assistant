# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make build
RUN go build -o /src/build/news_aggregator workspace/scripts/news_aggregator.go

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl python3 py3-requests

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY --from=builder /src/build/news_aggregator /usr/local/bin/news_aggregator

# Create non-root user and group
RUN addgroup -g 1000 picoclaw && \
    adduser -D -u 1000 -G picoclaw picoclaw

# Switch to non-root user
USER picoclaw

# Run onboard to create initial directories and config
RUN /usr/local/bin/picoclaw onboard

# Override the generated config with our custom config
# This allows environment variables to drive provider/model selection on Railway
COPY --chown=picoclaw:picoclaw config/config.json /home/picoclaw/.picoclaw/config.json

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
