# ctx - Context Gathering Tool

`ctx` is a simple, extensible tool to gather contextual information by running plugins. It discovers plugins named `ctx-*` in your PATH, executes them, aggregates their structured JSON output, and prints the combined context.

This tool is designed with simplicity and clear interfaces in mind. It assigns a `CTX_SESSION` ID to each run and supports passing configuration settings (like caching directories or operational budgets) to plugins via environment variables.

See `docs/PLUGIN_SPEC.md` for mandatory plugin requirements and details on optional conventions like environment variables and metadata reporting.

## Installation

```bash
# Replace with your actual repo path before publishing
go install github.com/tmc/ctx/cmd/ctx@latest
```

## Usage

Gather context (default output: YAML):
```bash
ctx```

Pass configuration to plugins (example):
```bash
ctx --output-token-budget=2000 --cost-budget=50 --cache-dir=/tmp/ctxcache --output json
```

List discovered plugins:
```bash
ctx --list-plugins
```

## Configuration Flags

`ctx` accepts flags to control its behavior and pass configuration down to plugins via `CTX_*` environment variables:

*   `--output`: Output format (`yaml` or `json`, default: `yaml`).
*   `--list-plugins`: List discovered plugins and exit.
*   `--version`: Show version information.
*   `--cache-dir`: Specify a directory for plugins to use for caching (sets `CTX_CACHE_DIR` for plugins). Defaults to `$XDG_CACHE_HOME/ctx` or disabled if unset/unwritable.
*   `--output-token-budget`: Inform plugins of an estimated token budget for their primary output (sets `CTX_OUTPUT_TOKEN_BUDGET`).
*   `--thinking-token-budget`: Inform plugins of an estimated token budget for internal 'thinking' or intermediate steps (sets `CTX_THINKING_TOKEN_BUDGET`).
*   `--cost-budget`: Inform plugins of an estimated cost budget in USD cents (sets `CTX_COST_BUDGET_CENTS`).

(Note: Implementation of plugin behavior based on these `CTX_*` variables resides within the individual plugins.)

## Plugins

Plugins are executables starting with `ctx-` located in your system's PATH. They MUST adhere to the contract defined in `docs/PLUGIN_SPEC.md`.

`ctx` propagates a `CTX_SESSION` environment variable and MAY set specific `CTX_*` configuration variables (e.g., `CTX_CACHE_DIR`, `CTX_OUTPUT_TOKEN_BUDGET`) in the plugin's environment based on its own flags or configuration. Plugins MAY also optionally return structured metadata (like usage metrics) in their JSON output. See `docs/PLUGIN_SPEC.md` for details on both required and optional interactions, including conventions around OpenTelemetry context propagation.

## Contributing

Contributions are welcome. Please focus on maintaining simplicity, clarity, and adherence to the defined specifications.

## License

ISC License. See [LICENSE](LICENSE).

