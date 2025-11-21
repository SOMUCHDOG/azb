# Azure Boards CLI - Technical Specification

## Overview

A cross-platform command-line interface for managing Azure Boards work items, providing both a Terminal UI (TUI) dashboard for interactive work and traditional CLI commands for automation and scripting.

## Project Goals

- **Efficiency**: Enable developers to manage Azure Boards work items without leaving the terminal
- **Cross-platform**: Support macOS, Linux, and Windows
- **Dual Interface**: Hybrid approach with TUI dashboard and traditional CLI commands
- **Performance**: Fast startup and responsive UI
- **Offline Support**: Cache work items for offline viewing and queuing operations

## Technical Stack

### Language & Runtime
- **Go 1.21+**: Single binary distribution, excellent cross-platform support, fast performance

### Key Libraries
- **Azure DevOps SDK**: [github.com/microsoft/azure-devops-go-api](https://github.com/microsoft/azure-devops-go-api)
- **CLI Framework**: [cobra](https://github.com/spf13/cobra) - Command structure and parsing
- **TUI Framework**: [bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI
- **UI Components**: [bubbles](https://github.com/charmbracelet/bubbles) - Reusable TUI components
- **Styling**: [lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- **Configuration**: [viper](https://github.com/spf13/viper) - Config file management
- **Tables**: [pterm](https://github.com/pterm/pterm) - Pretty terminal tables for CLI output

### Build & Distribution
- **GoReleaser**: Automated multi-platform builds
- **GitHub Actions**: CI/CD pipeline
- **Package Managers**: Homebrew (macOS), Snap (Linux), Chocolatey (Windows)

## Architecture

```
azure-boards-cli/
├── cmd/
│   ├── root.go              # Root command & global flags
│   ├── dashboard.go         # TUI dashboard command
│   ├── list.go              # List work items (CLI)
│   ├── show.go              # Show work item details (CLI)
│   ├── create.go            # Create work item (CLI)
│   ├── update.go            # Update work item (CLI)
│   ├── delete.go            # Delete work item (CLI)
│   ├── query.go             # Execute saved queries (CLI)
│   ├── config.go            # Configuration management
│   └── auth.go              # Authentication commands
├── internal/
│   ├── api/
│   │   ├── client.go        # Azure DevOps API client
│   │   ├── workitems.go     # Work item operations
│   │   └── queries.go       # Query operations
│   ├── tui/
│   │   ├── dashboard.go     # Main dashboard model
│   │   ├── list_view.go     # Work item list component
│   │   ├── detail_view.go   # Work item detail component
│   │   ├── filter_view.go   # Filter/search component
│   │   └── styles.go        # UI styling
│   ├── cache/
│   │   └── cache.go         # Local caching layer
│   ├── config/
│   │   └── config.go        # Configuration handling
│   └── auth/
│       └── auth.go          # Authentication & token management
├── pkg/
│   └── models/
│       └── workitem.go      # Shared data models
└── main.go
```

## Features

### 1. Authentication & Configuration

#### Authentication Methods
- Personal Access Token (PAT)
- Azure CLI authentication (reuse `az` credentials)
- Device code flow for interactive login

#### Configuration
```yaml
# ~/.azure-boards-cli/config.yaml
organization: "myorg"
project: "myproject"
default_area_path: "myproject\\Team A"
default_iteration: "Sprint 42"
cache_ttl: 300  # seconds
default_view: "assigned-to-me"
```

#### Commands
```bash
# Configure organization and project
azb config set organization <org>
azb config set project <project>

# Authenticate
azb auth login
azb auth login --pat <token>
azb auth status
azb auth logout
```

### 2. TUI Dashboard

Launch with: `azb dashboard` or `azb`

#### Layout
```
┌─ Azure Boards - myproject ────────────────────────────────────┐
│ Filter: [assigned to me ▼] Sprint: [Current ▼]  [?] Help     │
├───────────────────────────────────────────────────────────────┤
│ ID     │ Title                      │ State      │ Assigned   │
├───────────────────────────────────────────────────────────────┤
│ > 1234 │ Fix login bug              │ Active     │ @me        │
│   1235 │ Add new dashboard          │ New        │ @me        │
│   1236 │ Refactor API layer         │ In Review  │ Jane       │
│                                                                │
├───────────────────────────────────────────────────────────────┤
│ [Details] [Comments] [History]                                │
├───────────────────────────────────────────────────────────────┤
│ #1234 - Fix login bug                                         │
│ Type: Bug | State: Active | Priority: 1                       │
│                                                                │
│ Users unable to login when using SSO. Error occurs in the     │
│ authentication middleware after recent deployment.            │
│                                                                │
│ Assigned to: Casey Smith                                      │
│ Created: 2025-11-10 | Updated: 2025-11-15                     │
└───────────────────────────────────────────────────────────────┘
```

#### Features
- **Navigation**: Arrow keys, Vim-style (hjkl), PageUp/Down
- **Selection**: Enter to select, Space to multi-select
- **Actions**:
  - `n` - New work item
  - `e` - Edit selected work item
  - `d` - Delete selected work item(s)
  - `/` - Search/filter
  - `r` - Refresh
  - `s` - Change state (Active, Resolved, Closed, etc.)
  - `a` - Assign to user
  - `t` - Add tags
  - `?` - Help
  - `q` - Quit

#### Views & Filters
- Assigned to me
- Created by me
- Recent items
- By sprint
- By area path
- Custom queries
- Search by ID, title, description

#### Real-time Updates
- Auto-refresh every N seconds (configurable)
- Visual indicator for modified items
- Conflict detection for concurrent edits

### 3. CLI Commands

#### List Work Items
```bash
# List assigned to me
azb list

# List with filters
azb list --state Active --assigned-to @me
azb list --type Bug --sprint "Sprint 42"
azb list --area-path "myproject\\Team A"
azb list --tags "urgent,security"

# Output formats
azb list --format table  # default
azb list --format json
azb list --format csv
azb list --format ids    # just IDs for scripting

# Limit results
azb list --limit 50
```

#### Show Work Item
```bash
# Show details
azb show 1234

# Show with comments
azb show 1234 --comments

# Show with history
azb show 1234 --history

# Output format
azb show 1234 --format json
```

#### Create Work Item
```bash
# Interactive mode
azb create

# Command line mode
azb create --type Bug --title "Fix login issue" --assigned-to @me

# From template
azb create --template bug-template.yaml

# With full details
azb create \
  --type "User Story" \
  --title "Implement dashboard" \
  --description "Create a new dashboard for analytics" \
  --assigned-to "john@example.com" \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --priority 2 \
  --tags "frontend,dashboard"

# From file
azb create --from-file workitem.json
```

#### Update Work Item
```bash
# Update specific fields
azb update 1234 --state Active
azb update 1234 --assigned-to "jane@example.com"
azb update 1234 --title "New title"
azb update 1234 --add-tag "urgent"
azb update 1234 --remove-tag "low-priority"

# Interactive edit
azb update 1234

# Bulk update
azb update 1234,1235,1236 --state Resolved
```

#### Delete Work Item
```bash
# Delete single item
azb delete 1234

# Delete multiple
azb delete 1234,1235,1236

# With confirmation
azb delete 1234 --confirm

# Skip confirmation
azb delete 1234 --force
```

#### Queries
```bash
# List saved queries
azb query list

# Execute saved query
azb query run "My Bugs"

# Execute shared query
azb query run "Shared Queries/Team Backlog"

# Create query (opens editor)
azb query create

# Delete query
azb query delete "My Query"
```

#### Batch Operations
```bash
# Pipe work item IDs
azb list --type Bug --format ids | xargs azb update --state Resolved

# From file
cat work-items.txt | xargs azb delete --force
```

### 4. Work Item Types

Support all standard Azure Boards work item types:
- Epic
- Feature
- User Story
- Task
- Bug
- Issue
- Test Case

### 5. Advanced Features

#### Templates
Save commonly used work item configurations:

```yaml
# ~/.azure-boards-cli/templates/bug.yaml
type: Bug
assigned_to: "@me"
area_path: "myproject\\Team A"
iteration: "current"
priority: 2
tags:
  - needs-triage
fields:
  System.Description: |
    ## Steps to Reproduce
    1.

    ## Expected Behavior


    ## Actual Behavior

```

#### Aliases
```bash
# ~/.azure-boards-cli/aliases.yaml
aliases:
  my-bugs: list --type Bug --assigned-to @me --state Active
  sprint-tasks: list --type Task --iteration current
  close: update --state Closed --force

# Usage
azb my-bugs
azb sprint-tasks
azb close 1234
```

#### Export & Import
```bash
# Export to various formats
azb export --query "My Bugs" --format csv > bugs.csv
azb export --query "Sprint Backlog" --format json > backlog.json

# Import work items
azb import workitems.json
```

### 6. Caching & Offline Support

#### Cache Strategy
- Cache work items locally with TTL
- Queue operations when offline
- Sync when connection restored
- Conflict resolution on sync

#### Cache Commands
```bash
# Clear cache
azb cache clear

# Refresh cache
azb cache refresh

# Show cache status
azb cache status
```

## User Experience

### First Run Experience
```bash
$ azb
Welcome to Azure Boards CLI!

Let's get you set up.

Organization: myorg
Project: myproject

How would you like to authenticate?
  1. Personal Access Token (PAT)
  2. Use Azure CLI credentials
  3. Device code flow

Choice: 1
PAT: ****************************************************

✓ Authentication successful
✓ Configuration saved to ~/.azure-boards-cli/config.yaml

Run 'azb' to launch the dashboard or 'azb --help' for available commands.
```

### Error Handling
- Clear, actionable error messages
- Suggestions for common issues
- Automatic retry with exponential backoff
- Offline mode fallback

### Performance
- Startup time: < 100ms
- TUI response time: < 50ms
- API calls: Parallel requests where possible
- Pagination for large result sets

## Testing Strategy

### Unit Tests
- API client functions
- Data models
- Cache operations
- Config management

### Integration Tests
- Azure DevOps API integration
- Authentication flows
- CRUD operations

### E2E Tests
- TUI interactions (using automated testing framework)
- CLI command execution
- Multi-platform builds

## Security

### Credentials Storage
- Store tokens in OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- Never log sensitive data
- Secure file permissions on config files

### API Security
- HTTPS only
- Token refresh handling
- Rate limiting compliance

## Documentation

### Required Documentation
- README.md: Quick start, installation
- USAGE.md: Comprehensive command reference
- CONTRIBUTING.md: Development setup, guidelines
- API.md: Internal API documentation
- CHANGELOG.md: Version history

### Interactive Help
```bash
azb --help
azb <command> --help
azb dashboard  # Press '?' for help
```

## Release & Distribution

### Versioning
- Semantic versioning (MAJOR.MINOR.PATCH)
- Git tags for releases
- Automated changelog generation

### Distribution Channels
- GitHub Releases (binaries)
- Homebrew (macOS)
- Snap Store (Linux)
- Chocolatey (Windows)
- Direct download from releases page

### Build Matrix
- macOS (amd64, arm64)
- Linux (amd64, arm64, 386)
- Windows (amd64, 386)

## Future Enhancements (Post-MVP)

### Phase 2
- Sprint management features (capacity, burndown charts)
- Board view visualization (Kanban)
- Pull request integration
- Git branch association
- Time tracking

### Phase 3
- Plugin system
- Custom field support
- Advanced reporting
- Team dashboards
- AI-powered suggestions (e.g., similar work items)

### Phase 4
- Multi-project support
- Cross-project queries
- Analytics and metrics
- Export to various project management tools

## Success Metrics

- Installation count
- Active users
- Command usage analytics (opt-in)
- GitHub stars/forks
- Issue resolution time
- User feedback and feature requests

## Development Timeline (Estimated)

### Week 1-2: Foundation
- Project setup, dependencies
- Authentication & configuration
- Basic API client

### Week 3-4: Core CLI
- List, show, create, update, delete commands
- Query support
- Output formatting

### Week 5-6: TUI Dashboard
- Basic dashboard layout
- Navigation and selection
- Work item detail view

### Week 7-8: Polish & Testing
- Error handling
- Caching
- Unit and integration tests
- Documentation

### Week 9-10: Release Preparation
- Cross-platform builds
- Distribution setup
- Beta testing
- Release v1.0.0

## Open Questions

1. Should we support Azure DevOps Server (on-premises) or only cloud?
2. What level of customization for work item fields?
3. Should we support WIQL (Work Item Query Language) directly?
4. Multi-organization support in single config?
5. Support for attachments and rich text in descriptions?

## Appendix

### Similar Projects (for reference)
- GitHub CLI (`gh`)
- Azure CLI (`az`)
- Kubernetes CLI (`kubectl`)
- LazyGit (TUI example)

### Resources
- [Azure DevOps REST API](https://learn.microsoft.com/en-us/rest/api/azure/devops/)
- [Azure DevOps Go SDK](https://github.com/microsoft/azure-devops-go-api)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
