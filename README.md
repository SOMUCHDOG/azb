# Azure Boards CLI

A cross-platform command-line interface for managing Azure Boards work items.

## Features

- 🔐 **Secure Authentication**: PAT (Personal Access Token) support with secure storage
- 📋 **Work Item Management**: List, view, create, update, and delete work items
- 🔍 **Powerful Filtering**: Filter by state, assignee, type, sprint, area path, and tags
- 📊 **Multiple Output Formats**: Table, JSON, CSV, and IDs-only formats
- ⚙️ **Configuration Management**: Easy setup and configuration
- 🚀 **Fast & Lightweight**: Single binary, no dependencies required

## Installation

### From Source

```bash
git clone <repository-url>
cd azure-boards-cli
go build -o ab
```

Move the binary to your PATH:

```bash
# macOS/Linux
sudo mv ab /usr/local/bin/

# Or add to your local bin
mv ab ~/bin/
```

## Quick Start

### 1. Configure Organization and Project

```bash
ab config set organization myorg
ab config set project myproject
```

### 2. Authenticate

```bash
ab auth login
```

You'll be prompted to enter your Personal Access Token (PAT).

**Creating a PAT:**
1. Go to `https://dev.azure.com/{org}/_usersSettings/tokens`
2. Click "New Token"
3. Select scopes: `Work Items (Read, Write)`
4. Copy the generated token

### 3. List Work Items

```bash
# List all work items
ab list

# List work items assigned to me
ab list --assigned-to @me

# List active bugs
ab list --type Bug --state Active

# List work items in current sprint
ab list --sprint current
```

### 4. View Work Item Details

```bash
ab show 1234
```

## Usage

### Authentication

```bash
# Login with PAT
ab auth login
ab auth login --pat <your-token>

# Check authentication status
ab auth status

# Logout
ab auth logout
```

### Configuration

```bash
# Set configuration values
ab config set organization myorg
ab config set project myproject
ab config set default_area_path "myproject\\Team A"

# Get configuration value
ab config get organization

# List all configuration
ab config list
```

### List Work Items

```bash
# Basic listing
ab list

# Filter options
ab list --state <state>              # Filter by state (Active, Resolved, Closed)
ab list --assigned-to <user>         # Filter by assignee (@me for current user)
ab list --type <type>                # Filter by work item type (Bug, Task, User Story)
ab list --sprint <sprint>            # Filter by sprint (current, @current, or sprint name)
ab list --area-path <path>           # Filter by area path
ab list --tags <tags>                # Filter by tags (comma-separated)
ab list --limit <n>                  # Limit number of results (default: 50)

# Output formats
ab list --format table               # Table format (default)
ab list --format json                # JSON format
ab list --format csv                 # CSV format
ab list --format ids                 # IDs only (for scripting)

# Examples
ab list --type Bug --assigned-to @me --state Active
ab list --sprint "Sprint 42" --format json
ab list --tags "urgent,security" --limit 20
```

### Show Work Item

```bash
# Show work item details
ab show <id>

# Show with JSON format
ab show 1234 --format json

# Show with comments (coming soon)
ab show 1234 --comments

# Show with history (coming soon)
ab show 1234 --history
```

## Global Flags

All commands support these global flags:

```bash
--org <organization>     # Override configured organization
--project <project>      # Override configured project
--config <path>          # Use custom config file
```

Example:

```bash
ab list --org myorg --project myproject
```

## Configuration File

Configuration is stored in `~/.azure-boards-cli/config.yaml`:

```yaml
organization: myorg
project: myproject
default_area_path: "myproject\\Team A"
default_iteration: "Sprint 42"
cache_ttl: 300
default_view: "assigned-to-me"
```

## Authentication Token Storage

The Personal Access Token is securely stored in `~/.azure-boards-cli/token` with restricted file permissions (owner read/write only).

## Coming Soon

The following features are planned for future releases:

- 📝 **Create Work Items**: Interactive and command-line creation
- ✏️ **Update Work Items**: Modify work item fields
- 🗑️ **Delete Work Items**: Remove work items
- 🔍 **Query Support**: Execute saved queries
- 📱 **TUI Dashboard**: Interactive terminal UI
- 💾 **Caching**: Offline support with local caching
- 📋 **Templates**: Work item templates
- 🎯 **Aliases**: Custom command aliases
- 📊 **Export/Import**: Bulk operations

## Development

### Project Structure

```
azure-boards-cli/
├── cmd/                    # Command implementations
│   ├── root.go            # Root command
│   ├── auth.go            # Authentication commands
│   ├── config.go          # Configuration commands
│   ├── list.go            # List command
│   └── show.go            # Show command
├── internal/
│   ├── api/               # Azure DevOps API client
│   │   ├── client.go      # Client wrapper
│   │   ├── workitems.go   # Work item operations
│   │   └── queries.go     # Query operations
│   ├── auth/              # Authentication
│   └── config/            # Configuration management
├── pkg/
│   └── models/            # Shared data models
├── main.go                # Entry point
├── go.mod                 # Go module definition
├── SPEC.md                # Technical specification
└── README.md              # This file
```

### Building

```bash
go build -o ab
```

### Testing

```bash
go test ./...
```

## Troubleshooting

### "not authenticated" error

Run `ab auth login` to authenticate with your PAT.

### "organization not configured" error

Run `ab config set organization <org>` to set your organization.

### "project not configured" error

Run `ab config set project <project>` to set your project.

### Invalid PAT

Make sure your PAT has the correct scopes:
- Work Items (Read, Write)

And that it hasn't expired.

## Contributing

See [SPEC.md](SPEC.md) for the full technical specification and development roadmap.

## License

MIT License
