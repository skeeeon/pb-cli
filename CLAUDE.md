# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

pb-cli is a Go CLI tool for managing PocketBase instances. It provides multi-environment context management, authentication, CRUD operations on collection records, schema inspection, and backup management. Built with Cobra (commands), go-resty (HTTP), and Viper (config). The compiled binary is named `pb`. It targets PocketBase v0.23+ (collection definitions use `fields`; superuser auth lives in the `_superusers` collection).

## Build & Test Commands

```bash
go build -o pb .                       # Build the binary
go test ./...                          # Run all tests
go test ./internal/config/...          # Run tests for a specific package
go test -v -run TestName ./internal/config/...  # Run a single test
go vet ./...                           # Vet
gofmt -l .                             # List unformatted files (should be empty)
```

Releases are built with GoReleaser (`.goreleaser.yaml`, GoReleaser v2). The version
string is injected at build time via ldflags into `pb-cli/cmd.version`; a plain
`go build` leaves it as `dev`.

```bash
goreleaser check                       # Validate the release config
goreleaser build --snapshot --clean    # Cross-compile locally (no publish)
goreleaser release --clean             # Tag-driven release (needs GITHUB_TOKEN)
```

No Makefile or task runner — use `go`/`goreleaser` directly.

## Architecture

### Layers

1. **`cmd/`** — Cobra command definitions. `cmd/root.go` bootstraps the app: initializes `config.Manager` in `PersistentPreRunE`, then distributes it to subcommand packages via `SetConfigManager()` setters. `main.go` prints any returned error once to stderr and exits 1; the root command sets `SilenceErrors`/`SilenceUsage` so cobra does not also print it.
2. **`internal/pocketbase/`** — HTTP client wrapping PocketBase's REST API via go-resty. `client.go` handles all HTTP calls; `errors.go` converts HTTP responses into `PocketBaseError` with friendly messages and suggestions; `types.go` defines API data structures; `auth.go` handles login, token refresh, and JWT expiry parsing.
3. **`internal/config/`** — XDG-compliant config persistence (`~/.config/pb/`). Global config (`config.yaml`) tracks active context and defaults. Each context gets a subdirectory with `context.yaml` storing URL, auth collection, and the cached auth token/expiry.
4. **`internal/utils/`** — Output formatting (JSON/YAML/table), colored messaging helpers, validation, and interactive prompts (`prompt.go`).

### Command routing for collections

`cmd/collections/` uses proper Cobra subcommands with action-first syntax: `pb collections <action> <collection>` (alias: `pb c <action> <collection>`). Each action (list, get, create, update, delete) is its own file with scoped flags. Shared helpers (validation, client creation, JSON parsing) live in `root.go`.

Collection names are passed straight to the API — there is **no allowlist to register first** (`pb schema` lists what exists). `pb collections list` returns one page by default; `--all` walks every page (500/request) and is mutually exclusive with `--page`/`--limit`.

### Schema inspection

`cmd/schema/` implements `pb schema [collection]`: with no argument it lists collections; with a name it shows that collection's fields and access rules. It calls the collection endpoints (`GetCollections`/`GetCollectionSchema`), which are **superuser-only** in PocketBase, so a 401/403 surfaces a `pb auth --collection _superusers` hint.

### HTTP client

`client.go` funnels requests through `doRequest(client, method, endpoint, body)`, which centralizes URL building and error handling. `makeRequest` uses the default timeout-bounded client (`apiTimeout`, 30s) for ordinary API calls. Long-running backup operations (create/restore/upload/download) use `newTransferClient()`, which has **no timeout**, so large-database transfers aren't killed mid-stream.

## Key conventions

- **stdout vs stderr**: Data output goes to stdout (for piping); all status, prompts, and error messages go to stderr.
- **Confirmation prompts**: destructive actions confirm via `utils.Confirm` (y/N) or `utils.ConfirmWord` (type an exact word), which return `(bool, error)`. Callers MUST abort on a `false` result (`if !confirmed { return nil }`) *before* the destructive call — returning `nil` from a confirm helper does not stop anything. (A prior bug where cancel still deleted came from ignoring this.)
- **JSON input**: Create/update accept JSON from positional arg, `--file` flag, or stdin (pipe detection), in that precedence.
- **Config injection**: The config manager is passed to subcommands via setter functions, not globals.
- **Auth tokens**: Stored in context YAML files, checked for expiry before API calls. The context file is written `0600` and its directories `0700` because it holds the plaintext token — preserve these modes in `internal/config/manager.go`.
- **Non-interactive auth**: `pb auth` resolves email as `--email` > `PB_EMAIL` > prompt, and password as `--password` > `--password-stdin` > `PB_PASSWORD` > prompt. `pb auth status` (alias `whoami`) and `pb auth logout` inspect/clear the stored token.
- **Superuser operations**: `pb schema` and all `pb backup` commands require `_superusers` authentication (`pb auth --collection _superusers`). Record CRUD (`pb collections ...`) works with whatever collection the active token can access.
- **Output format**: every command resolves its format as `--output/-o` flag, else the global `output_format` (default `json`). Avoid hardcoding a per-command default; fall back to `config.Global.OutputFormat`.
- **No speculative code**: keep the surface minimal — delete unused helpers/types rather than keeping them "for later" (`git` remembers).
