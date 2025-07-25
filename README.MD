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

# Authenticate as admin (required for backup operations)
pb auth --collection admins

# Authenticate with custom credentials
pb auth --email admin@example.com --password mypassword
```

### 4. Manage Collections

```bash
# List all posts
pb collections posts list

# Get a specific post with expanded relations
pb collections posts get post_123 --expand author,comments

# Create a new post
pb collections posts create '{"title":"My Post","content":"Hello world","published":true}'

# Update a post
pb collections posts update post_123 '{"published":true}'

# Delete a post (with confirmation)
pb collections posts delete post_123
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
pb context collections remove <collection>
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

```bash
# List records
pb collections <collection> list [options]
  --offset int         Record offset for pagination
  --limit int          Records per page (default: 30)
  --filter string      PocketBase filter expression
  --sort string        Sort expression (e.g., 'title', '-created')
  --fields strings     Specific fields to return
  --expand strings     Relations to expand
  --output string      Output format (json|yaml|table)

# Get single record
pb collections <collection> get <record_id> [options]
  --expand strings     Relations to expand
  --output string      Output format

# Create record
pb collections <collection> create <json_data> [options]
pb collections <collection> create --file data.json

# Update record
pb collections <collection> update <record_id> <json_data> [options]
pb collections <collection> update <record_id> --file data.json

# Delete record
pb collections <collection> delete <record_id> [options]
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
pb collections posts list --filter 'published=true'

# Complex filtering
pb collections posts list --filter 'published=true && author.name~"John"'

# Sort by creation date (newest first)
pb collections posts list --sort '-created'

# Multiple sort fields
pb collections posts list --sort 'category,title'
```

### Pagination

```bash
# Get first 10 records
pb collections posts list --limit 10

# Get next 10 records
pb collections posts list --limit 10 --offset 2

# Large datasets
pb collections posts list --limit 100 --offset 4
```

### Working with Relations

```bash
# Expand single relation
pb collections posts get post_123 --expand author

# Expand multiple relations
pb collections posts get post_123 --expand author,comments,category

# List with expanded relations
pb collections posts list --expand author --filter 'published=true'
```

### Output Formats

```bash
# JSON output (default)
pb collections posts list --output json

# YAML output
pb collections posts list --output yaml

# Table output
pb collections posts list --output table

# Set global default
pb --output table collections posts list
```

### Creating Records with Files

```bash
# Create from file
pb collections posts create --file new-post.json

# Example new-post.json
{
  "title": "My New Post",
  "content": "This is the content of my post",
  "published": false,
  "author": "user_123",
  "tags": ["tech", "tutorial"]
}
```

### Advanced Backup Operations

#### Daily Backup Routine

```bash
# Authenticate as admin first
pb auth --collection admins

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
pb auth --collection admins
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
pb --debug collections posts list

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
pb collections posts list

pb context select prod  
pb backup list  # List production backups
```

## Error Handling

pb-cli provides helpful error messages and suggestions:

```bash
# Authentication required
$ pb collections posts list
Error: authentication required. Please run 'pb auth' to authenticate

# Collection not configured
$ pb collections blogs list  
Error: collection 'blogs' not configured in context. Available collections: posts, users. 
Add with 'pb context collections add blogs'

# Validation errors
$ pb collections posts create '{"title": ""}'
Error: validation failed:
  - Title is required
```

## Tips and Tricks

### 1. Use Aliases

Create shell aliases for common operations:

```bash
alias pbp="pb collections posts"
alias pbu="pb collections users"
alias pbb="pb backup"

# Now you can use:
pbp list --filter 'published=true'
pbu get user_123
pbb list
```

### 2. Context Switching

Quickly switch between environments:

```bash
alias pbdev="pb context select dev"
alias pbprod="pb context select prod"

pbdev
pb collections posts list

pbprod  
pb backup list  # View production backups
```

### 3. JSON Formatting

Use `jq` for advanced JSON processing:

```bash
# Get just the titles
pb collections posts list --output json | jq '.items[].title'

# Count published posts
pb collections posts list --filter 'published=true' --output json | jq '.totalItems'

# List backup sizes
pb backup list --output json | jq '.[] | {name: .key, size: .size}'
```

### 4. Backup Automation

Automate backup workflows:

```bash
# Create timestamped backup
alias pbbackup="pb backup create --name \"auto-backup-\$(date +%Y%m%d-%H%M)\""

# Download all backups for external storage
pb backup list --output json | jq -r '.[].key' | xargs -I {} pb backup download {} ~/external-backups/

# Cleanup old backups (be careful!)
pb backup list --output json | jq -r 'sort_by(.modified) | reverse | .[5:] | .[].key' | xargs -I {} pb backup delete {} --force
```

### 5. Multi-Environment Backup Strategy

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

