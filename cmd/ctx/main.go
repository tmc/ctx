package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"sigs.k8s.io/yaml"
)

// Populated by build flags.
var version = "dev"

// Environment variables to propagate if set in ctx's environment
var ambientEnvKeysToPropagate = []string{
	"TRACEPARENT", // OpenTelemetry Trace Context
	"TRACESTATE",  // OpenTelemetry Trace Context
	// No MCP_SESSION here anymore, CTX_SESSION replaces it
}

const ctxSessionEnvKey = "CTX_SESSION"

// Configuration struct to hold parsed flags
type config struct {
	outputFormat        string
	listPlugins         bool
	showVersion         bool
	cacheDir            string
	outputTokenBudget   int
	thinkingTokenBudget int
	costBudgetCents     int
}

// PluginData defines the expected JSON structure from plugins.
type PluginData struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"` // Keep data as raw JSON initially
}

func main() {
	log.SetFlags(0) // Basic logging format
	cfg := parseFlags()

	if cfg.showVersion {
		fmt.Printf("ctx version %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() *config {
	cfg := &config{}

	// --- Configuration & Flags ---
	flag.StringVar(&cfg.outputFormat, "output", "yaml", "Output format (yaml, json)")
	flag.BoolVar(&cfg.listPlugins, "list-plugins", false, "List discovered plugins and exit")
	flag.BoolVar(&cfg.showVersion, "version", false, "Show version and build information")
	flag.StringVar(&cfg.cacheDir, "cache-dir", "", "Specify a base directory for plugins to use for caching (sets CTX_CACHE_DIR). Uses XDG default if empty.")
	flag.IntVar(&cfg.outputTokenBudget, "output-token-budget", 0, "Inform plugins of an estimated token budget for output (sets CTX_OUTPUT_TOKEN_BUDGET, 0 means unset)")
	flag.IntVar(&cfg.thinkingTokenBudget, "thinking-token-budget", 0, "Inform plugins of an estimated token budget for internal work (sets CTX_THINKING_TOKEN_BUDGET, 0 means unset)")
	flag.IntVar(&cfg.costBudgetCents, "cost-budget", 0, "Inform plugins of an estimated cost budget in USD cents (sets CTX_COST_BUDGET_CENTS, 0 means unset)")

	flag.Parse()
	return cfg
}


func run(cfg *config) error {
	// --- Plugin Discovery ---
	log.Println("Discovering plugins in PATH...")
	discoveredPlugins, err := findPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}
	if len(discoveredPlugins) == 0 {
		log.Println("No ctx-* plugins found in PATH.")
	} else {
		log.Printf("Found %d potential plugin(s).", len(discoveredPlugins))
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
		log.Println("No plugins found to execute.")
		fmt.Println("{}") // YAML for empty map
		return nil
	}

	// --- Plugin Execution ---
	log.Printf("Executing %d potential plugin(s)...", len(discoveredPlugins))
	// Prepare environment once for all plugins
	sessionID := getSessionID()
	pluginEnv := getPluginEnv(cfg, sessionID, ambientEnvKeysToPropagate)
	log.Printf("Session ID: %s", sessionID) // Log the session ID being used

	// TODO: Implement context with timeout
	results := executePlugins(context.Background(), discoveredPlugins, pluginEnv)
	log.Printf("Finished execution. Aggregated results from %d plugin(s).", len(results))


	// --- Output Formatting ---
	output, err := formatOutput(results, cfg.outputFormat)
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

	checked := make(map[string]struct{}) // Avoid checking the same directory multiple times

	for _, path := range paths {
		if path == "" {	continue }
		absPath, err := filepath.Abs(path)
		if err != nil { continue } // Silently ignore paths that cannot be resolved
		if _, ok := checked[absPath]; ok { continue }
		checked[absPath] = struct{}{}

		files, err := os.ReadDir(absPath)
		if err != nil { continue } // Silently ignore errors reading directories

		for _, file := range files {
			if !file.IsDir() && strings.HasPrefix(file.Name(), "ctx-") {
				pluginPath := filepath.Join(absPath, file.Name())
				info, err := file.Info()
				if err == nil && (info.Mode()&0111 != 0 || runtime.GOOS == "windows") {
					plugins = append(plugins, pluginPath)
				}
			}
		}
	}
	return plugins, nil
}

// getSessionID retrieves the session ID from the environment or generates a new ULID.
func getSessionID() string {
	if sessionID := os.Getenv(ctxSessionEnvKey); sessionID != "" {
		// TODO: Validate format? For now, just propagate if set.
		return sessionID
	}
	// Generate a new ULID if not present
	entropy := ulid.Monotonic(rand.Reader, 0)
	newID := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return newID.String()
}


// getPluginEnv creates the environment slice for plugin execution,
// inheriting the current environment, propagating ambient context vars,
// setting the session ID, and adding configuration variables based on ctx flags.
func getPluginEnv(cfg *config, sessionID string, ambientKeysToPropagate []string) []string {
	currentEnv := os.Environ()
	// Estimate slice size: current + managed session + potential ambient + potential config
	finalEnv := make([]string, 0, len(currentEnv)+1+len(ambientKeysToPropagate)+4)

	// Variables to explicitly set/overwrite
	varsToSet := make(map[string]string)
	managedKeys := make(map[string]struct{}) // Keep track of keys we are managing

	// 1. Set CTX_SESSION (always managed)
	varsToSet[ctxSessionEnvKey] = sessionID
	managedKeys[ctxSessionEnvKey] = struct{}{}

	// 2. Handle ambient context propagation
	for _, key := range ambientKeysToPropagate {
		if value, exists := os.LookupEnv(key); exists {
			varsToSet[key] = value
		}
		managedKeys[key] = struct{}{} // Mark as managed even if not set, to prevent inheritance
	}

	// 3. Handle configuration propagation from flags
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
	// managedKeys["CTX_APPROVED"] = struct{}{} // If approval flow implemented

	// Filter out existing variables from currentEnv that we intend to manage/set explicitly
	for _, envVar := range currentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) > 0 {
			if _, manageThis := managedKeys[parts[0]]; !manageThis {
				finalEnv = append(finalEnv, envVar) // Inherit if not managed
			}
		}
	}

	// Add the explicitly managed variables
	for key, value := range varsToSet {
		finalEnv = append(finalEnv, fmt.Sprintf("%s=%s", key, value))
	}

	return finalEnv
}


