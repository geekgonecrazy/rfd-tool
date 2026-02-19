# RFD Tool - Project Overview

## Description

RFD Tool is a web application for managing and displaying Requests for Discussion (RFDs) / Architecture Decision Records (ADRs). It syncs with a GitHub repository containing RFD markdown files and provides a web interface to browse, view, and create new RFDs.

Inspired by [Oxide's RFD process](https://oxide.computer/blog/rfd-1-requests-for-discussion).

## Architecture

### Tech Stack
- **Language**: Go 1.24
- **Web Framework**: Gin
- **Database**: SQLite (primary) or BoltDB
- **Authentication**: OIDC (OpenID Connect)
- **Templates**: Go HTML templates
- **Styling**: Custom CSS (dark theme)

### Project Structure

```
├── cmd/
│   ├── rfd-server/      # Main web server
│   └── rfd-client/      # CLI client
├── config/              # Configuration loading
├── controllers/         # HTTP handlers (pages & API)
├── core/                # Business logic
│   ├── rfd.go           # RFD operations
│   ├── github.go        # GitHub integration
│   ├── oidc.go          # OIDC authentication
│   └── sessionToken.go  # JWT session handling
├── models/              # Data models
├── renderer/            # Markdown rendering
│   └── d2/              # D2 diagram support
├── router/              # Route definitions
├── store/               # Data persistence
│   ├── sqlitestore/     # SQLite implementation
│   └── boltstore/       # BoltDB implementation
├── templates/           # HTML templates
├── assets/              # Static assets (CSS, logo)
├── webhook/             # GitHub webhook handler
└── utils/               # Utility functions
```

### Key Features
- **GitHub Sync**: Pulls RFD markdown files from a configured GitHub repo
- **OIDC Auth**: Supports any OIDC provider (Dex for local dev)
- **Markdown Rendering**: Goldmark with syntax highlighting
- **Diagram Support**: Mermaid and D2 diagrams
- **Tagging & Authors**: Filter RFDs by tags or authors
- **Create RFDs**: Web form to create new RFDs (commits to GitHub)

### Configuration

Configuration via `config.yaml`:
- `site.url` - Public URL of the application
- `repo.url` - GitHub repo URL for RFDs
- `repo.folder` - Folder within repo containing RFDs
- `oidc.*` - OIDC provider settings
- `jwt.*` - JWT signing keys for sessions

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | RFD list (redirects to login if not authenticated) |
| GET | `/:id` | View single RFD |
| GET | `/tag/:tag` | Filter RFDs by tag |
| GET | `/author/:author` | Filter RFDs by author |
| GET | `/create` | Create RFD form |
| GET | `/api/v1/rfds` | List all RFDs (JSON) |
| GET | `/api/v1/rfds/:id` | Get single RFD (JSON) |
| POST | `/api/v1/rfds` | Create new RFD |

### Local Development

1. Run `./scripts/setup-dev.sh` to generate keys and config
2. Add deploy key to GitHub repo
3. Start Dex: `docker compose up -d`
4. Build and run: `go build -o rfd-server ./cmd/rfd-server && ./rfd-server`
5. Open http://localhost:8877

Test users (password: `password`):
- alice@acme.com (engineering, engineering-leads)
- bob@acme.com (engineering)
- carol@acme.com (design)
- dave@acme.com (engineering, design)

## Recent Changes

- **Button Contrast Fix**: Changed Create button from blue (#4299e1) to green (#22c55e) for better visibility on dark background
