# pb-cli - Generic PocketBase CLI

A command-line interface for managing [PocketBase](https://pocketbase.io) instances. This CLI provides multi-environment context management, authentication, full CRUD operations, and complete backup management for any PocketBase deployment.

## Features

- **Multi-Environment Context Management**: Work with multiple PocketBase instances
- **Authentication Support**: Works with any auth collection (users, _superusers, custom collections)
- **Generic CRUD Operations**: Perform operations on any collection
- **Backup Management**: Create, download, upload, delete, and restore backups (**requires superuser access**)
- **Multiple Output Formats**: JSON, YAML, and table outputs
- **Pagination Support**: Handle large datasets efficiently

## Installation

### From Source

```bash
git clone https://github.com/skeeeon/pb-cli.git
cd pb-cli
go build -o pb main.go
```

## Quick Start

### 1. Create a Context

A context contains the configuration for a specific PocketBase environment:

```bash
# Create a context for your PocketBase instance
pb context create myapp \
  --url https://myapp.pocketbase.io \
  --collections posts,comments,users,categories

# For local development
pb context create local \
  --url http://localhost:8090 \
  --collections posts,users
```

### 2. Select the Active Context

```bash
pb context select myapp
```

### 3. Authenticate

```bash
# Authenticate with the default 'users' collection
pb auth

# Authenticate as superuser (required for backup operations)
pb auth --collection _superusers

# Authenticate with custom credentials
pb auth --email admin@example.com --password mypassword
```

### 4. Manage Collections

```bash
# List all posts
pb collections list posts

# Get a specific post with expanded relations
pb collections get posts post_123 --expand author,comments

# Create a new post
pb collections create posts '{"title":"My Post","content":"Hello world","published":true}'

# Update a post
pb collections update posts post_123 '{"published":true}'

# Delete a post (with confirmation)
pb collections delete posts post_123

# Short alias — "c" for "collections"
pb c list posts
pb c get posts post_123
```

### 5. Manage Backups ⚠️ **Superuser Required**

```bash
# First, authenticate as superuser
pb auth --collection _superusers

# List all backups
pb backup list

# Create a new backup
pb backup create --name "pre-migration"

# Download a backup to context folder
pb backup download backup_2024_01_15

# Upload a backup file
pb backup upload ./my-backup.zip --name "imported-backup"

# Restore from backup (with confirmation)
pb backup restore backup_2024_01_15

# Delete old backup
pb backup delete old_backup
```

## Commands Reference

### Context Management

```bash
# Create a new context
pb context create <n> --url <url> [--collections <collections>] [--auth-collection <collection>]

# List all contexts
pb context list

# Select active context
pb context select <n>

# Show context details
pb context show [name]

# Delete a context
pb context delete <n>

# Manage collections in context
pb context collections add <collections...>
pb context collections remove <collections...>
pb context collections list
pb context collections clear
```

### Authentication

```bash
# Authenticate (interactive)
pb auth

# Authenticate with specific credentials
pb auth --email user@example.com --password secret

# Authenticate with different collection
pb auth --collection _superusers   # Required for backup operations

# Authenticate with custom collection
pb auth --collection customers
```

### Collections CRUD

All collections commands use the pattern `pb collections <action> <collection>`. The `collections` command can be shortened to `c`.

```bash
# List records
pb collections list <collection> [options]
  --page int           Page number (default: 1)
  --limit int          Records per page (default: 30)
  --filter string      PocketBase filter expression
  --sort string        Sort expression (e.g., 'title', '-created')
  --fields strings     Specific fields to return
  --expand strings     Relations to expand
  --output string      Output format (json|yaml|table)

# Get single record
pb collections get <collection> <record_id> [options]
  --expand strings     Relations to expand
  --fields strings     Specific fields to return
  --output string      Output format

# Create record
pb collections create <collection> <json_data> [options]
pb collections create <collection> --file data.json
  --file string        Path to JSON file containing record data

# Update record
pb collections update <collection> <record_id> <json_data> [options]
pb collections update <collection> <record_id> --file data.json
  --file string        Path to JSON file containing record data

# Delete record
pb collections delete <collection> <record_id> [options]
  --force             Skip confirmation
  --quiet             Suppress output
```

### Backup Management ⚠️ **Superuser Required**

> **Important**: All backup operations require authentication with a superuser account.

```bash
# List all backups
pb backup list [options]
  --output string      Output format (json|yaml|table)

# Create backup
pb backup create [options]
  --name string        Custom backup name (optional)

# Download backup
pb backup download <backup_name> [output_path]
  --force             Overwrite existing files

# Upload backup
pb backup upload <file_path> [options]
  --name string        Custom backup name (uses filename if not specified)

# Delete backup
pb backup delete <backup_name> [options]
  --force             Skip confirmation

# Restore from backup
pb backup restore <backup_name> [options]
  --force             Skip confirmation (dangerous!)
```

## Configuration

### Context Directory Structure

pb-cli stores configuration in XDG-compliant directories:

```
~/.config/pb/
├── config.yaml           # Global configuration
├── myapp/                # Context directory
│   ├── context.yaml      # Context configuration
│   └── backups/          # Downloaded backups
│       ├── backup_2024_01_15
│       └── daily-backup-20240125
└── local/                # Another context
    ├── context.yaml
    └── backups/
```

### Global Configuration (`~/.config/pb/config.yaml`)

```yaml
active_context: myapp
output_format: json
colors_enabled: true
pagination_size: 30
debug: false
```

### Context Configuration (`~/.config/pb/myapp/context.yaml`)

```yaml
name: myapp
pocketbase:
  url: https://myapp.pocketbase.io
  auth_collection: _superusers  # Use '_superusers' for backup operations
  available_collections:
    - posts
    - comments
    - users
    - categories
  auth_token: "***HIDDEN***"
  auth_expires: "2024-02-01T10:00:00Z"
  auth_record: {}
```

## Advanced Usage

### Filtering and Sorting

```bash
# Filter published posts
pb collections list posts --filter 'published=true'

# Complex filtering
pb collections list posts --filter 'published=true && author.name~"John"'

# Sort by creation date (newest first)
pb collections list posts --sort '-created'

# Multiple sort fields
pb collections list posts --sort 'category,title'
```

### Pagination

```bash
# Get first 10 records
pb collections list posts --limit 10

# Get page 2
pb collections list posts --limit 10 --page 2

# Large datasets
pb collections list posts --limit 100 --page 4
```

### Working with Relations

```bash
# Expand single relation
pb collections get posts post_123 --expand author

# Expand multiple relations
pb collections get posts post_123 --expand author,comments,category

# List with expanded relations
pb collections list posts --expand author --filter 'published=true'
```

### Output Formats

```bash
# JSON output (default)
pb collections list posts --output json

# YAML output
pb collections list posts --output yaml

# Table output
pb collections list posts --output table

# Set global default
pb --output table collections list posts
```

### Creating Records with Files

```bash
# Create from file
pb collections create posts --file new-post.json

# Example new-post.json
{
  "title": "My New Post",
  "content": "This is the content of my post",
  "published": false,
  "author": "user_123",
  "tags": ["tech", "tutorial"]
}
```

### Piping and stdin

```bash
# Pipe JSON data into create
cat post.json | pb collections create posts

# Chain get and update via jq
pb collections get posts post_123 | jq '.title = "Updated"' | pb collections update posts post_123
```

### Advanced Backup Operations

#### Daily Backup Routine

```bash
# Authenticate as superuser first
pb auth --collection _superusers

# Create timestamped backup
pb backup create --name "daily-$(date +%Y%m%d)"

# List all backups with sizes and dates
pb backup list

# Download recent backups for external storage
pb backup download daily-20240125 ~/external-backups/
```

#### Backup Migration Between Environments

```bash
# Production backup
pb context select production
pb auth --collection _superusers
pb backup create --name "migration-source"
pb backup download migration-source ./temp-backup.zip

# Import to staging
pb context select staging
pb auth --collection _superusers
pb backup upload ./temp-backup.zip --name "from-production"
pb backup restore from-production

# Cleanup
rm ./temp-backup.zip
```

#### Backup Verification

```bash
# Check backup integrity
pb backup list --output json | jq '.[] | {name: .key, size: .size, age: .modified}'

# Download and verify backup
pb backup download important-backup ./verify/
ls -la ./verify/important-backup
```

### Debugging

```bash
# Enable debug output
pb --debug collections list posts

# Debug authentication issues
pb --debug auth --email user@example.com

# Debug backup operations
pb --debug backup create --name "test"
```

## Working with Different PocketBase Setups

### Admin Authentication for Backups

```bash
# Standard users authentication
pb context create prod \
  --url https://pb.mydomain.com \
  --auth-collection users \
  --collections posts,users

pb auth --collection _superusers
pb backup list
```

### Custom Auth Collections

```bash
# If your PocketBase uses 'members' instead of 'users'
pb context create myapp \
  --url https://myapp.pocketbase.io \
  --auth-collection members \
  --collections posts,members

pb auth --collection members
```

### Self-Hosted PocketBase

```bash
# Local development
pb context create dev \
  --url http://localhost:8090 \
  --auth-collection admins \
  --collections posts,users

# Docker deployment
pb context create prod \
  --url https://pb.mydomain.com \
  --auth-collection admins \
  --collections posts,users,categories
```

### Multiple Environments

```bash
# Development environment
pb context create dev --url http://localhost:8090 --collections posts,users
pb context select dev

# Production environment
pb context create prod --url https://api.myapp.com --collections posts,users
pb context select prod

# Switch between environments
pb context select dev
pb collections list posts

pb context select prod
pb backup list  # List production backups
```

## Tips and Tricks

### 1. Use the Short Alias

The `collections` command can be shortened to `c`:

```bash
pb c list posts --filter 'published=true'
pb c get users user_123
pb c create posts '{"title":"Quick post"}'
pb c delete posts post_123 -f
```

### 2. Use Shell Aliases

Create shell aliases for common operations:

```bash
alias pbb="pb backup"

# Now you can use:
pbb list
pbb create --name "quick-backup"
```

### 3. Context Switching

Quickly switch between environments:

```bash
alias pbdev="pb context select dev"
alias pbprod="pb context select prod"

pbdev
pb c list posts

pbprod
pb backup list  # View production backups
```

### 4. JSON Formatting

Use `jq` for advanced JSON processing:

```bash
# Get just the titles
pb c list posts --output json | jq '.items[].title'

# Count published posts
pb c list posts --filter 'published=true' --output json | jq '.totalItems'

# List backup sizes
pb backup list --output json | jq '.[] | {name: .key, size: .size}'
```

### 5. Backup Automation

Automate backup workflows:

```bash
# Create timestamped backup
alias pbbackup="pb backup create --name \"auto-backup-\$(date +%Y%m%d-%H%M)\""

# Download all backups for external storage
pb backup list --output json | jq -r '.[].key' | xargs -I {} pb backup download {} ~/external-backups/

# Cleanup old backups (be careful!)
pb backup list --output json | jq -r 'sort_by(.modified) | reverse | .[5:] | .[].key' | xargs -I {} pb backup delete {} --force
```

### 6. Multi-Environment Backup Strategy

Sync backups between environments:

```bash
# Production backup workflow
pb context select production
pb auth --collection _superusers
pb backup create --name "daily-backup-$(date +%Y%m%d)"
pb backup download "daily-backup-$(date +%Y%m%d)" ~/backups/prod/

# Restore to staging
pb context select staging
pb auth --collection _superusers
pb backup upload ~/backups/prod/daily-backup-$(date +%Y%m%d) --name "from-prod"
pb backup restore from-prod
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
