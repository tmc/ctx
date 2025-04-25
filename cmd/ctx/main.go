package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug" // For build info
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tmc/ctx/docs"
	"sigs.k8s.io/yaml"
)

const ctxSessionEnvKey = "CTX_SESSION"
const ctxShlvlEnvKey = "CTX_SHLVL"
const ctxShowSourceEnvKey = "CTX_SHOW_SOURCE" // Always implies txtar format

// Environment variables to propagate if set in ctx's environment
var ambientEnvKeysToPropagate = []string{
	"TRACEPARENT", // OpenTelemetry Trace Context
	"TRACESTATE",  // OpenTelemetry Trace Context
	// Add other standard ambient variables here as needed
}

// Configuration struct to hold parsed flags
type config struct {
	outputFormat        string
	listPlugins         bool
	showVersion         bool
	printSpec           bool
	actAsPlugin         bool // Act as a ctx-* plugin itself
	cacheDir            string
	outputTokenBudget   int
	thinkingTokenBudget int
	costBudgetCents     int
	allowedTools        string
	pluginTimeout       time.Duration
	pluginRetries       int
	indent              int
	summary             bool
	maxParallelPlugins  int  // Maximum number of plugins to run in parallel
	printSource         bool // Print plugin source when available (always in txtar format)
	verbose             bool // Enable verbose logging
}

// PluginData defines the expected JSON structure from plugins.
type PluginData struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"` // Keep data as raw JSON initially
}

// XML structure for aggregated output
type XMLResults struct {
	XMLName   xml.Name    `xml:"ctx_results"`
	SessionID string      `xml:"session_id,attr"`
	Plugins   []XMLPlugin `xml:"plugin"`
}

type XMLPlugin struct {
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr,omitempty"`
	// Embed data as marshaled JSON within a CDATA section or similar
	Data xml.CharData `xml:"data"`
}

// handleHelpFlag checks if -h, -help, or --help flags are present and exits with code 1
func handleHelpFlag() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "-help" || arg == "--help" {
			fmt.Fprintf(os.Stderr, "ctx - Context Gathering Tool\n\n")
			flag.Usage()
			os.Exit(1) // Exit with non-zero status per plugin spec
		}
	}
}

// runAsPlugin outputs the ctx tool's metadata in plugin format
func runAsPlugin() error {
	// Gather all environment variables starting with CTX_
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "CTX_") {
			env[parts[0]] = parts[1]
		}
	}
	
	// Create a simple data structure
	data := map[string]interface{}{
		"environment": env,
		"description": "Core ctx metadata",
	}
	
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin metadata: %w", err)
	}
	
	// Base plugin response
	pluginData := PluginData{
		Name:    "ctx",
		Version: getVersion(),
		Data:    dataBytes,
	}
	
	jsonBytes, err := json.Marshal(pluginData)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin data: %w", err)
	}
	
	fmt.Println(string(jsonBytes))
	return nil
}

// Verbose output mode
var verbose bool

// Custom logger that only prints when verbose mode is enabled
type verboseLogger struct{}

func (l *verboseLogger) Write(p []byte) (n int, err error) {
	if verbose {
		return os.Stderr.Write(p)
	}
	return len(p), nil
}

