# ctx - Context Gathering Tool for Language Models

ctx is a flexible and extensible tool designed to gather contextual information primarily for use with Language Models (LLMs). It streamlines the process of providing accurate and comprehensive information to language models.

## Table of Contents
1. [Key Features](#key-features)
2. [Architecture](#architecture)
3. [Installation](#installation)
4. [Usage Examples](#usage-examples)
5. [Key Plugins](#key-plugins)
6. [Configuration](#configuration)
7. [CLI Interface](#cli-interface)
8. [Plugin Development](#plugin-development)
9. [Contributing](#contributing)
10. [Troubleshooting](#troubleshooting)

## Key Features

- **Modular Plugin System**: Easily extendable with custom plugins
- **Auto-discovery**: Automatically detects programming environment and relevant plugins
- **Configurable Output**: Supports various output formats (YAML, JSON, plaintext)
- **Environment-aware**: Gathers information about the current working directory, detected programming languages, and other relevant factors
- **Flexible Configuration**: Configurable via command-line flags, environment variables, or configuration files
- **Dry Run Capability**: Preview ctx's actions without executing plugins or gathering real data

## Architecture

ctx follows a three-phase execution model:

1. **Discovery Phase**: Searches for plugins with the `ctx-` prefix in the system PATH and gathers initial environment information.
2. **Planning Phase**: Analyzes discovered plugins, considers user-provided hints, and generates a comprehensive plan for context gathering.
3. **Execution Phase**: Follows the generated plan, executes plugins, collects context information, and formats the output.

For more details, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation

```bash
go install github.com/tmc/ctx/cmd/ctx@latest
```

For more detailed installation instructions, see [INSTALLATION.md](docs/INSTALLATION.md).

## Usage Examples

```bash
# Gather context for the current directory
ctx

# Gather context and output as JSON
ctx --output json

# Perform a dry run
ctx --dry-run

# Use specific plugins
ctx --plugins git,src,env
```

## Key Plugins

- ctx-git: Gathers information about the current git repository, including branch, commits, and configuration.
- ctx-src: Analyzes the file system to gather information about the source code structure and contents.
- ctx-env: Collects relevant environment variables that may impact the context.
- ctx-src-languages: Detects the programming languages used in the source code and provides language-specific insights.
- ctx-src-python: Gathers detailed information about Python source code, including imports, classes, and functions.

## Configuration

ctx can be configured using:
- Command-line flags
- Environment variables
- Configuration files


## CLI Interface

Run `ctx --help` for a list of available commands and options.

## Plugin Development

For information on developing plugins for ctx, see [PLUGIN_SPEC.md](docs/PLUGIN_SPEC.md).

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](docs/CONTRIBUTING.md) for more information on how to get started, report bugs, or request features.

## License

See [LICENSE](LICENSE) for details.
