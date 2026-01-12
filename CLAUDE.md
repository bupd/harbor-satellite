# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Harbor Satellite is a registry fleet management and artifact distribution solution that extends Harbor container registry to edge computing environments. The system consists of two main components:

1. **Satellite**: Runs at edge locations as a lightweight, standalone registry that acts as both a primary registry for local workloads and a fallback for the central Harbor instance
2. **Ground Control**: Cloud-side management service that handles device management, onboarding, state management, and artifact orchestration

## Build and Development Commands

### Building

Using Dagger (recommended):
```bash
# Build satellite binary
dagger call build --source=. --component=satellite export --path=./bin

# Build ground-control binary
dagger call build-dev --platform "linux/amd64" --component "ground-control" export --path=./gc-dev

# Build both components (for CI)
dagger call build --component=satellite
dagger call build --component=ground-control
```

Using Go directly:
```bash
# Run satellite
go run cmd/main.go --token "<token>" --ground-control-url "<url>"

# Run ground-control (from ground-control directory)
cd ground-control && go run main.go
```

### Testing

```bash
# Run all tests with Dagger
dagger run go test ./... -v -count=1

# Run tests without Dagger
go test ./... -v -count=1 -args -abs=false

# Run E2E tests
dagger call test-end-to-end

# Test release (snapshot)
dagger call snapshot-release
```

### Linting

```bash
# Run golangci-lint via Dagger
dagger call lint-report export --path=golangci-lint.report

# The project uses extensive linting rules (see golangci.yaml)
# Key linters: gosec, govet, staticcheck, errcheck, revive, and many others
```

### Vulnerability Checking

```bash
# Run vulnerability checks
dagger call vulnerability-check-report export --path=vulnerability-check.report
```

### Running Locally

For Satellite:
```bash
# With Docker Compose (recommended)
docker compose up -d

# With Go
go run cmd/main.go --token "<token>" --ground-control-url "http://127.0.0.1:8080"

# With mirror configuration for container runtimes
go run cmd/main.go --token "<token>" --ground-control-url "<url>" --mirrors=containerd:docker.io,quay.io
```

For Ground Control:
```bash
# With Docker Compose
cd ground-control && docker compose up

# With Dagger
dagger call run-ground-control up

# With Go (after setting up .env file)
cd ground-control && go run main.go
```

## Architecture

### Satellite Component Structure

- `cmd/main.go`: Entry point for satellite binary. Handles CLI flags (token, ground-control-url, mirrors, json-logging)
- `pkg/config/`: Configuration management, validation, and hot-reloading
- `internal/satellite/`: Core satellite orchestration logic
- `internal/state/`: State management (replication, fetching, artifact handling, registration)
- `internal/registry/`: Local OCI registry management (Zot integration)
- `internal/scheduler/`: Cron-based job scheduling for state replication and registration
- `internal/container_runtime/`: CRI configuration management (Docker, containerd, CRI-O, Podman mirror setup)
- `internal/server/`: HTTP server for satellite metrics and health
- `internal/watcher/`: Config file watching for hot-reload
- `internal/hotreload/`: Hot-reload mechanism for configuration changes
- `internal/logger/`: Zerolog-based structured logging
- `internal/notifier/`: Email notification system
- `internal/utils/`: Shared utilities

### Ground Control Component Structure

- `ground-control/main.go`: Entry point that checks Harbor health and starts server
- `ground-control/internal/server/`: HTTP API server with handlers for satellites, groups, configs
- `ground-control/internal/database/`: Database models and operations (PostgreSQL)
- `ground-control/internal/models/`: Domain models
- `ground-control/internal/harborhealth/`: Harbor registry health checking
- `ground-control/reg/harbor/`: Harbor API client (projects, robots, replication)
- `ground-control/migrator/`: Database migration handling
- `ground-control/seed/`: Database seeding utilities

### Key Concepts

**Groups**: Collections of container images that satellites need to replicate. Contains artifact metadata (repository, tag, digest, type).

**Configs**: Configuration artifacts that define how satellites connect to Ground Control, replication intervals, and local registry settings (including Zot config).

**State Replication**: Periodic sync process where satellites fetch desired state from Ground Control and replicate artifacts locally.

**Registration**: Periodic process where satellites register/heartbeat with Ground Control using their token.

**Mirror Configuration**: Satellites can configure container runtimes (Docker, containerd, CRI-O, Podman) to use the local registry as a mirror, with fallback to upstream registries.

