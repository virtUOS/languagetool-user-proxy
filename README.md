# LanguageTool User Proxy

A fast and lightweight HTTP proxy for LanguageTool with OIDC authentication and per-user API keys.

## Features

- **OIDC Authentication**: Login via any OpenID Connect provider (Keycloak, Dex, Auth0, etc.)
- **Per-User API Keys**: Each user gets a unique 64-character API key
- **Simple UI**: Single-page dashboard with pure HTML/CSS
- **API Key Regeneration**: Users can regenerate their API key at any time
- **Reverse Proxy**: Proxies requests to LanguageTool backend with API key validation
- **Per-Key Rate Limiting**: Each API key has an independent token bucket (default: 5 req/s, burst 10)

## Quick Start

### 1. Build the binary

```bash
go build -o languagetool-proxy ./cmd/server
```

### 2. Configure

Copy the example file and fill in your values.
The example file contains detailed descriptions of all configuration options.

```bash
cp .env.example .env
```

### 3. Run the server

```bash
./languagetool-proxy
```

The server will automatically load the `.env` file if it exists in the current directory.

## Usage

### User Flow

1. User visits the proxy URL
2. User is redirected to OIDC login
3. After successful login, user is redirected to the dashboard
4. Dashboard shows the user's API key
5. User can use the API key in their LanguageTool requests

### API Endpoint Format

```
https://your-proxy-domain.com/{API_KEY}/v2/
```

You can use cURL to test the API:

```sh
curl -si \
  -d "language=en-US" \
  -d "text=a simple test" \
  https://proxy.example.com/a1b2c3d4e5f6.../v2/check
```

### API Key Regeneration

- Click "Regenerate API Key" on the dashboard
- Old key is immediately invalidated
- New 64-character key is generated and displayed
- Update your applications with the new key

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Browser   │────▸│  LanguageTool    │────▸│  LanguageTool   │
│             │     │  User Proxy      │     │  Backend        │
└─────────────┘     └──────────────────┘     └─────────────────┘
                            │
                            ▾
                    ┌──────────────────┐
                    │   SQLite DB      │
                    │  (users, keys)   │
                    └──────────────────┘
```

## Security

- API keys are stored as SHA-256 hashes (never plain text)
- Sessions use secure, HTTP-only cookies
- Session expiration with auto-extend
- Per-key rate limiting protects the backend from individual key abuse (429 with `Retry-After` header)
- HTTPS should be handled by a reverse proxy (Caddy, Nginx)

## Building for Production

### Docker

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### Static Binary

```bash
go build -o languagetool-user-proxy cmd/server/main.go
```

## License

MIT
