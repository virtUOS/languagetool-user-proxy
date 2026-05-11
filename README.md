# LanguageTool User Proxy

A fast and lightweight HTTP proxy for LanguageTool with OIDC authentication and per-user API keys.

## Features

- **OIDC Authentication**: Login via any OpenID Connect provider (Keycloak, Dex, Auth0, etc.)
- **Per-User API Keys**: Each user gets a unique 64-character API key
- **Simple UI**: Single-page dashboard with pure HTML/CSS
- **API Key Regeneration**: Users can regenerate their API key at any time
- **Reverse Proxy**: Proxies requests to LanguageTool backend with API key validation

## Quick Start

### 1. Build the binary

```bash
go build -o languagetool-proxy ./cmd/server
```

### 2. Configure

#### Option A: Using a `.env` file (Recommended)

Copy the example file and fill in your values:

```bash
cp .env.example .env
# Edit .env with your configuration
```

#### Option B: Using environment variables

```bash
export PORT=8080
export DATABASE_PATH=./data/languagetool.db
export OIDC_ISSUER_URL=https://your-keycloak.example.com/realms/yourrealm
export OIDC_CLIENT_ID=languagetool-proxy
export OIDC_CLIENT_SECRET=your-client-secret
export OIDC_REDIRECT_URI=https://your-domain.com/callback
export BACKEND_URL=http://localhost:8080
export SESSION_DURATION_HOURS=24
export COOKIE_SECRET=your-random-secret-key
```

**Note**: Environment variables take precedence over `.env` file values.

### 3. Run the server

```bash
./languagetool-proxy
```

The server will automatically load the `.env` file if it exists in the current directory.

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | Server port |
| `DATABASE_PATH` | No | `./data/languagetool.db` | SQLite database path |
| `OIDC_ISSUER_URL` | Yes | - | OIDC provider issuer URL |
| `OIDC_CLIENT_ID` | Yes | - | OIDC client ID |
| `OIDC_CLIENT_SECRET` | Yes | - | OIDC client secret |
| `OIDC_REDIRECT_URI` | Yes | - | Callback URL for OIDC |
| `OIDC_SCOPE` | No | `openid profile email` | OIDC scopes |
| `BACKEND_URL` | Yes | - | LanguageTool backend URL |
| `SESSION_DURATION_HOURS` | No | `24` | Session duration in hours |
| `COOKIE_SECRET` | No | (auto-generated) | Secret for cookie encryption |

## Usage

### User Flow

1. User visits the proxy URL
2. User is redirected to OIDC login
3. After successful login, user is redirected to the dashboard
4. Dashboard shows the user's API key
5. User can use the API key in their LanguageTool requests

### API Endpoint Format

```
https://your-proxy-domain.com/{API_KEY}/v2/check
```

Example:
```
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
│   Browser   │────▶│  LanguageTool    │────▶│  LanguageTool   │
│             │     │  User Proxy      │     │  Backend        │
└─────────────┘     └──────────────────┘     └─────────────────┘
                            │
                            ▼
                    ┌──────────────────┐
                    │   SQLite DB      │
                    │  (users, keys)   │
                    └──────────────────┘
```

## Security

- API keys are stored as SHA-256 hashes (never plain text)
- Sessions use secure, HTTP-only cookies
- Session expiration with auto-extend
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
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server
```

## License

MIT
