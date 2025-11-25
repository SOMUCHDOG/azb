# Azure Boards CLI (azb) - User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [CLI Commands](#cli-commands)
4. [TUI Dashboard](#tui-dashboard)
5. [Advanced Features](#advanced-features)
6. [Tips and Best Practices](#tips-and-best-practices)
7. [Troubleshooting](#troubleshooting)

---

## Introduction

Azure Boards CLI (azb) is a cross-platform command-line interface for managing Azure Boards work items. It provides two ways to interact with Azure Boards:

- **CLI Mode**: Traditional command-line interface for automation and scripting
- **TUI Dashboard**: Interactive terminal UI for browsing and managing work items

### Key Features

- Secure authentication with Personal Access Token (PAT) support
- Work item management (list, view, create, update, delete)
- Powerful filtering by state, assignee, type, sprint, area path, and tags
- Interactive and CLI modes for creating and updating work items
- Tag management (add/remove tags)
- Bulk operations for updating and deleting multiple work items
- Multiple output formats (table, JSON, CSV, IDs-only)
- Template system for reusable work item configurations
- Query execution for saved Azure DevOps queries
- Fast and lightweight single binary with no dependencies

---

## Getting Started

### Installation

#### macOS

**Intel:**
```bash
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Darwin_x86_64.tar.gz | tar xz
sudo mv azb /usr/local/bin/
```

**Apple Silicon:**
```bash
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Darwin_arm64.tar.gz | tar xz
sudo mv azb /usr/local/bin/
```

#### Linux

```bash
curl -L https://github.com/SOMUCHDOG/ado-admin/releases/latest/download/azb_Linux_x86_64.tar.gz | tar xz
sudo mv azb /usr/local/bin/
```

#### Windows

Download the `.zip` file for your architecture from the [Releases page](https://github.com/SOMUCHDOG/ado-admin/releases), extract it, and add to your PATH.

#### From Source

```bash
git clone https://github.com/SOMUCHDOG/ado-admin
cd ado-admin
go build -o azb
sudo mv azb /usr/local/bin/
```

### Initial Configuration

#### 1. Set Organization and Project

```bash
azb config set organization myorg
azb config set project myproject
```

#### 2. Authenticate

```bash
azb auth login
```

You'll be prompted to enter your Personal Access Token (PAT).

**Creating a Personal Access Token:**

1. Go to `https://dev.azure.com/{org}/_usersSettings/tokens`
2. Click "New Token"
3. Select scopes: `Work Items (Read, Write)`
4. Copy the generated token
5. Paste it when prompted by `azb auth login`

#### 3. Verify Authentication

```bash
azb auth status
```

### Configuration File

Your configuration is stored in `~/.azure-boards-cli/config.yaml`:

```yaml
organization: myorg
project: myproject
default_area_path: "myproject\\Team A"
default_iteration: "Sprint 42"
cache_ttl: 300
default_view: "assigned-to-me"
```

You can edit this file directly or use `azb config set` commands.

---

## CLI Commands

### Work Item Management

#### List Work Items

Display work items with various filters:

```bash
# List all work items
azb list

# List work items assigned to you
azb list --assigned-to @me

# List active bugs
azb list --type Bug --state Active

# List work items in current sprint
azb list --sprint current

# List with multiple filters
azb list --type "User Story" --assigned-to @me --state Active --limit 20

# List with tags
azb list --tags "urgent,security"

# List by area path
azb list --area-path "myproject\\Team A"
```

**Output Formats:**

```bash
azb list --format table    # Default: formatted table
azb list --format json     # JSON output
azb list --format csv      # CSV format
azb list --format ids      # IDs only (great for scripting)
```

**Common Filter Options:**

| Option | Description | Example |
|--------|-------------|---------|
| `--state` | Filter by state | `--state Active` |
| `--assigned-to` | Filter by assignee | `--assigned-to @me` |
| `--type` | Filter by work item type | `--type Bug` |
| `--sprint` | Filter by sprint | `--sprint current` |
| `--area-path` | Filter by area path | `--area-path "myproject\\Team A"` |
| `--tags` | Filter by tags (comma-separated) | `--tags "urgent,bug"` |
| `--limit` | Limit number of results | `--limit 50` |

#### View Work Item Details

```bash
# Show work item details
azb show 1234

# Show with JSON format
azb show 1234 --format json
```

#### Create Work Item

**Interactive Mode** (recommended for first-time use):

```bash
azb create
```

This will prompt you for:
- Work item type (Bug, Task, User Story, Feature, Epic)
- Title (required)
- Description (optional)
- Assigned To (optional, use @me for yourself)
- Area Path (optional)
- Iteration (optional)
- Priority (1-4, default: 2)
- Tags (comma-separated)
- Any custom required fields for your organization

**Command-Line Mode:**

```bash
# Quick bug creation
azb create --type Bug --title "Button not working" --assigned-to @me --priority 1

# Create with full details
azb create \
  --type "User Story" \
  --title "Implement dashboard" \
  --description "Create a new dashboard for analytics" \
  --assigned-to @me \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --priority 2 \
  --tags "frontend,dashboard"

# Create from template
azb create --template my-template --title "New work item"
```

**Create with Parent Relationship:**

```bash
# Create a child work item
azb create --type Task --title "Fix bug" --parent-id 12345
```

#### Update Work Item

**Single Field Updates:**

```bash
azb update 1234 --state Resolved
azb update 1234 --title "New title"
azb update 1234 --assigned-to @me
azb update 1234 --priority 1
```

**Multiple Fields:**

```bash
azb update 1234 --state Active --assigned-to "Jane Doe" --priority 2
```

**Custom Fields:**

```bash
azb update 1234 --field "Custom.ApplicationName=MyApp"
azb update 1234 --field "Microsoft.VSTS.Scheduling.StoryPoints=5"
```

**Tag Operations:**

```bash
azb update 1234 --add-tag "urgent,bug"
azb update 1234 --remove-tag "needs-triage"
azb update 1234 --add-tag "reviewed" --remove-tag "needs-review"
```

**Bulk Updates:**

```bash
# Update multiple work items at once
azb update 1234,1235,1236 --state Closed
azb update 1234,1235,1236 --add-tag "sprint-42"
```

**Interactive Mode:**

```bash
azb update 1234 --interactive
# or
azb update 1234 -i
```

Interactive mode example:
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

#### Delete Work Item

```bash
# Delete a single work item (with confirmation prompt)
azb delete 1234

# Delete multiple work items
azb delete 1234,1235,1236

# Skip confirmation (for scripting)
azb delete 1234 --force
azb delete 1234,1235,1236 -f
```

### Templates

Templates allow you to save commonly used work item configurations for quick reuse.

#### Create a Template

**Option 1: Initialize from example**

```bash
# Create a template file with example fields
azb template init my-story "User Story"

# Edit the template
azb template edit my-story
```

**Option 2: Save from command-line flags**

```bash
# Save a bug report template
azb template save bug-report \
  --type Bug \
  --description "Template for bug reports" \
  --assigned-to @me \
  --priority 1 \
  --tags "bug,needs-triage" \
  --field "Custom.ApplicationName=MyApp"

# Save a user story template
azb template save user-story \
  --type "User Story" \
  --description "Template for user stories" \
  --area-path "myproject\\Team A" \
  --iteration "Sprint 42" \
  --field "Microsoft.VSTS.Common.ValueArea=Business"
```

#### Use a Template

```bash
# Create a work item from a template
azb create --template bug-report --title "Button not working"

# Interactive mode with template defaults
azb create -t user-story
```

#### Manage Templates

```bash
# List all templates
azb template list

# Show template details
azb template show bug-report

# Show in JSON format
azb template show bug-report --format json

# Edit a template
azb template edit bug-report

# Show template storage location
azb template path

# Show specific template path
azb template path bug-report

# Delete a template
azb template delete bug-report
```

#### Template File Structure

Templates are stored as YAML files in `~/.azure-boards-cli/templates/`

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
      type: Task
      description: Another child task
      assignedTo: Casey Kawamura
```

### Queries

Execute and manage saved Azure DevOps queries.

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

**Example Output:**

```
$ azb query list
NAME                                                 TYPE    PATH
Shared Queries                                       Folder  Shared Queries
  Open Work                                          Query   Shared Queries/Open Work
  Blocked Work                                       Query   Shared Queries/Blocked Work
My Queries                                           Folder  My Queries
  Followed work items                                Query   My Queries/Followed work items
  Assigned to me                                     Query   My Queries/Assigned to me

$ azb query show "Assigned to me"
Name: Assigned to me
Path: My Queries/Assigned to me
ID: cddeffc6-ad80-4a71-a6ae-0df0a6be03ec
Type: Personal

WIQL:
select [System.Id], [System.WorkItemType], [System.Title], [System.State]
from WorkItems where [System.AssignedTo] = @me order by [System.ChangedDate] desc

$ azb query run "Assigned to me" --limit 5
ID       Title                                              Type            State           Assigned To
------------------------------------------------------------------------------------------------------------------------
73807    Fix login bug                                      Bug             Active          Casey Kawamura
73504    Add new dashboard feature                          User Story      New             Casey Kawamura
73667    Update documentation                               Task            Active          Casey Kawamura
```

### Work Item Types Inspection

Use the inspect command to discover required fields for your organization:

```bash
# Inspect a work item type to see all fields
azb inspect "User Story"

# See what fields are required for bugs
azb inspect Bug
```

This is helpful for understanding custom fields your organization requires, which you can then include in templates.

### Configuration Management

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

### Authentication Management

```bash
# Login with PAT
azb auth login
azb auth login --pat <your-token>

# Check authentication status
azb auth status

# Logout (removes stored token)
azb auth logout
```

### Global Flags

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

---

## TUI Dashboard

The TUI Dashboard provides an interactive terminal interface for managing work items. Launch it with:

```bash
azb dashboard
# or simply
azb
```

### Dashboard Layout

```
┌─ Azure Boards - myproject ────────────────────────────────────┐
│ [Queries] [Work Items] [Templates] [Pipelines] [Agents]       │
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

### Navigation

#### Global Navigation

| Key | Action |
|-----|--------|
| `?` | Show help overlay |
| `q` or `Ctrl+C` | Quit dashboard |
| `Tab` | Switch to next tab |
| `Shift+Tab` | Switch to previous tab |
| `r` | Refresh current view |
| `↑/↓` or `j/k` | Navigate up/down in lists |
| `/` | Start filtering/search |
| `Esc` | Cancel current action |

### Tabs

#### 1. Queries Tab

Browse and execute saved Azure DevOps queries.

**Keybindings:**

| Key | Action |
|-----|--------|
| `Enter` | Execute selected query or expand/collapse folder |
| `E` | Expand all folders |
| `C` | Collapse all folders |

**Features:**
- View both personal and shared queries
- Folder navigation
- Execute queries to load work items in Work Items tab

#### 2. Work Items Tab

Manage work items with full CRUD operations.

**Keybindings:**

| Key | Action |
|-----|--------|
| `Enter` | Toggle details panel for selected item |
| `n` | Create new work item |
| `e` | Edit work item in $EDITOR |
| `w` | Download work item as YAML template |
| `d` | Delete work item (with confirmation) |
| `s` | Change work item state |
| `a` | Assign work item to user |
| `t` | Add tags to work item |

**Features:**

- **Toggle Details**: Press `Enter` to show/hide detailed view of selected work item
- **Create New**: Press `n` to create a new work item (interactive prompts)
- **Edit in Editor**: Press `e` to open the work item in your `$EDITOR` as YAML
  - The dashboard suspends while your editor is open
  - Modify the YAML, save, and close to update the work item
  - Invalid YAML will show an error notification
- **Download as Template**: Press `w` to save the work item as a reusable template
  - Templates are saved to `~/.azure-boards-cli/templates/`
  - Filename format: `workitem-{id}-{sanitized-title}.yaml`
- **Delete with Children**: Press `d` to delete a work item
  - Shows confirmation dialog with child count
  - Deletes all child work items first, then parent
  - Parent work item (if exists) remains unchanged
- **Change State**: Press `s` to change work item state (Active, Resolved, Closed, etc.)
- **Assign**: Press `a` to assign to a user
- **Add Tags**: Press `t` to add tags

**Filtering Work Items:**

Press `/` to enter filter mode. You can filter by:
- ID
- Title
- State
- Assigned user
- Tags

#### 3. Templates Tab

Manage work item templates.

**Keybindings:**

| Key | Action |
|-----|--------|
| `c` | Copy template with new name |
| `n` | Create new template |
| `f` | Create new folder |
| `e` | Edit template in $EDITOR |
| `d` | Delete template (with confirmation) |

**Features:**
- Browse template library
- Edit templates in your preferred editor
- Create folders to organize templates
- Duplicate templates for variations

#### 4. Pipelines Tab

View and manage Azure Pipelines (coming soon).

#### 5. Agents Tab

Manage build agents (coming soon).

### Customizing Keybindings

You can customize all keybindings by editing `~/.azure-boards-cli/keybinds.yaml`:

```yaml
# Global actions
global:
  quit: ["q", "ctrl+c"]
  help: ["?"]
  next_tab: ["tab"]
  prev_tab: ["shift+tab"]
  refresh: ["r"]

# Queries tab
queries:
  execute: ["enter"]
  expand_all: ["E"]
  collapse_all: ["C"]

# Work items tab
work_items:
  details: ["enter"]
  download: ["w"]
  edit: ["e"]
  delete: ["d"]
  create: ["n"]
  change_state: ["s"]
  assign: ["a"]
  add_tags: ["t"]

# Templates tab
templates:
  copy: ["c"]
  new_template: ["n"]
  new_folder: ["f"]
  edit: ["e"]
  delete: ["d"]
```

**Valid key formats:**
- Single keys: `"a"`, `"b"`, `"1"`, `"2"`
- Modified keys: `"ctrl+c"`, `"alt+enter"`, `"shift+tab"`
- Special keys: `"enter"`, `"esc"`, `"space"`, `"backspace"`, `"delete"`
- Arrow keys: `"up"`, `"down"`, `"left"`, `"right"`
- Function keys: `"f1"`, `"f2"`, etc.
- Multiple keys per action: `["q", "ctrl+c"]`

Changes take effect on next dashboard launch.

### Help System

Press `?` at any time to show context-aware help. The help overlay displays:
- Global keybindings available on all tabs
- Tab-specific keybindings for the current tab
- Descriptions of each action

Press `?` again to close the help overlay.

### Notifications

The dashboard shows notifications for:
- **Success**: Work item created, updated, or deleted
- **Errors**: API failures, validation errors, network issues
- **Information**: Action results, status updates

Notifications appear at the bottom of the screen and auto-dismiss after a few seconds.

### Confirmation Dialogs

Destructive operations (like deleting work items) show confirmation dialogs:

```
Delete work item #1234: 'Fix login bug' and its 3 child task(s)?
Press 'y' to confirm, 'n' or 'Esc' to cancel
```

Press `y` to proceed or `n`/`Esc` to cancel.

---

## Advanced Features

### Bulk Operations with Scripting

Combine CLI commands for powerful batch operations:

```bash
# Update all active bugs to resolved
azb list --type Bug --state Active --format ids | xargs -I {} azb update {} --state Resolved

# Delete all closed work items from last sprint
azb list --sprint "Sprint 41" --state Closed --format ids | xargs azb delete --force

# Add a tag to all user stories in current sprint
azb list --type "User Story" --sprint current --format ids | xargs -I {} azb update {} --add-tag "sprint-review"

# Export active bugs to CSV
azb list --type Bug --state Active --format csv > active-bugs.csv

# Create multiple work items from a template
for i in {1..5}; do
  azb create --template task-template --title "Task $i"
done
```

### Working with Custom Fields

Many organizations have custom fields. Use the `--field` flag to work with them:

```bash
# Set custom field during creation
azb create --type Bug --title "Bug title" \
  --field "Custom.ApplicationName=MyApp" \
  --field "Custom.Environment=Production"

# Update custom fields
azb update 1234 --field "Custom.SeverityLevel=Critical"

# Inspect work item type to see all custom fields
azb inspect Bug
```

### Template Best Practices

**1. Create templates for common scenarios:**

```bash
# Bug report template
azb template save bug-template \
  --type Bug \
  --priority 1 \
  --tags "needs-triage" \
  --field "Custom.ApplicationName=MyApp"

# Sprint task template
azb template save sprint-task \
  --type Task \
  --iteration "current" \
  --assigned-to @me \
  --field "Microsoft.VSTS.Scheduling.RemainingWork=4"
```

**2. Use templates with parent-child relationships:**

```yaml
# Story with predefined tasks
name: user-story-with-tasks
type: User Story
fields:
  System.Title: User Story Title
  System.AreaPath: myproject\\Team A
  Microsoft.VSTS.Common.Priority: 2

relations:
  children:
    - title: Design
      type: Task
      assignedTo: "@me"
    - title: Implementation
      type: Task
      assignedTo: "@me"
    - title: Testing
      type: Task
      assignedTo: "@me"
    - title: Documentation
      type: Task
      assignedTo: "@me"
```

**3. Organize templates in folders:**

Templates are stored in `~/.azure-boards-cli/templates/`. Create subdirectories:

```
~/.azure-boards-cli/templates/
├── bugs/
│   ├── production-bug.yaml
│   └── ui-bug.yaml
├── stories/
│   ├── feature-story.yaml
│   └── spike-story.yaml
└── tasks/
    ├── dev-task.yaml
    └── test-task.yaml
```

### Environment Variables

Set default values using environment variables:

```bash
export AZURE_BOARDS_ORG="myorg"
export AZURE_BOARDS_PROJECT="myproject"
export EDITOR="code --wait"  # Use VS Code for editing

azb list  # Uses environment variables
```

### Integration with Other Tools

**GitHub CLI Integration:**

```bash
# Create Azure Boards work item from GitHub issue
gh issue view 123 --json title,body | jq -r '.title,.body' | \
  xargs -I {} azb create --type Bug --title "{}"

# Link Azure Boards ID to GitHub PR
PR_NUMBER=$(gh pr create --title "Fix bug" --body "Fixes #1234")
azb update 1234 --field "Custom.GitHubPR=$PR_NUMBER"
```

**Git Integration:**

```bash
# Create branch from work item
WORKITEM_ID=1234
TITLE=$(azb show $WORKITEM_ID --format json | jq -r '.fields."System.Title"')
BRANCH_NAME="feature/$WORKITEM_ID-${TITLE// /-}"
git checkout -b "$BRANCH_NAME"

# Add work item ID to commit messages
git commit -m "feat: implement feature [AB#1234]"
```

---

## Tips and Best Practices

### 1. Use Aliases for Common Commands

Add to your shell profile (`.bashrc`, `.zshrc`, etc.):

```bash
alias abl='azb list --assigned-to @me --state Active'
alias abm='azb list --sprint current'
alias abs='azb show'
alias abc='azb create --template'
alias abd='azb dashboard'
```

### 2. Quick Work Item Creation

For frequently created work item types:

```bash
# Create function in shell profile
bug() {
  azb create --type Bug --title "$1" --assigned-to @me --priority 1
}

task() {
  azb create --type Task --title "$1" --assigned-to @me --sprint current
}

# Usage
bug "Login button not working"
task "Update documentation"
```

### 3. Daily Workflow

Start your day with:

```bash
# Launch dashboard to review work
azb

# Or use CLI to see your work
azb list --assigned-to @me --state Active

# Review work in current sprint
azb list --sprint current --format table
```

### 4. End-of-Sprint Cleanup

```bash
# Find completed work to close
azb list --sprint current --state Resolved --format table

# Bulk close resolved items
azb list --sprint current --state Resolved --format ids | \
  xargs -I {} azb update {} --state Closed

# Move incomplete work to next sprint
azb list --sprint current --state Active --format ids | \
  xargs -I {} azb update {} --iteration "Sprint 43"
```

### 5. Template Organization

- Create specific templates for different scenarios
- Use descriptive template names
- Include default values that rarely change
- Document your templates in a README in the templates directory

### 6. Keyboard Shortcuts in Dashboard

- Learn the keybindings for your most common actions
- Customize keybindings to match your workflow
- Use the help overlay (`?`) as a reference until you memorize shortcuts

### 7. Output Formats for Different Purposes

```bash
# Human-readable table (default)
azb list

# For scripting/automation
azb list --format ids

# For data analysis
azb list --format csv > data.csv

# For API integration
azb list --format json | jq '.[] | .fields."System.Title"'
```

### 8. Filtering Strategies

Be specific with filters to reduce noise:

```bash
# Too broad
azb list

# Better
azb list --assigned-to @me --state Active

# Best (very specific)
azb list --assigned-to @me --state Active --sprint current --type Bug
```

### 9. Work Item Relationships

When creating related work items:

1. Create parent first (Epic → Feature → User Story)
2. Use `--parent-id` when creating children
3. Or use templates with predefined children for consistent task breakdown

### 10. Regular Maintenance

```bash
# Check authentication status
azb auth status

# Verify configuration
azb config list

# Keep templates organized
ls -l ~/.azure-boards-cli/templates/

# Review and update keybindings
azb template edit keybinds  # if you have a keybinds template
```

---

## Troubleshooting

### Authentication Issues

**"not authenticated" error**

```bash
# Check status
azb auth status

# Re-authenticate
azb auth logout
azb auth login
```

**"invalid token" error**

Ensure your PAT has the correct scopes:
- Work Items (Read, Write)

Check expiration:
1. Go to `https://dev.azure.com/{org}/_usersSettings/tokens`
2. Verify token hasn't expired
3. Regenerate if needed
4. Run `azb auth login` with new token

### Configuration Issues

**"organization not configured" error**

```bash
azb config set organization <org>
```

**"project not configured" error**

```bash
azb config set project <project>
```

**Verify all configuration:**

```bash
azb config list
```

### Work Item Issues

**"work item not found" error**

- Verify the work item ID exists
- Check if you have permissions to view it
- Ensure you're in the correct project

**"failed to update work item" error**

- Check if the field name is correct (use `azb inspect <type>`)
- Verify field values are valid for the field type
- Some fields may be read-only

**"required field missing" error**

- Use `azb inspect <type>` to see required fields
- Some organizations have custom required fields
- Check with your Azure DevOps administrator

### TUI Dashboard Issues

**Dashboard not responding**

- Press `Esc` to cancel current action
- Press `q` to quit and restart
- Check your terminal size (minimum 80x24 recommended)

**Keybindings not working**

- Ensure you're not in filter mode (press `Esc` first)
- Check `~/.azure-boards-cli/keybinds.yaml` for conflicts
- Some terminals may not support certain key combinations

**Editor not opening (Edit action)**

```bash
# Set your EDITOR environment variable
export EDITOR="vim"
# or
export EDITOR="code --wait"
# or
export EDITOR="nano"

# Add to your shell profile to persist
echo 'export EDITOR="vim"' >> ~/.bashrc
```

### Template Issues

**"template not found" error**

```bash
# List available templates
azb template list

# Check template path
azb template path <template-name>
```

**Template validation errors**

- Ensure YAML syntax is correct
- Verify field names match Azure DevOps field reference names
- Check that work item type exists in your project

### Query Issues

**"query not found" error**

```bash
# List all queries to find the correct name/path
azb query list

# Use full path for shared queries
azb query run "Shared Queries/Team Backlog"
```

### Performance Issues

**Slow list commands**

- Use `--limit` to reduce result set
- Add more specific filters
- Check network connection to Azure DevOps

**Dashboard slow to load**

- Reduce default query result size in configuration
- Check cache settings
- Ensure stable network connection

### Common Error Messages

| Error | Solution |
|-------|----------|
| `TF401019: The Git repository with name or identifier does not exist` | Wrong project or organization configured |
| `VS403403: The current user does not have permission` | Check PAT permissions and scopes |
| `TF401232: Work item does not exist` | Verify work item ID in correct project |
| `VS402337: Required field '{field}' is not present` | Add required field or use template with all required fields |

### Getting Help

If you encounter issues not covered here:

1. Check the [GitHub Issues](https://github.com/SOMUCHDOG/ado-admin/issues)
2. Review the [README](README.md) for updates
3. Check the [SPEC](SPEC.md) for feature status
4. Open a new issue with:
   - Command you ran
   - Expected behavior
   - Actual behavior
   - Error messages
   - Your configuration (sanitized, no tokens)

### Debug Mode

For detailed debugging information:

```bash
# Enable verbose logging (if supported)
export AZB_DEBUG=1
azb list --assigned-to @me
```

### File Locations

Configuration and data files:

```
~/.azure-boards-cli/
├── config.yaml          # Main configuration
├── token               # Stored PAT (secure permissions)
├── keybinds.yaml       # Custom keybindings
└── templates/          # Work item templates
    ├── bug-report.yaml
    └── user-story.yaml
```

If you suspect file corruption:

```bash
# Backup current config
cp -r ~/.azure-boards-cli ~/.azure-boards-cli.backup

# Reset to defaults (will lose configuration)
rm -rf ~/.azure-boards-cli

# Reconfigure
azb config set organization myorg
azb config set project myproject
azb auth login
```

---

## Appendix

### Keyboard Reference

#### Global (All Tabs)
- `?` - Help
- `q` / `Ctrl+C` - Quit
- `Tab` - Next tab
- `Shift+Tab` - Previous tab
- `r` - Refresh
- `↑/↓` or `j/k` - Navigate lists
- `/` - Filter/search
- `Esc` - Cancel

#### Queries Tab
- `Enter` - Execute query / Toggle folder
- `E` - Expand all
- `C` - Collapse all

#### Work Items Tab
- `Enter` - Toggle details
- `n` - New work item
- `e` - Edit in editor
- `w` - Download as template
- `d` - Delete
- `s` - Change state
- `a` - Assign
- `t` - Add tags

#### Templates Tab
- `c` - Copy template
- `n` - New template
- `f` - New folder
- `e` - Edit template
- `d` - Delete template

### Common Field Names

Standard Azure DevOps fields you can use with `--field`:

| Field Name | Description | Example |
|------------|-------------|---------|
| `System.Title` | Work item title | `"Fix login bug"` |
| `System.Description` | Description | `"Users unable to login"` |
| `System.State` | State | `"Active"`, `"Resolved"`, `"Closed"` |
| `System.AssignedTo` | Assigned user | `"user@example.com"` or `"@me"` |
| `System.Tags` | Tags | `"bug,urgent"` |
| `System.AreaPath` | Area path | `"Project\\Team A"` |
| `System.IterationPath` | Iteration | `"Project\\Sprint 1"` |
| `Microsoft.VSTS.Common.Priority` | Priority | `1`, `2`, `3`, `4` |
| `Microsoft.VSTS.Common.Severity` | Severity | `"1 - Critical"`, `"2 - High"`, etc. |
| `Microsoft.VSTS.Scheduling.StoryPoints` | Story points | `5` |
| `Microsoft.VSTS.Scheduling.RemainingWork` | Remaining hours | `8` |
| `Microsoft.VSTS.Common.AcceptanceCriteria` | Acceptance criteria | Markdown text |
| `Microsoft.VSTS.Common.ValueArea` | Value area | `"Business"` or `"Architectural"` |

Use `azb inspect <type>` to discover all fields and custom fields for your organization.

### Work Item States

Common states (may vary by work item type and process template):

- `New` - Newly created, not started
- `Active` - Work in progress
- `Resolved` - Work completed, awaiting verification
- `Closed` - Verified and closed
- `Removed` - Cancelled or deleted

### Exit Codes

The CLI uses these exit codes:

- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Authentication error
- `4` - API error
- `5` - Validation error

Useful for scripting:

```bash
if azb update 1234 --state Closed; then
  echo "Work item closed successfully"
else
  echo "Failed to close work item"
  exit 1
fi
```

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.