func main() {
	// Set up custom logger that respects verbose mode
	log.SetFlags(0)
	log.SetOutput(&verboseLogger{})
	
	// Check for help flag first (for plugin spec compliance)
	handleHelpFlag()
	
	cfg := parseFlags()
	verbose = cfg.verbose

	if cfg.showVersion {
		fmt.Println(getVersion()) // Use dynamic version info
		os.Exit(0)
	}

	if cfg.printSpec {
		// Print the embedded spec content (loaded during init) and exit
		specContent, err := docs.All.ReadFile("PLUGIN_SPEC.md")
		if err != nil {
			log.Fatalf("Error reading embedded spec file: %v", err)
		}
		fmt.Println(string(specContent))
		os.Exit(0)
	}
	
	if cfg.actAsPlugin {
		if err := runAsPlugin(); err != nil {
			log.Fatalf("Error running as plugin: %v", err)
		}
		return
	}

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() *config {
	// Initialize with defaults
	cfg := &config{
		outputFormat:       "yaml",
		maxParallelPlugins: 1,
		indent:             2,
	}
	
	// Define CLI flags
	flag.StringVar(&cfg.outputFormat, "output", cfg.outputFormat, "Output format (yaml, json, xml)")
	flag.BoolVar(&cfg.listPlugins, "list-plugins", cfg.listPlugins, "List discovered plugins and exit")
	flag.BoolVar(&cfg.showVersion, "version", cfg.showVersion, "Show version and build information")
	flag.BoolVar(&cfg.printSpec, "print-spec", cfg.printSpec, "Print the plugin specification to stdout and exit")
	flag.BoolVar(&cfg.actAsPlugin, "plugin", cfg.actAsPlugin, "Act as a ctx-* plugin itself and output JSON according to the plugin spec")
	flag.StringVar(&cfg.cacheDir, "cache-dir", cfg.cacheDir, "Specify a base directory for plugins to use for caching (sets CTX_CACHE_DIR). Uses XDG default if empty.")
	flag.IntVar(&cfg.outputTokenBudget, "output-token-budget", cfg.outputTokenBudget, "Inform plugins of an estimated token budget for output (sets CTX_OUTPUT_TOKEN_BUDGET, 0 means unset)")
	flag.IntVar(&cfg.thinkingTokenBudget, "thinking-token-budget", cfg.thinkingTokenBudget, "Inform plugins of an estimated token budget for internal work (sets CTX_THINKING_TOKEN_BUDGET, 0 means unset)")
	flag.IntVar(&cfg.costBudgetCents, "cost-budget", cfg.costBudgetCents, "Inform plugins of an estimated cost budget in USD cents (sets CTX_COST_BUDGET_CENTS, 0 means unset)")
	flag.StringVar(&cfg.allowedTools, "allowed-tools", cfg.allowedTools, "Comma-separated list of external commands plugins are permitted to call (sets CTX_ALLOWED_TOOLS)")
	flag.DurationVar(&cfg.pluginTimeout, "plugin-timeout", cfg.pluginTimeout, "Suggests a timeout duration for individual plugin execution (e.g., '30s', '1m'). Sets related CTX_* env vars. 0 means unset.")
	flag.IntVar(&cfg.pluginRetries, "plugin-retries", cfg.pluginRetries, "Suggests a maximum number of retries plugins might attempt (sets CTX_RETRY_MAX, 0 means unset).")
	flag.IntVar(&cfg.indent, "indent", cfg.indent, "Number of spaces for JSON/XML output indentation.")
	flag.BoolVar(&cfg.summary, "summary", cfg.summary, "Output compact JSON/XML without indentation (overrides --indent).")
	flag.IntVar(&cfg.maxParallelPlugins, "P", cfg.maxParallelPlugins, "Maximum number of plugins to run in parallel. Default is 1 for safety.")
	flag.IntVar(&cfg.maxParallelPlugins, "parallel", cfg.maxParallelPlugins, "Maximum number of plugins to run in parallel. Default is 1 for safety.")
	flag.BoolVar(&cfg.printSource, "show-source", false, "Request plugins to include their source code in txtar format (sets CTX_SHOW_SOURCE=true)")
	flag.BoolVar(&cfg.verbose, "v", false, "Enable verbose output for debugging")

	flag.Parse()
	return cfg
}

// getVersion attempts to read build info. Requires Go 1.18+.
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)" // Not built with module info
	}

	version := info.Main.Version
	if version == "" || version == "(devel)" {
		// Fallback to VCS info if main version isn't helpful
		var revision string
		var modTime string
		var modified bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.time":
				modTime = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}
		if revision != "" {
			shortRev := revision
			if len(shortRev) > 7 {
				shortRev = shortRev[:7]
			}
			version = fmt.Sprintf("vcs-%s", shortRev)
			if modTime != "" {
				// Basic format adjustment for readability
				if t, err := time.Parse(time.RFC3339Nano, modTime); err == nil {
					modTime = t.Format("20060102T150405Z")
				}
				version += "-" + modTime
			}
			if modified {
				version += "-modified"
			}
			return version
		}
		return "(devel)" // Still couldn't find useful info
	}
	return version
}

