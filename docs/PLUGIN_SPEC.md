# ctx Plugin Specification v0.1.0

This specification defines the requirements and conventions for executables acting as plugins for the `ctx` context-gathering tool. Adherence to the mandatory requirements ensures basic compatibility and interoperability.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

## 1. Naming Convention

Plugins discovered via the system PATH **MUST** be named with the prefix `ctx-` followed by a descriptive name (e.g., `ctx-git`, `ctx-env`).

## 2. Execution

The `ctx` tool executes discovered plugins as child processes.

*   When executed **without arguments**, a plugin **MUST** perform its primary context gathering function and print its result to standard output according to the format defined in Section 3.
*   Plugins **SHOULD** generally be robust against receiving unknown flags or arguments, perhaps by ignoring them or printing a warning to standard error.
*   Plugins **SHOULD** use standard error for diagnostic messages, logs, or warnings not intended for context aggregation.

### 2.1 Standard Environment Variables

Plugins **MUST** expect that their execution environment will be largely inherited from the `ctx` process. In addition, `ctx` **MAY** explicitly set or propagate the following environment variables. Plugins **SHOULD** check for these variables if their behavior might be affected.

| Variable                       | Set/Propagated By `ctx`? | Description                                                                                                 | Notes                                                                                                        |
| :----------------------------- | :----------------------- | :---------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------- |
| `CTX_SESSION`                  | Yes (Generated/Propagated)| An opaque string ID (ULID by default) unique to the current `ctx` invocation. Useful for correlation/logging. | `ctx` ensures this is set.                                                                                   |
| `CTX_CACHE_DIR`                | Yes (Based on Flag/XDG)  | Specifies the base directory plugins SHOULD use for caching.                                                | Set if `--cache-dir` used or XDG dir found. Plugins MAY disable caching if unset/unusable.             |
| `CTX_OUTPUT_TOKEN_BUDGET`      | Yes (Based on Flag)      | Informs the plugin of an estimated token budget (integer) for its primary output (`data` field).            | Set if `--output-token-budget > 0`. Interpretation of "token" is context-dependent.                       |
| `CTX_THINKING_TOKEN_BUDGET`    | Yes (Based on Flag)      | Informs the plugin of an estimated token budget (integer) for internal reasoning or intermediate steps.   | Set if `--thinking-token-budget > 0`. Useful for plugins that consume tokens internally.                   |
| `CTX_COST_BUDGET_CENTS`        | Yes (Based on Flag)      | Informs the plugin of an estimated cost budget in **USD cents** (integer).                                | Set if `--cost-budget > 0`. Plugin MAY attempt to stay within budget.                                      |
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

## 4. Error Handling

*   If a plugin encounters an error that prevents it from successfully gathering context and producing the REQUIRED JSON output, it **MUST** exit with a non-zero status code.
*   Plugins **SHOULD** print a descriptive error message to standard error upon failure.
*   Plugins **MUST NOT** print partial or invalid JSON to standard output on error. Standard output **MUST** be empty or contain only the single, valid JSON object defined in Section 3 upon successful (exit code 0) execution.

## 5. Incubating Conventions

The following concepts are under consideration but are **not yet stable** parts of the specification. They MAY change or be removed in future versions. Plugins implementing these do so on an experimental basis.

*   **User Approval Flow:** A mechanism where a plugin outputs a special `requires_approval` JSON object (instead of `data`) to signal `ctx` to prompt the user, potentially re-running the plugin with `CTX_APPROVED=true` set in the environment.
*   **Cache Information Metadata:** An optional `cache_info` top-level JSON object where plugins can provide hints about the cacheability of their data (e.g., `cacheable: true`, `ttl_seconds`).
*   **Capabilities Reporting:** A potential mechanism (e.g., a `--capabilities` flag) for plugins to report metadata about themselves, their dependencies, or the context they provide, possibly used by `ctx` for future planning logic.

## 6. Specification Stability

This is version `0.1.0` of the specification. Future versions MAY introduce changes, potentially including new REQUIRED fields or behaviors. Compatibility considerations will be outlined in future specification versions.

