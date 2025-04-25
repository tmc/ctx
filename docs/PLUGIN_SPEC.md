# ctx Plugin Specification v0.1.0

This specification defines the requirements and conventions for executables acting as plugins for the `ctx` context-gathering tool. Adherence to the mandatory requirements ensures basic compatibility and interoperability.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

## 1. Naming Convention

Plugins discovered via the system PATH **MUST** be named with the prefix `ctx-` followed by a descriptive name (e.g., `ctx-git`, `ctx-env`).

## 2. Execution

The `ctx` tool executes discovered plugins as child processes.

*   When executed **without arguments**, a plugin **MUST** perform its primary context gathering function and print its result to standard output according to the format defined in Section 3.
*   Plugins **MUST** respond to help flags (`-h`, `-help`, or `--help`) by printing usage information to standard error and exiting with a non-zero status code.
*   Plugins **SHOULD** generally be robust against receiving unknown flags or arguments, perhaps by ignoring them or printing a warning to standard error.
*   Plugins **SHOULD** use standard error for diagnostic messages, logs, or warnings not intended for context aggregation.

### 2.1 Environment Variables

Plugins **MUST** expect that their execution environment will be largely inherited from the `ctx` process. In addition, `ctx` **MAY** explicitly set or propagate certain standard environment variables listed below. Plugins **MAY** also read other standard system or tool-specific environment variables for their own configuration needs (e.g., a `ctx-git` plugin might read standard `GIT_*` variables). Plugins **SHOULD** check for the `CTX_*` variables listed here if their behavior might be affected by `ctx`-provided configuration or context.

| Variable                       | Set/Propagated By `ctx`? | Description                                                                                                 | Notes                                                                                                        |
| :----------------------------- | :----------------------- | :---------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------- |
| `CTX_SESSION`                  | Yes (Generated/Propagated)| An opaque string ID (ULID by default) unique to the current `ctx` invocation. Useful for correlation/logging. | `ctx` ensures this is set.                                                                                   |
| `CTX_SHLVL`                    | Yes (Incremented)        | Indicates the nesting level of `ctx` execution (integer, starts at 1). Similar to `SHLVL`.                | Incremented from parent `CTX_SHLVL` or 1 if parent unset. Useful for detecting recursion.                |
| `CTX_CACHE_DIR`                | Yes (Based on Flag/XDG)  | Specifies the base directory plugins SHOULD use for caching.                                                | Set if `--cache-dir` used or XDG dir found. Plugins MAY disable caching if unset/unusable.             |
| `CTX_OUTPUT_TOKEN_BUDGET`      | Yes (Based on Flag)      | Informs the plugin of an estimated token budget (integer) for its primary output (`data` field).            | Set if `--output-token-budget > 0`. Interpretation of "token" is context-dependent.                       |
| `CTX_THINKING_TOKEN_BUDGET`    | Yes (Based on Flag)      | Informs the plugin of an estimated token budget (integer) for internal reasoning or intermediate steps.   | Set if `--thinking-token-budget > 0`. Useful for plugins that consume tokens internally.                   |
| `CTX_COST_BUDGET_CENTS`        | Yes (Based on Flag)      | Informs the plugin of an estimated cost budget in **USD cents** (integer).                                | Set if `--cost-budget > 0`. Plugin MAY attempt to stay within budget.                                      |
| `CTX_ALLOWED_TOOLS`            | Yes (Based on Flag)      | A comma-separated list of external command names that the plugin is permitted to call.                    | Set if `--allowed-tools` is provided. Plugins SHOULD respect this if they call external tools.           |
| `CTX_TIMEOUT_SECONDS`          | Yes (Based on Flag)      | Suggests a timeout in seconds (integer) for the plugin's operation.                                       | Set if `--plugin-timeout > 0`. Plugins MAY use this to configure internal operations.                        |
| `CTX_DEADLINE_TIMESTAMP`       | Yes (Based on Flag)      | Suggests an absolute deadline as a Unix timestamp (integer seconds since epoch) for operation completion. | Set if `--plugin-timeout > 0`. Plugins MAY use this to avoid starting work near the deadline.             |
| `CTX_RETRY_MAX`                | Yes (Based on Flag)      | Suggests a maximum number of retries (integer) the plugin might attempt internally for transient errors.  | Set if `--plugin-retries > 0`.                                                                                |
| `CTX_SHOW_SOURCE`              | Yes (Based on Flag)      | Requests plugins to include their source code in txtar format when available.                          | Set if `--show-source` is provided. Plugins SHOULD include source in txtar format when requested. When outputting in txtar format, plugins MUST escape any top-level '-- filename --' directives in source files to '\-- filename --' to prevent them from being interpreted as txtar directives.               |
| `TRACEPARENT`                  | Propagated               | W3C Trace Context parent identifier.                                                                        | Propagated only if set in `ctx`'s environment. Instrumented plugins SHOULD respect this.                 |
| `TRACESTATE`                   | Propagated               | W3C Trace Context state information.                                                                        | Propagated only if set in `ctx`'s environment alongside `TRACEPARENT`.                                   |
| `CTX_APPROVED`                 | Yes (Conditionally)      | Set to "true" by `ctx` when re-running a plugin after user approval.                                      | Experimental: Part of the Incubating User Approval Flow.                                                      |