// executePlugins runs discovered plugins concurrently and aggregates their JSON output.
// Returns a map where keys are plugin names (from JSON output) and values are the plugin's data field.
func executePlugins(ctx context.Context, pluginPaths []string, pluginEnv []string) map[string]json.RawMessage {
	results := make(map[string]json.RawMessage)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, pluginPath := range pluginPaths {
		wg.Add(1)
		go func(pPath string) {
			defer wg.Done()
			execName := filepath.Base(pPath)
			log.Printf("Running plugin: %s...", execName)

			cmd := exec.CommandContext(ctx, pPath)
			cmd.Env = pluginEnv // Set the calculated environment
			stdout, err := cmd.Output()

			if err != nil {
				errMsg := fmt.Sprintf("failed to execute plugin '%s': %v", execName, err)
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					errMsg = fmt.Sprintf("%s. Stderr: %s", errMsg, string(exitErr.Stderr))
				}
				log.Printf("Error: %s", errMsg)
				return
			}

			var data PluginData
			if err := json.Unmarshal(stdout, &data); err != nil {
				const maxLogLen = 200
				outStr := string(stdout)
				if len(outStr) > maxLogLen { outStr = outStr[:maxLogLen] + "..." }
				log.Printf("Error: failed parsing JSON output from plugin '%s': %v. Output (truncated): %s", execName, err, outStr)
				return
			}

			if data.Name == "" || data.Version == "" || data.Data == nil {
				log.Printf("Error: Plugin '%s' output missing required field ('name', 'version', or 'data'). Skipping.", execName)
				return
			}

			log.Printf("Success: Plugin '%s' (reported version %s) executed.", data.Name, data.Version)

			mu.Lock()
			defer mu.Unlock()
			if _, exists := results[data.Name]; exists {
				log.Printf("Warning: Duplicate plugin name '%s' detected (from '%s'). Overwriting previous result.", data.Name, execName)
			}
			results[data.Name] = data.Data

		}(pluginPath)
	}

	wg.Wait()
	return results
}

// formatOutput converts the aggregated results (map[pluginName]RawMessage) to the desired string format.
func formatOutput(results map[string]json.RawMessage, format string) (string, error) {
	outputData := make(map[string]interface{}, len(results))
	for name, rawData := range results {
		var data interface{}
		if err := json.Unmarshal(rawData, &data); err != nil {
			log.Printf("Warning: Could not unmarshal stored raw data for plugin '%s', outputting as string: %v", name, err)
			outputData[name] = string(rawData) // Fallback
		} else {
			outputData[name] = data
		}
	}

	var outputBytes []byte
	var err error
	switch strings.ToLower(format) {
	case "json":
		outputBytes, err = json.MarshalIndent(outputData, "", "  ")
		if err != nil { return "", fmt.Errorf("failed to marshal results to JSON: %w", err) }
	case "yaml":
		fallthrough
	default:
		outputBytes, err = yaml.Marshal(outputData)
		if err != nil { return "", fmt.Errorf("failed to marshal results to YAML: %w", err) }
	}
	return string(outputBytes), nil
}

```