### Configuration Files

Satellite uses JSON configuration with three sections:
- `state_config`: Registry credentials and state URL
- `app_config`: Ground Control URL, log level, replication intervals, local registry settings
- `zot_config`: Embedded Zot registry configuration (storage, HTTP, logging)

Ground Control uses environment variables (see ground-control/.env.example):
- Harbor credentials (HARBOR_USERNAME, HARBOR_PASSWORD, HARBOR_URL)
- Database connection (DB_HOST, DB_PORT, DB_DATABASE, DB_USERNAME, DB_PASSWORD)
- Server settings (PORT, APP_ENV)

## Important Development Notes

### Two-Module Structure

This repository contains two Go modules:
1. Root module (`go.mod`): Satellite component
2. `ground-control/go.mod`: Ground Control component

When making changes, be aware which module you're working in. Dependencies and imports are separate between modules.

### State Management

The satellite maintains state in a JSON file (default: config.json) that contains:
- Current configuration
- List of artifacts to replicate
- Registry URLs and credentials

State is fetched from Ground Control at regular intervals (default: every 10 seconds, configurable via `state_replication_interval`).

### Container Runtime Integration

Satellite supports configuring multiple CRIs as mirrors using the `--mirrors` flag:
- Format: `--mirrors=<CRI>:<registry1>,<registry2>`
- Example: `--mirrors=containerd:docker.io,quay.io --mirrors=podman:docker.io`
- Note: Docker only supports mirroring docker.io, use `--mirrors=docker:true`
- Requires sudo for updating CRI config files
- Docker service restart is required for changes to take effect

### Database Migrations

Ground Control uses SQL migrations in `ground-control/migrator/`. Migrations run automatically on startup.

### Hot Reload

Satellite supports hot-reloading configuration changes without restart. The watcher monitors config file changes and triggers reload when detected.

## Testing Strategy

- Unit tests are colocated with source files (`*_test.go`)
- E2E tests are in `test/e2e/`
- Test configuration files are in `test/e2e/testconfig/`
- Tests can be run with or without Dagger (use `-args -abs=false` for non-Dagger runs)

## Code Style and Linting

The project uses strict linting via golangci-lint with 50+ linters enabled. Key requirements:
- No global variables (gochecknoglobals)
- No init functions (gochecknoinits)
- Cyclomatic complexity limits (cyclop, gocognit)
- Function length limits (funlen: 100 lines, 50 statements)
- Error handling required (errcheck)
- Security checks (gosec)
- Proper struct tags (musttag)
- No magic numbers (gomnd)

When writing code, follow the existing patterns to pass linting.

## CI/CD Pipeline

GitHub Actions workflows:
- `.github/workflows/test.yaml`: Runs vulnerability checks, test release, builds, and E2E tests
- `.github/workflows/lint.yaml`: Runs golangci-lint and fails on any issues
- `.github/workflows/release.yaml`: Handles releases via GoReleaser

All CI uses Dagger for consistent builds across environments.

## Architecture Decisions

The `docs/decisions/` directory contains ADRs (Architecture Decision Records):
- ADR-0001: Skopeo vs Crane (chose Skopeo for image copying)
- ADR-0002: Zot vs Docker Registry (chose Zot for OCI compliance and features)
- ADR-0003: Remote config injection (chose API-based config delivery)

Read these for context on key technical decisions.

## Common Workflows

### Adding a new API endpoint to Ground Control

1. Add handler function in `ground-control/internal/server/*_handlers.go`
2. Register route in `ground-control/internal/server/routes.go`
3. Add database operations if needed in `ground-control/internal/database/`
4. Update models in `ground-control/internal/models/` if needed
5. Run tests and lint

### Adding a new satellite feature

1. Implement in appropriate `internal/` package
2. Update `pkg/config/` if configuration changes needed
3. Add validation in `pkg/config/validate.go`
4. Update `cmd/main.go` if new CLI flags needed
5. Update config.example.json
6. Run tests and lint

### Modifying state replication logic

State replication is handled in `internal/state/`:
- `fetcher.go`: Fetching state from Ground Control
- `replicator.go`: Replicating artifacts to local registry
- `state_process.go`: Orchestrating the state sync process
- `registration_process.go`: Satellite registration with Ground Control

Be careful with changes here as this is core functionality.
