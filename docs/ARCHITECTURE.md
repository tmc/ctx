# Architecture

## Execution Model

ctx follows a three-phase execution model:
1. Discovery Phase
2. Planning Phase
3. Execution Phase

### Discovery Phase

During the discovery phase, ctx performs the following actions:
- Searches for plugins with the `ctx-` prefix in the system PATH
- Identifies available plugins and their capabilities
- Gathers initial environment information

### Planning Phase

The planning phase involves:
- Analyzing the discovered plugins and their reported capabilities
- Considering any user-provided plan hints or explicit plans
- Determining the optimal order of plugin execution
- Generating a comprehensive plan for context gathering

### Execution Phase

In the execution phase, ctx:
- Follows the plan generated in the planning phase
- Executes each plugin in the determined order
- Collects and aggregates the context information from all plugins
- Formats the output according to user preferences (JSON, YAML, etc.)
- Handles any errors or exceptions that occur during plugin execution

## Key Components

1. Plugin Manager: Responsible for discovering, loading, and managing plugins
2. Planner: Determines the execution plan based on available plugins and user input
3. Executor: Runs the plugins according to the plan and collects their output
4. Output Formatter: Processes the collected data into the desired output format

## Data Flow

1. User input → Discovery Phase
2. Discovery Phase output → Planning Phase
3. Planning Phase output → Execution Phase
4. Execution Phase output → Output Formatter
5. Formatted output → User (likely going to a language model)

## Error Handling

- Each phase has its own error handling mechanisms
- Plugins are expected to handle their own errors and report them appropriately
- The main ctx process aggregates and reports errors from all phases and plugins

## Extensibility

The architecture is designed for easy extensibility:
- New plugins can be added without modifying the core ctx code
- The planning phase can be customized or replaced entirely
- Output formats can be extended to support new types

## Scalability and Performance Considerations

- Implement parallel plugin execution
- Use efficient data structures for large datasets
- Implement caching mechanisms
- Consider distributed architecture for large-scale deployments

# Future Topics

- Evaluation harness for testing and benchmarking plugins.
- Caching and optimization strategies.
- Plugin development guidelines.
- Security considerations (e.g., plugin sandboxing, data protection, code signing, plugin
    permissions, etc.).
- Integration with third-party tools and services.
- Implement token budget management for efficient LLM usage
- Implement advanced context relevance scoring
- Implement evaluation metrics for gathered context quality
- Consider a plugin management subcommand for installing, updating, and removing plugins
- Consider a plugin repository for sharing and discovering plugins
- Implement a plugin development kit with templates, examples, and documentation
