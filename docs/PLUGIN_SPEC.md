# ctx Plugin Specification

This document outlines the core requirements for ctx plugins.

## Naming
- Prefix: `ctx-`
- Example: `ctx-git`, `ctx-docker`

## Execution
- No arguments for default operation
- Support `--capabilities` and `--plan-relevance` flags

## Output
- Flexible output format.

## Error Handling
- Non-zero exit code for errors
- Error message to stderr
- JSON error output:
  ```json
  {
    "error": "Error message"
  }
  ```

## Capabilities
- `--capabilities` flag
- JSON output with supported features

## Plan Relevance
- `--plan-relevance` flag
- Output: float between 0 and 1

## Versioning
- Semantic versioning (MAJOR.MINOR.PATCH)
- Include `ctx_version` in capabilities output

## Configuration
- Support environment variables
- Follow XDG Base Directory Specification

## Security
- Validate all input
- Avoid executing user-supplied code
- Use secure coding practices

## Performance
- Implement caching for expensive operations
- Minimize external process calls

## Documentation
- README with installation and usage instructions
- CHANGELOG for version changes

Plugins adhering to this specification will be compatible with the ctx tool and provide consistent, reliable context information.