func run(cfg *config) error {
	// --- Plugin Discovery ---
	if verbose {
		log.Println("Discovering plugins in PATH...")
	}
	discoveredPlugins, err := findPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}
	if verbose {
		if len(discoveredPlugins) == 0 {
			log.Println("No ctx-* plugins found in PATH.")
		} else {
			log.Printf("Found %d potential plugin(s).", len(discoveredPlugins))
		}
	}

	if cfg.listPlugins {
		fmt.Println("Discovered potential plugins (executables named ctx-* in PATH):")
		if len(discoveredPlugins) == 0 {
			fmt.Println("  (None found)")
		}
		for _, p := range discoveredPlugins {
			fmt.Printf("  - %s\n", p) // Show full path for clarity
		}
		return nil
	}

	if len(discoveredPlugins) == 0 {
		if verbose {
			log.Println("No plugins found to execute.")
		}
		emptyResult := map[string]PluginData{}
		emptyOutput, _ := formatOutput(emptyResult, cfg) // Use empty map
		fmt.Println(emptyOutput)
		return nil
	}

	// --- Plugin Execution ---
	if verbose {
		log.Printf("Executing %d potential plugin(s)...", len(discoveredPlugins))
	}
	sessionID := getSessionID()
	pluginEnv := getPluginEnv(cfg, sessionID, ambientEnvKeysToPropagate)
	if verbose {
		log.Printf("Session ID: %s", sessionID)
	}

	// Create execution context
	var execCtx context.Context
	var cancel context.CancelFunc
	
	if cfg.pluginTimeout > 0 {
		execCtx, cancel = context.WithTimeout(context.Background(), cfg.pluginTimeout)
		defer cancel()
	} else {
		execCtx = context.Background()
	}

	results := executePlugins(execCtx, discoveredPlugins, pluginEnv, cfg.maxParallelPlugins)
	if verbose {
		log.Printf("Finished execution. Aggregated results from %d plugin(s).", len(results))
	}

	// --- Output Formatting ---
	output, err := formatOutput(results, cfg) // Pass full config
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Println(output)
	return nil
}

// findPlugins searches PATH for executables starting with "ctx-".
// Returns a list of full paths to potential plugins.
func findPlugins() ([]string, error) {
	var plugins []string
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, errors.New("PATH environment variable is not set")
	}
	paths := filepath.SplitList(pathEnv)

	checked := make(map[string]struct{})
	selfPath, _ := os.Executable() // Get our own path to ensure we don't create infinite loop

	for _, path := range paths {
		if path == "" {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, ok := checked[absPath]; ok {
			continue
		}
		checked[absPath] = struct{}{}

		files, err := os.ReadDir(absPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			
			fileName := file.Name()
			if !strings.HasPrefix(fileName, "ctx-") {
				continue
			}
			
			pluginPath := filepath.Join(absPath, fileName)
			
			// Skip ourselves to avoid infinite recursion
			if pluginPath == selfPath {
				continue
			}
			
			info, err := file.Info()
			if err != nil || !(info.Mode()&0111 != 0 || runtime.GOOS == "windows") {
				continue
			}
			
			plugins = append(plugins, pluginPath)
		}
	}
	return plugins, nil
}

// getSessionID retrieves the session ID from the environment or generates a new timestamp-based ID.
func getSessionID() string {
	if sessionID := os.Getenv(ctxSessionEnvKey); sessionID != "" {
		return sessionID
	}
	return fmt.Sprintf("ctx_%d", time.Now().Unix())
}

// getCurrentShlvl reads the current CTX_SHLVL or SHLVL, defaulting to 0.
func getCurrentShlvl() int {
	levelStr := os.Getenv(ctxShlvlEnvKey)
	if levelStr == "" {
		levelStr = os.Getenv("SHLVL")
	}
	level, err := strconv.Atoi(levelStr)
	if err != nil || level < 0 {
		return 0
	}
	return level
}

