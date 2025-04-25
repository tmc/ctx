# Changelog

## [Unreleased]
### Added
- Added `CTX_SESSION` environment variable, generated per `ctx` run and propagated to plugins.
- Added flags (`--output-token-budget`, `--thinking-token-budget`, `--cost-budget`) to `ctx`.
- `ctx` now propagates budget configuration to plugins via `CTX_*` environment variables (`CTX_OUTPUT_TOKEN_BUDGET`, `CTX_THINKING_TOKEN_BUDGET`, `CTX_COST_BUDGET_CENTS`).
- Added convention for plugins to optionally report `input/output/thinking_token_count` and `cost_estimate_cents` in JSON output metrics.
- Added `--cache-dir` flag and `CTX_CACHE_DIR` env var propagation.
- Consolidated documentation into `docs/PLUGIN_SPEC.md`, covering core requirements, env vars, optional metrics, and incubating conventions. Formalized spec using RFC 2119 keywords.
- Implemented concurrent plugin execution.
- Added basic logging for plugin execution status.

### Changed
- Renamed `--token-budget` flag/var to `--output-token-budget`/`CTX_OUTPUT_TOKEN_BUDGET`.
- Plugins MUST now output structured JSON adhering to `PLUGIN_SPEC.md`. Raw string output is no longer supported.
- Refactored core logic for simplicity and robustness.
- OpenTelemetry trace propagation (`TRACEPARENT`, `TRACESTATE`) is now documented as SHOULD/MAY within the plugin spec, not enabled by default in code (can be added back easily if needed).

### Removed
- Removed separate `docs/SEMANTIC_CONVENTIONS.md` and `docs/FUTURE.md` files (content merged or dropped).
- Removed configuration/propagation for Temperature and Reasoning Level.
- Removed Risk Assessment convention documentation.
- Removed placeholder logic for planning, security filters, token budget management from core.
- Removed `--config`, `--print-plugin-spec`, `--plugin` flags from v1 code.
- Removed `--capabilities` requirement/handling for plugins (moved to Incubating).

## [0.1.0] - YYYY-MM-DD
### Added
- Initial project structure and basic concepts (based on v1 branch history).