Other standard system variables (like `PATH`, `HOME`, `TMPDIR`, proxy variables, locale variables) are typically inherited, and plugins SHOULD respect them as appropriate for their function. Other `CTX_*` variables MAY be added in future versions.

## 3. Output Format

Plugins **MUST** output their results to standard output as a single **JSON object**.

*   This JSON object **MUST** be the only output on standard output upon successful execution.
*   The JSON object **MUST** contain the following top-level fields:

    | Field     | Type   | Description                                                                                                                                                                                           | Required |
    | :-------- | :----- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------- |
    | `name`    | string | The logical name of the plugin (e.g., "git", "environment"). This **MUST** be unique across plugins used in a single `ctx` invocation, serving as the key in the aggregated output.                  | Yes      |
    | `version` | string | The version string of the plugin (e.g., "0.1.0"). Semantic Versioning 2.0.0 is RECOMMENDED.                                                                                                             | Yes      |
    | `data`    | object | An object containing the actual context data gathered by the plugin. The structure of this object is defined by the specific plugin. This field **MUST** be present, but MAY contain an empty object `{}`. | Yes      |

### 3.1 Optional Output Metadata

Plugins **MAY** include additional OPTIONAL top-level fields in their standard output JSON object to provide metadata back to `ctx` or downstream consumers.

#### 3.1.1 Metrics Object

Plugins MAY include a top-level `metrics` object.

*   **Field:** `metrics`
*   **Type:** object
*   **Description:** Contains quantitative measurements related to the plugin's execution or output. Standard field names are RECOMMENDED where applicable.
*   **Example Structure:**
    ```json
    "metrics": {
      "input_token_count": 50,    // Optional: Tokens consumed by plugin input/prompt
      "output_token_count": 450,   // Optional: Tokens in the plugin's 'data' output
      "thinking_token_count": 1200, // Optional: Tokens consumed by intermediate steps
      "cost_estimate_cents": 5,    // Optional: Estimated cost in USD cents
      "output_size_bytes": 1024    // Optional
      // Other relevant metrics...
    }
    ```
*   **Status:** Recommended (for relevant plugins)

#### 3.1.2 Data Schema Information

Plugins MAY provide information about the structure of their `data` field.

*   **Field:** `data_schema_url`
*   **Type:** string (URL)
*   **Description:** A URL pointing to a JSON Schema definition for the structure of the `data` field.
*   **Status:** Optional

*   **Field:** `data_schema`
*   **Type:** object (JSON Schema)
*   **Description:** An embedded JSON Schema definition for the structure of the `data` field.
*   **Status:** Optional

Plugins SHOULD provide at most one of `data_schema_url` or `data_schema`.

## 4. Error Handling

*   If a plugin encounters an error that prevents it from successfully gathering context and producing the REQUIRED JSON output, it **MUST** exit with a non-zero status code.
*   Plugins **SHOULD** print a descriptive error message to standard error upon failure.
*   Plugins **MUST NOT** print partial or invalid JSON to standard output on error. Standard output **MUST** be empty or contain only the single, valid JSON object defined in Section 3 upon successful (exit code 0) execution.

## 5. Incubating Conventions

The following concepts are under consideration but are **not yet stable** parts of the specification. They MAY change or be removed in future versions. Plugins implementing these do so on an experimental basis.

*   **User Approval Flow:** A mechanism where a plugin outputs a special `requires_approval` JSON object (instead of `data`) to signal `ctx` to prompt the user, potentially re-running the plugin with `CTX_APPROVED=true` set in the environment.
*   **Cache Information Metadata:** An optional `cache_info` top-level JSON object where plugins can provide hints about the cacheability of their data (e.g., `cacheable: true`, `ttl_seconds`).
*   **Capabilities Reporting / Plugin Self-Specification:** A potential mechanism (e.g., a `--ctx-spec` flag provided by the plugin) for plugins to report metadata about themselves (including supported `CTX_*` variables), their dependencies, or the structure of the context they provide (potentially using JSON Schema). This could be used by `ctx` for future planning logic or validation.
*   **Tool Approval Mechanism:** A potential system (likely configured within `ctx`, possibly respecting `CTX_ALLOWED_TOOLS`) to approve or deny the execution of specific plugins or external tools called by plugins, potentially based on name (regex), version constraints, and/or SHA256 hash verification. Approvals could be global or tied to the `CTX_SESSION`.
*   **Plugin Integrity and Provenance:** Concepts for ensuring the trustworthiness of plugins before execution:
    *   *Code Signing:* Requiring plugins to be cryptographically signed by a trusted authority.
    *   *SHA Verification:* `ctx` configuration could include expected SHA256 hashes for known plugin versions, preventing execution if the binary doesn't match.
    *   *Certificate Transparency Log / Binary Transparency:* Potential use of public logs to record plugin hashes/signing certificates, allowing `ctx` or users to verify provenance and detect tampering or unexpected binaries.
*   **Anonymous Usage Reporting:** An OPTIONAL mechanism where `ctx` or plugins could report anonymized usage data (e.g., plugin name + version + SHA256 hash used in `CTX_SESSION`) to a central service for security monitoring or usage statistics. This would require explicit opt-in and clear privacy policies.

## 6. Specification Stability

This is version `0.1.0` of the specification. Future versions MAY introduce changes, potentially including new REQUIRED fields or behaviors. Compatibility considerations will be outlined in future specification versions.

