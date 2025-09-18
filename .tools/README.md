# Project Tools Directory

This directory contains project-specific tooling organized into two main categories:

## Directory Structure

```
.tools/
├── ext/          # External tools (ignored by git)
│   └── bin/      # Downloaded/installed external tools
└── repo/         # Repository-specific scripts (committed)
    └── bin/      # Project-specific executable scripts
```

## External Tools (`.tools/ext/`)

Contains external tools that are downloaded or installed locally to avoid polluting the global system or requiring specific versions in PATH.

- **Purpose**: Isolated dependency management for development tools
- **Git Status**: Ignored (added to `.gitignore`)
- **Installation**: Use project-specific recipes (e.g., `just install-golangci-lint`)
- **Examples**: `golangci-lint`, `staticcheck`, etc.

### Installing External Tools

```bash
# Install golangci-lint locally
just install-golangci-lint

# The tool will be available at:
# ~/.tools/ext/bin/golangci-lint
```

## Repository Scripts (`.tools/repo/`)

Contains project-specific scripts and utilities that are part of the codebase and should be version controlled.

- **Purpose**: Consistent tooling across all contributors
- **Git Status**: Committed to repository
- **Naming**: Use kebab-case (e.g., `go-fmt-check`)
- **Examples**: Format checkers, custom linters, build helpers

### Current Scripts

- **`go-fmt-check`**: Validates Go code formatting without modifying files
  - Used in CI pipeline to ensure code is properly formatted
  - Fails with exit code 1 if formatting issues are found
  - Usage: `./.tools/repo/bin/go-fmt-check`

## Design Principles

1. **Separation of Concerns**: External tools vs project-specific scripts
2. **Git Hygiene**: Only commit what belongs in version control
3. **Team Consistency**: Everyone uses the same tools and scripts
4. **No Global Pollution**: Tools don't interfere with system installations
5. **Self-Documenting**: Clear naming and organization

## Usage in Justfile

The project's `Justfile` uses this structure for consistent tool access:

```just
# External tools with fallbacks
lint:
    @if command -v ~/.tools/ext/bin/golangci-lint >/dev/null 2>&1; then \
        ~/.tools/ext/bin/golangci-lint run; \
    elif command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run; \
    else \
        echo "golangci-lint not found, falling back to go vet"; \
        go vet ./...; \
    fi

# Repository scripts
fmt-check:
    ./.tools/repo/bin/go-fmt-check
```

## Adding New Tools

### External Tool
1. Add installation recipe to `Justfile`
2. Update lint/build recipes to use the tool
3. Document in this README

### Repository Script
1. Create executable script in `.tools/repo/bin/`
2. Use kebab-case naming
3. Add usage recipe to `Justfile`
4. Document in this README

## Benefits

- **Reproducible builds**: Everyone uses the same tool versions
- **Clean git history**: No binary tools in commits
- **Isolated dependencies**: Tools don't conflict with system packages
- **Easy onboarding**: New contributors get consistent environment
- **CI compatibility**: Same tools work in CI and local development