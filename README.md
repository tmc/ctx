# ctx - Context Gathering Tool

`ctx` is a simple, extensible tool to gather contextual information by running plugins. It discovers plugins named `ctx-*` in your PATH, executes them, aggregates their structured JSON output, and prints the combined context in YAML, JSON, or XML formats.

This tool is designed with simplicity and clear interfaces in mind. It assigns a `CTX_SESSION` ID and `CTX_SHLVL` to each run, and supports passing configuration settings (like caching directories, operational budgets, timeouts, or allowed tools) to plugins via environment variables.

See `docs/PLUGIN_SPEC.md` for mandatory plugin requirements and details on optional conventions like environment variables and metadata reporting. Use `ctx --print-spec` to view the specification directly.

## Installation

```bash
# Replace with your actual repo path before publishing
# Build using Go 1.18+ for debug.ReadBuildInfo support
go install github.com/tmc/ctx/cmd/ctx@latest
```

## Usage

Gather context (default output: YAML):
```bash
ctx```

Pass configuration and get XML output (example):
```bash
ctx --output-token-budget=2000 --cost-budget=50 --plugin-timeout=15s --output xml
```

Get compact JSON output:
```bash
ctx --output json --summary```

List discovered plugins:
```bash
ctx --list-plugins
```

Print the plugin specification:
```bash
ctx --print-spec
```

Display version information (dynamically read from build info):
```bash
ctx --version
```

## Configuration Flags

`ctx` accepts flags to control its behavior and pass configuration down to plugins via `CTX_*` environment variables:

*   `--output`: Output format (`yaml`, `json`, or `xml`, default: `yaml`).
*   `--list-plugins`: List discovered plugins and exit.
*   `--version`: Show version information derived from build metadata.
*   `--print-spec`: Print the plugin specification (`docs/PLUGIN_SPEC.md`) to stdout and exit.
*   `--cache-dir`: Specify a directory for plugins to use for caching (sets `CTX_CACHE_DIR` for plugins). Defaults to `$XDG_CACHE_HOME/ctx` or disabled if unset/unwritable.
*   `--output-token-budget`: Inform plugins of an estimated token budget for their primary output (sets `CTX_OUTPUT_TOKEN_BUDGET`).
*   `--thinking-token-budget`: Inform plugins of an estimated token budget for internal 'thinking' or intermediate steps (sets `CTX_THINKING_TOKEN_BUDGET`).
*   `--cost-budget`: Inform plugins of an estimated cost budget in USD cents (sets `CTX_COST_BUDGET_CENTS`).
*   `--allowed-tools`: Comma-separated list of external commands plugins are permitted to call (sets `CTX_ALLOWED_TOOLS`).
*   `--plugin-timeout`: Suggests a timeout duration for plugin execution (e.g., "30s", "1m"). Sets `CTX_TIMEOUT_SECONDS` and `CTX_DEADLINE_TIMESTAMP`.
*   `--plugin-retries`: Suggests a maximum number of retries for plugins (sets `CTX_RETRY_MAX`).
*   `-P, --parallel`: Maximum number of plugins to run in parallel (default: 1). This limits resource usage and prevents potential fork bombs.
*   `--indent`: Number of spaces for JSON/XML output indentation (default: 2).
*   `--summary`: Output compact JSON/XML without indentation (overrides --indent).
*   `--show-source`: Request plugins to include their source code in txtar format (sets `CTX_SHOW_SOURCE=true`). When using txtar format, any '-- filename --' directives in source files are escaped as '\-- filename --'.
*   `-v`: Enable verbose logging for debugging.

(Note: Implementation of plugin behavior based on `CTX_*` variables resides within the individual plugins.)

## Plugins

Plugins are executables starting with `ctx-` located in your system's PATH. They MUST adhere to the contract defined in `docs/PLUGIN_SPEC.md`, primarily outputting structured JSON.

`ctx` propagates `CTX_SESSION`, `CTX_SHLVL`, standard OpenTelemetry context (`TRACEPARENT`, `TRACESTATE`), and MAY set specific `CTX_*` configuration variables (e.g., `CTX_CACHE_DIR`, `CTX_TIMEOUT_SECONDS`) in the plugin's environment based on its own flags or configuration. Plugins MAY also optionally return structured metadata (like usage metrics or schema info) in their JSON output. See `docs/PLUGIN_SPEC.md` for details on both required and optional interactions, including incubating features like plugin integrity checks.

**Note on Existing Standalone Tools:** Tools like `ctx-exec` or `ctx-go-doc` from the `misc/ctx-plugins` repository have different output formats and are not directly compatible as plugins for this `ctx` runner. To use them, create simple **wrapper plugins** (e.g., `ctx-wrapper-exec`) that call the original tool and transform its output into the JSON format required by `docs/PLUGIN_SPEC.md`.

## Contributing

Contributions are welcome. Please focus on maintaining simplicity, clarity, and adherence to the defined specifications.

## License

ISC License. See [LICENSE](LICENSE).