// getPluginEnv creates the environment slice for plugin execution.
func getPluginEnv(cfg *config, sessionID string, ambientKeysToPropagate []string) []string {
	currentEnv := os.Environ()
	finalEnv := make([]string, 0, len(currentEnv)+2+len(ambientKeysToPropagate)+8)

	varsToSet := make(map[string]string)
	managedKeys := make(map[string]struct{})

	// 1. Set CTX_SESSION
	varsToSet[ctxSessionEnvKey] = sessionID
	managedKeys[ctxSessionEnvKey] = struct{}{}

	// 2. Set CTX_SHLVL
	currentLevel := getCurrentShlvl()
	varsToSet[ctxShlvlEnvKey] = strconv.Itoa(currentLevel + 1)
	managedKeys[ctxShlvlEnvKey] = struct{}{}
	managedKeys["SHLVL"] = struct{}{} // Don't inherit standard SHLVL

	// 3. Handle ambient context propagation (Otel etc)
	for _, key := range ambientKeysToPropagate {
		if value, exists := os.LookupEnv(key); exists {
			varsToSet[key] = value
		}
		managedKeys[key] = struct{}{}
	}

	// 4. Handle configuration propagation from flags
	var cacheDirToSet string
	if cfg.cacheDir != "" {
		absCacheDir, err := filepath.Abs(cfg.cacheDir)
		if err == nil {
			cacheDirToSet = absCacheDir
		} else {
			log.Printf("Warning: Could not resolve absolute path for --cache-dir '%s'. CTX_CACHE_DIR will not be set.", cfg.cacheDir)
		}
	} else {
		xdgCacheHome := os.Getenv("XDG_CACHE_HOME")
		if xdgCacheHome == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				xdgCacheHome = filepath.Join(homeDir, ".cache")
			}
		}
		if xdgCacheHome != "" {
			cacheDirToSet = filepath.Join(xdgCacheHome, "ctx")
		}
	}
	if cacheDirToSet != "" {
		varsToSet["CTX_CACHE_DIR"] = cacheDirToSet
		managedKeys["CTX_CACHE_DIR"] = struct{}{}
	}

	if cfg.outputTokenBudget > 0 {
		varsToSet["CTX_OUTPUT_TOKEN_BUDGET"] = strconv.Itoa(cfg.outputTokenBudget)
		managedKeys["CTX_OUTPUT_TOKEN_BUDGET"] = struct{}{}
	}
	if cfg.thinkingTokenBudget > 0 {
		varsToSet["CTX_THINKING_TOKEN_BUDGET"] = strconv.Itoa(cfg.thinkingTokenBudget)
		managedKeys["CTX_THINKING_TOKEN_BUDGET"] = struct{}{}
	}
	if cfg.costBudgetCents > 0 {
		varsToSet["CTX_COST_BUDGET_CENTS"] = strconv.Itoa(cfg.costBudgetCents)
		managedKeys["CTX_COST_BUDGET_CENTS"] = struct{}{}
	}
	if cfg.allowedTools != "" {
		varsToSet["CTX_ALLOWED_TOOLS"] = cfg.allowedTools
		managedKeys["CTX_ALLOWED_TOOLS"] = struct{}{}
	}
	if cfg.pluginTimeout > 0 {
		timeoutSec := int(cfg.pluginTimeout.Seconds())
		if timeoutSec > 0 {
			deadlineTs := time.Now().Add(cfg.pluginTimeout).Unix()
			varsToSet["CTX_TIMEOUT_SECONDS"] = strconv.Itoa(timeoutSec)
			varsToSet["CTX_DEADLINE_TIMESTAMP"] = strconv.FormatInt(deadlineTs, 10)
			managedKeys["CTX_TIMEOUT_SECONDS"] = struct{}{}
			managedKeys["CTX_DEADLINE_TIMESTAMP"] = struct{}{}
		}
	}
	if cfg.pluginRetries > 0 {
		varsToSet["CTX_RETRY_MAX"] = strconv.Itoa(cfg.pluginRetries)
		managedKeys["CTX_RETRY_MAX"] = struct{}{}
	}
	
	// Set source flag if enabled
	if cfg.printSource {
		// When showing source, it's always in txtar format
		varsToSet[ctxShowSourceEnvKey] = "true"
		managedKeys[ctxShowSourceEnvKey] = struct{}{}
	}
	
	// managedKeys["CTX_APPROVED"] = struct{}{} // If approval flow implemented

	// Filter currentEnv, keeping only non-managed vars
	for _, envVar := range currentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) > 0 {
			if _, manageThis := managedKeys[parts[0]]; !manageThis {
				finalEnv = append(finalEnv, envVar)
			}
		}
	}
	// Add the explicitly managed vars
	for key, value := range varsToSet {
		finalEnv = append(finalEnv, fmt.Sprintf("%s=%s", key, value))
	}
	return finalEnv
}

