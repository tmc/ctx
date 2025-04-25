# Changelog

## [Unreleased]

### Core Functionality
- Added `-P, --parallel` flag to limit concurrent plugin execution (default: 1), preventing potential fork bombs
- Added `-v` flag for verbose logging and removed default logging for cleaner output
- Removed help flag exit code warning, making plugins more lenient with exit codes
- Added `--show-source` flag to output plugin source code in txtar format, with proper escaping

### Output Formats & Configuration
- Added XML output support and improved formatting with `--indent` and `--summary` flags
- Added timeout and retry control with `--plugin-timeout` and `--plugin-retries` flags
- Added resource budgeting with `--output-token-budget`, `--thinking-token-budget`, `--cost-budget`
- Added `--allowed-tools` for security and `--cache-dir` for caching support

### Environment Variables
- Added propagation of key variables: `CTX_SESSION`, `CTX_SHLVL`, `CTX_SHOW_SOURCE`
- Added budget env vars, timeout env vars, and retry env vars
- Re-added OpenTelemetry context propagation (`TRACEPARENT`, `TRACESTATE`)

### Documentation & Specifications
- Consolidated documentation in `docs/PLUGIN_SPEC.md` using RFC 2119 keywords
- Added `--print-spec` flag to output the embedded plugin specification
- Added conventions for plugins to report metrics and schema information
- Added sections for plugin integrity, approvals, and reporting mechanisms

### Changed
- Version info derived dynamically from build info using `runtime/debug`
- Refactored core logic for simplicity and robustness
- Plugins now must output structured JSON (raw output no longer supported)

### Removed
- Removed separate docs files, simplified convention documentation
- Removed unused configuration flags and temperature/reasoning level props
- Removed placeholder logic for planning, security filters, and budget management

## [0.1.0] - YYYY-MM-DD
- Initial project structure and basic concepts (based on v1 branch history)

