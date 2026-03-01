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

## Data Management

### Import ADRs/RFDs

The RFD tool includes a client for importing existing ADRs from external repositories:

```bash
# Build the import client
go build -o rfd-client ./cmd/rfd-client

# Set environment variables for your production instance
export RFD_SERVER=https://your-rfd-site.com
export RFD_TOKEN=your-api-secret-token

# Import from main folder (e.g., for ADRs in a single directory)
./rfd-client -import -folder /path/to/adrs -skip-discussion

# Import from git branches (e.g., for branch-based ADR workflows)
./rfd-client -import-branches -repo /path/to/repo -rfd-folder adr -skip-discussion

# Import a single RFD
./rfd-client -rfd 0001 -folder /path/to/adrs
```

**Parameters:**
- `-import`: Import all ADRs from a folder
- `-import-branches`: Import ADRs from git branches in a repository
- `-folder`: Path to folder containing ADR files
- `-repo`: Path to git repository (for branch imports)
- `-rfd-folder`: Folder name within repo containing ADRs (default: "adr")
- `-skip-discussion`: Skip creating GitHub discussions during bulk imports
- `-rfd NNNN`: Import a specific RFD by number

### API Access

All API endpoints use token authentication. Include the API token in requests:

```bash
# Get all authors
curl -H "api-token: your-token" "https://your-rfd-site.com/api/v1/authors"

# Get all RFDs
curl -H "api-token: your-token" "https://your-rfd-site.com/api/v1/rfds"

# Get RFDs by author
curl -H "api-token: your-token" "https://your-rfd-site.com/api/v1/authors/{author-id}/rfds"

# Get a specific RFD
curl -H "api-token: your-token" "https://your-rfd-site.com/api/v1/rfds/{rfd-id}"
```

### Database Migrations

The system includes an automatic migration system:

```bash
# Run pending migrations
./rfd-server migrate -configFile config.yaml

# Check migration status  
./rfd-server migrate -configFile config.yaml status
```

Migrations are automatically applied on server startup, but you can run them manually if needed.

