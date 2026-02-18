# RFD Tool

**Very much a WIP tool.** Created for specific need I found with my engineering team. Your results may vary

If you want to find out more about RFD's see Oxide's post [here](https://oxide.computer/blog/rfd-1-requests-for-discussion)

This post is great for outlining the process. But they use an all-in-one CIO tool to handle their RFD's.  If you want to be able to display the RFDs like they do you are on your own.

This tool looks to fill this gap.

## Local Development

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- A GitHub repo for storing RFDs (can use [rfd-example](https://github.com/geekgonecrazy/rfd-example) as a template)

### Quick Start

1. **Run the setup script** to generate keys and config:
   ```bash
   ./scripts/setup-dev.sh
   ```

2. **Add the deploy key** to your GitHub repo:
   - Go to your repo → Settings → Deploy keys → Add deploy key
   - Paste the public key shown by the setup script
   - Check "Allow write access" if you want to create RFDs

3. **Start Dex** (local OIDC provider):
   ```bash
   docker compose up -d
   ```

4. **Build and run** the server:
   ```bash
   go build -o rfd-server ./cmd/rfd-server
   ./rfd-server
   ```

5. **Open** http://localhost:8877 and sign in

### Test Users

The local Dex instance has these pre-configured users (password: `password`):

| Email | Groups |
|-------|--------|
| alice@acme.com | engineering, engineering-leads |
| bob@acme.com | engineering |
| carol@acme.com | design |
| dave@acme.com | engineering, design |

### Configuration

The setup script generates `config.yaml` with sensible defaults for local development. Key settings:

- **site.url**: Where the app is hosted (default: `http://localhost:8877`)
- **repo.url**: Your GitHub repo containing RFDs
- **repo.folder**: Folder within the repo where RFDs are stored (default: `rfds`)
- **oidc.***: OIDC provider settings (pre-configured for local Dex)

See `config.example.yaml` for production configuration options.

### Manual Setup

If you prefer to set up manually instead of using the script:

```bash
# Generate JWT keys
openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem

# Generate deploy key
ssh-keygen -t rsa -b 4096 -f deploy_key -N ""

# Copy and edit config
cp config.dev.yaml config.yaml
# Edit config.yaml to add your keys
```

## Diagram Support

RFD Tool supports both Mermaid and D2 diagrams within your RFD documents:

- **Mermaid** - Excellent for flowcharts, sequence diagrams, and Git graphs
- **D2** - Great for architectural diagrams, entity relationships, and complex layouts with custom styling

Both diagram types can be used together in the same document. See the [example RFD](docs/example-rfd.md) for a comprehensive demonstration of mixed diagram usage.

