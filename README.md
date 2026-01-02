# Azure Boards CLI (azb)

[![CI](https://github.com/SOMUCHDOG/ado-admin/actions/workflows/test.yml/badge.svg)](https://github.com/SOMUCHDOG/ado-admin/actions/workflows/test.yml)
[![Release](https://github.com/SOMUCHDOG/ado-admin/actions/workflows/release.yml/badge.svg)](https://github.com/SOMUCHDOG/ado-admin/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform Support](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](#installation)

A cross-platform command-line interface for managing Azure Boards work items.

## Features

- 🔐 **Secure Authentication**: PAT (Personal Access Token) support with secure storage
- 📋 **Work Item Management**: List, view, create, update, and delete work items
- 🔍 **Powerful Filtering**: Filter by state, assignee, type, sprint, area path, and tags
- 📝 **Interactive & CLI Modes**: Create and update work items interactively or via command-line flags
- 🏷️ **Tag Management**: Add and remove tags from work items
- 📦 **Bulk Operations**: Update and delete multiple work items at once
- 📊 **Multiple Output Formats**: Table, JSON, CSV, and IDs-only formats
- ⚙️ **Configuration Management**: Easy setup and configuration
- 🚀 **Fast & Lightweight**: Single binary, no dependencies required

## Installation

### Pre-built Binaries (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/SOMUCHDOG/ado-admin/releases).

#### macOS

```bash
# Intel
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Darwin_x86_64.tar.gz | tar xz
sudo mv azb /usr/local/bin/

# Apple Silicon
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Darwin_arm64.tar.gz | tar xz
sudo mv azb /usr/local/bin/
```

#### Linux

```bash
# AMD64
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Linux_x86_64.tar.gz | tar xz
sudo mv azb /usr/local/bin/
```

#### Windows

Download the `.zip` file for your architecture from the [Releases page](https://github.com/SOMUCHDOG/ado-admin/releases), extract it, and add to your PATH.

### From Source

```bash
git clone https://github.com/SOMUCHDOG/ado-admin
cd ado-admin
go build -o azb
```

Move the binary to your PATH:

```bash
# macOS/Linux
sudo mv azb /usr/local/bin/

# Or add to your local bin
mv azb ~/bin/
```

## Quick Start

### 1. Configure Organization and Project

```bash
azb config set organization myorg
azb config set project myproject
```

### 2. Authenticate

```bash
azb auth login
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
azb list

# List work items assigned to me
azb list --assigned-to @me

# List active bugs
azb list --type Bug --state Active

# List work items in current sprint
azb list --sprint current
```

### 4. View Work Item Details

```bash
azb show 1234
```

## Usage

### Authentication

```bash
# Login with PAT
azb auth login
azb auth login --pat <your-token>

# Check authentication status
azb auth status

# Logout
azb auth logout
```

### Configuration

```bash
# Set configuration values
azb config set organization myorg
azb config set project myproject
azb config set default_area_path "myproject\\Team A"

# Get configuration value
azb config get organization

# List all configuration
azb config list
```

### List Work Items

```bash
# Basic listing
azb list

# Filter options
azb list --state <state>              # Filter by state (Active, Resolved, Closed)
azb list --assigned-to <user>         # Filter by assignee (@me for current user)
azb list --type <type>                # Filter by work item type (Bug, Task, User Story)
azb list --sprint <sprint>            # Filter by sprint (current, @current, or sprint name)
azb list --area-path <path>           # Filter by area path
azb list --tags <tags>                # Filter by tags (comma-separated)
azb list --limit <n>                  # Limit number of results (default: 50)

# Output formats
azb list --format table               # Table format (default)
azb list --format json                # JSON format
azb list --format csv                 # CSV format
azb list --format ids                 # IDs only (for scripting)

# Examples
azb list --type Bug --assigned-to @me --state Active
azb list --sprint "Sprint 42" --format json
azb list --tags "urgent,security" --limit 20
```

### Show Work Item

```bash
# Show work item details
azb show <id>

# Show with JSON format
azb show 1234 --format json

# Show with comments (coming soon)
azb show 1234 --comments

# Show with history (coming soon)
azb show 1234 --history
```

### Update Work Item

```bash
# Update a single field
azb update 1234 --state Resolved
azb update 1234 --title "New title"
azb update 1234 --assigned-to @me
azb update 1234 --priority 1

# Update multiple fields at once
azb update 1234 --state Active --assigned-to "Jane Doe" --priority 2

# Update custom fields
azb update 1234 --field "Custom.ApplicationName=MyApp"
azb update 1234 --field "Microsoft.VSTS.Scheduling.StoryPoints=5"

# Tag operations
azb update 1234 --add-tag "urgent,bug"
azb update 1234 --remove-tag "needs-triage"
azb update 1234 --add-tag "reviewed" --remove-tag "needs-review"

# Bulk update (update multiple work items)
azb update 1234,1235,1236 --state Closed
azb update 1234,1235,1236 --add-tag "sprint-42"

# Interactive mode (prompts for each field)
azb update 1234 --interactive
azb update 1234 -i
```

**Interactive Mode Example:**
```
$ azb update 1234 -i
Interactive update for work item 1234
Leave blank to keep current value, enter new value to update

Title [Fix login bug]:
Description [Users unable to login]: Updated description
State [Active]: Resolved
Assigned To [Casey Kawamura]:
Tags [bug,urgent]:
Priority [1]: 2

Fields to update:
  System.Description = Updated description
  System.State = Resolved
  Microsoft.VSTS.Common.Priority = 2

Update work item? (y/N): y

✓ Updated work item 1234
```

### Create Work Item

```bash
# Interactive mode (will prompt for all fields)
azb create

# Command line mode with required fields
azb create --type Bug --title "Fix login issue"

# With all fields specified
azb create \
  --type "User Story" \
  --title "Implement dashboard" \
  --description "Create a new dashboard for analytics" \
  --assigned-to @me \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --priority 2 \
  --tags "frontend,dashboard"

# Quick bug creation
azb create --type Bug --title "Button not working" --assigned-to @me --priority 1

# Create task
azb create --type Task --title "Update documentation" --tags "docs"
```

**Interactive Mode:**
When you run `azb create` without flags, you'll be prompted for each field:
- Work item type (Bug, Task, User Story, Feature, Epic)
- Title (required)
- Description (optional)
- Assigned To (optional, use @me for yourself)
- Area Path (optional, uses config default)
- Iteration (optional, uses config default)
- Priority (optional, 1-4, default is 2)
- Tags (optional, comma-separated)
- Custom required fields (dynamically discovered)

### Templates

Templates allow you to save commonly used work item configurations for reuse.

**Creating a Template:**

There are two ways to create templates:

*Option 1: Initialize from example (recommended)*
```bash
# Create a template file with example fields
azb template init my-story "User Story"

# Edit the template file with your defaults
azb template edit my-story

# Or manually edit the file
# Templates are in: ~/.azure-boards-cli/templates/
```

*Option 2: Save from command-line flags*
```bash
# Save a template for bug reports
azb template save bug-report \
  --type Bug \
  --description "Template for bug reports" \
  --assigned-to @me \
  --priority 1 \
  --tags "bug,needs-triage" \
  --field "Custom.ApplicationName=MyApp"

# Save a template for user stories
azb template save user-story \
  --type "User Story" \
  --description "Template for user stories" \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --field "Microsoft.VSTS.Common.ValueArea=Business" \
  --field "Custom.ApplicationName=MyApp"
```

**Using a Template:**
```bash
# Create a work item from a template
azb create --template bug-report --title "Button not working"

# Create from template (interactive mode will use template defaults)
azb create -t user-story
```

**Managing Templates:**
```bash
# List all templates
azb template list

# Show template details
azb template show bug-report

# Show template in JSON format
azb template show bug-report --format json

# Edit a template in your editor ($EDITOR or $VISUAL)
azb template edit bug-report

# Show where templates are stored
azb template path

# Show path to a specific template
azb template path bug-report

# Delete a template
azb template delete bug-report
```

**Template Storage:**
Templates are stored as YAML files in `~/.azure-boards-cli/templates/`

You can also manually create or edit template files. See `notes/example-template.yaml` for a full example with all available fields.

**Example Template Structure:**
```yaml
name: my-template
description: Template description
type: User Story
fields:
  System.Title: Default Title
  System.Description: Default description
  System.AssignedTo: "@me"
  System.Tags: tag1,tag2
  System.AreaPath: myproject\\Team A
  System.IterationPath: myproject\\Sprint 1
  Microsoft.VSTS.Common.Priority: 2
  Microsoft.VSTS.Common.ValueArea: Business
  Microsoft.VSTS.Scheduling.StoryPoints: 0
  Microsoft.VSTS.Common.AcceptanceCriteria: |
    - [ ] Criteria 1
    - [ ] Criteria 2
  Custom.ApplicationName: MyApp

# Optional: Work item relationships
relations:
  # Link to an existing parent work item
  parentId: 12345

  # Automatically create child work items
  children:
    - title: Child Task 1
      type: Task
      description: Description for child task
      assignedTo: "@me"
      fields:
        Microsoft.VSTS.Scheduling.RemainingWork: 4

    - title: Child Task 2
      description: Another child task
      assignedTo: Casey Kawamura
```

**Creating Work Items with Relationships:**
```bash
# Create a child work item linked to a parent
azb create --type Task --title "Fix bug" --parent-id 12345

# Create from template with parent and children
# (creates parent + all children in one command)
azb create --template my-story-with-tasks

# Override parent ID from template
azb create --template my-template --parent-id 99999
```

### Query Commands

```bash
# List all saved queries
azb query list
azb query list --format json

# Show query details (including WIQL)
azb query show "Assigned to me"
azb query show "Sprint Backlog" --format json

# Run a saved query
azb query run "Assigned to me"
azb query run "My Bugs" --limit 20
azb query run "Sprint Backlog" --format json
azb query run "Active Tasks" --format ids
```

**Query List Example:**
```
$ azb query list
NAME                                                 TYPE    PATH
Shared Queries                                       Folder  Shared Queries
  Open Work                                          Query   Shared Queries/Open Work
  Blocked Work                                       Query   Shared Queries/Blocked Work
  Active Projects                                    Query   Shared Queries/Active Projects
My Queries                                           Folder  My Queries
  Followed work items                                Query   My Queries/Followed work items
  Assigned to me                                     Query   My Queries/Assigned to me
```

**Query Show Example:**
```
$ azb query show "Assigned to me"
Name: Assigned to me
Path: My Queries/Assigned to me
ID: cddeffc6-ad80-4a71-a6ae-0df0a6be03ec
Type: Personal

WIQL:
select [System.Id], [System.WorkItemType], [System.Title], [System.State]
from WorkItems where [System.AssignedTo] = @me order by [System.ChangedDate] desc
```

**Query Run Example:**
```
$ azb query run "Assigned to me" --limit 5
ID       Title                                              Type            State           Assigned To
------------------------------------------------------------------------------------------------------------------------
73807    Fix login bug                                      Bug             Active          Casey Kawamura
73504    Add new dashboard feature                          User Story      New             Casey Kawamura
73667    Update documentation                               Task            Active          Casey Kawamura
```

### Delete Work Item

```bash
# Delete a single work item (with confirmation)
azb delete 1234

# Delete multiple work items
azb delete 1234,1235,1236

# Delete without confirmation (for scripting)
azb delete 1234 --force
azb delete 1234,1235,1236 -f
```

**Confirmation Prompt:**
```
$ azb delete 1234
The following work items will be deleted:
  - ID 1234: Fix login bug

Are you sure you want to delete these work items? This cannot be undone. (y/N): y

✓ Deleted work item 1234

Summary: 1 deleted, 0 failed
```

### Inspecting Work Item Types

Use the inspect command to discover required fields for your organization:

```bash
# Inspect a work item type to see all fields
azb inspect "User Story"

# See what fields are required for bugs
azb inspect Bug
```

This helps you understand what custom fields your organization requires, which you can then include in templates.

## Global Flags

All commands support these global flags:

```bash
--org <organization>     # Override configured organization
--project <project>      # Override configured project
--config <path>          # Use custom config file
```

Example:

```bash
azb list --org myorg --project myproject
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

- 📱 **TUI Dashboard**: Interactive terminal UI
- 💾 **Caching**: Offline support with local caching
- 🎯 **Aliases**: Custom command aliases
- 📊 **Export/Import**: Bulk operations
- 🔍 **Advanced Query Features**: Create and manage queries via CLI

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

Run `azb auth login` to authenticate with your PAT.

### "organization not configured" error

Run `azb config set organization <org>` to set your organization.

### "project not configured" error

Run `azb config set project <project>` to set your project.

### Invalid PAT

Make sure your PAT has the correct scopes:
- Work Items (Read, Write)

And that it hasn't expired.

## Contributing

Contributions are welcome! This project uses automated semantic versioning based on commit messages.

### Commit Message Format

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification. Each commit message should be structured as follows:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` A new feature (triggers minor version bump)
- `fix:` A bug fix (triggers patch version bump)
- `perf:` Performance improvement (triggers patch version bump)
- `refactor:` Code refactoring (triggers patch version bump)
- `docs:` Documentation only changes (no version bump)
- `test:` Adding or updating tests (no version bump)
- `chore:` Maintenance tasks (no version bump)
- `ci:` CI/CD changes (no version bump)

**Breaking Changes:**
Add `BREAKING CHANGE:` in the footer or `!` after the type to trigger a major version bump:
```
feat!: remove support for legacy API
```

**Examples:**
```bash
feat(auth): add support for OAuth authentication
fix(list): handle empty work item response correctly
docs: update installation instructions
chore(deps): update dependencies
```

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Write tests for your changes
5. Run tests: `go test ./...`
6. Run linter: `golangci-lint run`
7. Commit with conventional commit message
8. Push to your fork
9. Open a Pull Request

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

- **CI Workflow**: Runs on every push and PR
  - Linting with golangci-lint
  - Testing with `go test`
  - Build verification for all platforms

- **Release Workflow**: Runs on push to main branch
  - Analyzes commit messages for semantic versioning
  - Creates git tags and GitHub releases automatically
  - Builds binaries for multiple platforms using GoReleaser
  - Generates changelog from commit messages

Releases are fully automated - just merge to main with proper commit messages!

For more details, see [SPEC.md](SPEC.md) for the full technical specification and development roadmap.