// executePlugins runs discovered plugins concurrently and aggregates their JSON output.
func executePlugins(ctx context.Context, pluginPaths []string, pluginEnv []string, maxParallel int) map[string]PluginData {
	results := make(map[string]PluginData)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	// Semaphore to limit concurrent executions
	semaphore := make(chan struct{}, maxParallel)

	for _, pluginPath := range pluginPaths {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore token
		go func(pPath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore token
			execName := filepath.Base(pPath)

			// Skip the help flag check - we're more lenient now
			if verbose {
				log.Printf("[%s] Running plugin...", execName)
			}
			
			cmd := exec.CommandContext(ctx, pPath)
			cmd.Env = pluginEnv
			stdout, err := cmd.Output()

			if err != nil {
				if verbose {
					errMsg := fmt.Sprintf("failed to execute plugin '%s': %v", execName, err)
					var exitErr *exec.ExitError
					if errors.As(err, &exitErr) {
						errMsg = fmt.Sprintf("%s. Stderr: %s", errMsg, string(exitErr.Stderr))
					}
					log.Printf("[%s] Error: %s", execName, errMsg)
				}
				return
			}

			var data PluginData
			if err := json.Unmarshal(stdout, &data); err != nil {
				if verbose {
					const maxLogLen = 200
					outStr := string(stdout)
					if len(outStr) > maxLogLen {
						outStr = outStr[:maxLogLen] + "..."
					}
					log.Printf("[%s] Error: failed parsing JSON output: %v. Output (truncated): %s", execName, err, outStr)
				}
				return
			}

			if data.Name == "" || data.Version == "" || data.Data == nil {
				if verbose {
					log.Printf("[%s] Error: Plugin output missing required field ('name', 'version', or 'data'). Skipping.", execName)
				}
				return
			}

			if verbose {
				log.Printf("[%s] Success (Reported Version: %s).", data.Name, data.Version)
			}

			mu.Lock()
			defer mu.Unlock()
			if _, exists := results[data.Name]; exists && verbose {
				log.Printf("Warning: Duplicate plugin name '%s' detected (from '%s'). Overwriting previous result.", data.Name, execName)
			}
			results[data.Name] = data

		}(pluginPath)
	}

	wg.Wait()
	return results
}

// formatOutput converts the aggregated results to the desired string format.
func formatOutput(results map[string]PluginData, cfg *config) (string, error) {
	outputData := make(map[string]any, len(results))
	pluginMetas := make(map[string]PluginData) // Store original meta for XML
	for name, result := range results {
		pluginMetas[name] = result // Keep version etc.
		var data any
		if err := json.Unmarshal(result.Data, &data); err != nil {
			log.Printf("Warning: Could not unmarshal stored raw data for plugin '%s', outputting as string: %v", name, err)
			outputData[name] = string(result.Data) // Fallback
		} else {
			outputData[name] = data
		}
	}

	var outputBytes []byte
	var err error
	outputFormat := strings.ToLower(cfg.outputFormat)
	indentStr := ""
	prefixStr := ""
	if !cfg.summary && cfg.indent > 0 {
		indentStr = strings.Repeat(" ", cfg.indent)
		prefixStr = "" // xml/json handle prefix internally via indent
	}

	switch outputFormat {
	case "json":
		if indentStr == "" {
			outputBytes, err = json.Marshal(outputData)
		} else {
			outputBytes, err = json.MarshalIndent(outputData, prefixStr, indentStr)
		}
		if err != nil {
			return "", fmt.Errorf("failed to marshal results to JSON: %w", err)
		}
	case "xml":
		xmlRoot := XMLResults{SessionID: os.Getenv(ctxSessionEnvKey)} // Use the current session ID
		for name, data := range outputData {
			jsonDataBytes, jsonErr := json.Marshal(data) // Marshal just the data part
			if jsonErr != nil {
				log.Printf("Warning: Could not re-marshal data for plugin '%s' to JSON for XML embedding: %v", name, jsonErr)
				jsonDataBytes = []byte("Error re-marshaling data")
			}
			meta := pluginMetas[name] // Get original metadata
			xmlPlugin := XMLPlugin{Name: meta.Name, Version: meta.Version, Data: xml.CharData(jsonDataBytes)}
			xmlRoot.Plugins = append(xmlRoot.Plugins, xmlPlugin)
		}
		if indentStr == "" {
			outputBytes, err = xml.Marshal(xmlRoot)
		} else {
			outputBytes, err = xml.MarshalIndent(xmlRoot, prefixStr, indentStr)
		}
		if err != nil {
			return "", fmt.Errorf("failed to marshal results to XML: %w", err)
		}
		// Add XML header manually if needed, MarshalIndent doesn't include it.
		outputBytes = append([]byte(xml.Header), outputBytes...)

	case "yaml":
		fallthrough
	default: // Default to YAML
		// YAML marshaller doesn't support indentation control easily in the standard lib
		outputBytes, err = yaml.Marshal(outputData)
		if err != nil {
			return "", fmt.Errorf("failed to marshal results to YAML: %w", err)
		}
	}

	// Add trailing newline if not empty and not already present
	if len(outputBytes) > 0 && !bytes.HasSuffix(outputBytes, []byte("\n")) {
		return string(outputBytes) + "\n", nil
	}
	return string(outputBytes), nil
}
