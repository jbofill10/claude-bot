# Stage 1: Build frontend
FROM node:22-alpine AS frontend-build
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
# Debian-based image required — go-sqlite3 uses CGO with glibc
FROM golang:1.23-bookworm AS go-build
RUN apt-get update && apt-get install -y gcc && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -o /claude-bot ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim

# Install base dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 22 (required for Claude CLI)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI
RUN npm install -g @anthropic-ai/claude-code

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y gh \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/bash botuser

# Set up app directory — must be /app so relative path "frontend/dist" resolves
WORKDIR /app

# Copy binary and frontend assets
COPY --from=go-build /claude-bot /app/claude-bot
COPY --from=frontend-build /build/frontend/dist /app/frontend/dist

# Create data and git directories
RUN mkdir -p /data /home/botuser/git \
    && chown -R botuser:botuser /data /home/botuser

# Allow git operations on any mounted repo
RUN git config --global safe.directory '*'

USER botuser

ENV DB_PATH=/data/claude-bot.db
ENV ADDR=:3111

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:3111/api/users || exit 1

EXPOSE 3111
CMD ["/app/claude-bot"]
