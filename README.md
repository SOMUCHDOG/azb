# Azure Boards CLI

A cross-platform command-line interface for managing Azure Boards work items.

## Features

- 🔐 **Secure Authentication**: PAT (Personal Access Token) support with secure storage
- 📋 **Work Item Management**: List, view, create, and update work items
- 🔍 **Powerful Filtering**: Filter by state, assignee, type, sprint, area path, and tags
- 📝 **Interactive & CLI Modes**: Create and update work items interactively or via command-line flags
- 🏷️ **Tag Management**: Add and remove tags from work items
- 📦 **Bulk Operations**: Update multiple work items at once
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

### Update Work Item

```bash
# Update a single field
ab update 1234 --state Resolved
ab update 1234 --title "New title"
ab update 1234 --assigned-to @me
ab update 1234 --priority 1

# Update multiple fields at once
ab update 1234 --state Active --assigned-to "Jane Doe" --priority 2

# Update custom fields
ab update 1234 --field "Custom.ApplicationName=MyApp"
ab update 1234 --field "Microsoft.VSTS.Scheduling.StoryPoints=5"

# Tag operations
ab update 1234 --add-tag "urgent,bug"
ab update 1234 --remove-tag "needs-triage"
ab update 1234 --add-tag "reviewed" --remove-tag "needs-review"

# Bulk update (update multiple work items)
ab update 1234,1235,1236 --state Closed
ab update 1234,1235,1236 --add-tag "sprint-42"

# Interactive mode (prompts for each field)
ab update 1234 --interactive
ab update 1234 -i
```

**Interactive Mode Example:**
```
$ ab update 1234 -i
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
ab create

# Command line mode with required fields
ab create --type Bug --title "Fix login issue"

# With all fields specified
ab create \
  --type "User Story" \
  --title "Implement dashboard" \
  --description "Create a new dashboard for analytics" \
  --assigned-to @me \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --priority 2 \
  --tags "frontend,dashboard"

# Quick bug creation
ab create --type Bug --title "Button not working" --assigned-to @me --priority 1

# Create task
ab create --type Task --title "Update documentation" --tags "docs"
```

**Interactive Mode:**
When you run `ab create` without flags, you'll be prompted for each field:
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
ab template init my-story "User Story"

# Edit the template file with your defaults
ab template edit my-story

# Or manually edit the file
# Templates are in: ~/.azure-boards-cli/templates/
```

*Option 2: Save from command-line flags*
```bash
# Save a template for bug reports
ab template save bug-report \
  --type Bug \
  --description "Template for bug reports" \
  --assigned-to @me \
  --priority 1 \
  --tags "bug,needs-triage" \
  --field "Custom.ApplicationName=MyApp"

# Save a template for user stories
ab template save user-story \
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
ab create --template bug-report --title "Button not working"

# Create from template (interactive mode will use template defaults)
ab create -t user-story
```

**Managing Templates:**
```bash
# List all templates
ab template list

# Show template details
ab template show bug-report

# Show template in JSON format
ab template show bug-report --format json

# Edit a template in your editor ($EDITOR or $VISUAL)
ab template edit bug-report

# Show where templates are stored
ab template path

# Show path to a specific template
ab template path bug-report

# Delete a template
ab template delete bug-report
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
ab create --type Task --title "Fix bug" --parent-id 12345

# Create from template with parent and children
# (creates parent + all children in one command)
ab create --template my-story-with-tasks

# Override parent ID from template
ab create --template my-template --parent-id 99999
```

### Inspecting Work Item Types

Use the inspect command to discover required fields for your organization:

```bash
# Inspect a work item type to see all fields
ab inspect "User Story"

# See what fields are required for bugs
ab inspect Bug
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

- 🗑️ **Delete Work Items**: Remove work items
- 🔍 **Query Support**: Execute saved queries
- 📱 **TUI Dashboard**: Interactive terminal UI
- 💾 **Caching**: Offline support with local caching
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
