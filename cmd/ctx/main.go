package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/ctx/docs"
	"sigs.k8s.io/yaml"
)

const version = "0.1.0"

type Config struct {
	Plugins        []string `yaml:"plugins"`
	SecurityLevel  string   `yaml:"security_level"`
	OutputFormat   string   `yaml:"output_format"`
	TokenLimit     int      `yaml:"token_limit"`
	ConfigFilePath string   `yaml:"config_file_path"`
}

type Plugin struct {
	Name         string
	Version      string
	Capabilities []string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config := &Config{}
	flag.StringVar(&config.ConfigFilePath, "config", "", "Path to configuration file")
	listPlugins := flag.Bool("list-plugins", false, "Show available plugins")
	printPluginSpec := flag.Bool("print-plugin-spec", false, "Print plugin specification")
	flag.StringVar(&config.SecurityLevel, "security", "medium", "Set security level (low, medium, high)")
	flag.StringVar(&config.OutputFormat, "output", "yaml", "Set output format (text, json, yaml, markdown)")
	showVersion := flag.Bool("version", false, "Display version information")
	dryRun := flag.Bool("dry-run", false, "Preview actions without executing")
	pluginAction := flag.String("plugin", "", "Manage plugins (install, update, remove)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ctx version %s\n", version)
		return nil
	}

	if *printPluginSpec {
		return printPluginSpecification()
	}

	if err := loadConfig(config); err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	if *listPlugins {
		return listAvailablePlugins()
	}

	if *pluginAction != "" {
		return managePlugins(*pluginAction)
	}

	ctx := context.Background()
	env, err := discoverEnvironment()
	if err != nil {
		return fmt.Errorf("error discovering environment: %w", err)
	}

	plugins, err := discoverPlugins()
	if err != nil {
		return fmt.Errorf("error discovering plugins: %w", err)
	}

	plan := planExecution(env, plugins, config)

	if *dryRun {
		return printExecutionPlan(plan)
	}

	result, err := executePlugins(ctx, plan)
	if err != nil {
		return fmt.Errorf("error executing plugins: %w", err)
	}

	if err := applySecurityFilters(result, config.SecurityLevel); err != nil {
		return fmt.Errorf("error applying security filters: %w", err)
	}

	if err := manageTokenBudget(result, config.TokenLimit); err != nil {
		return fmt.Errorf("error managing token budget: %w", err)
	}

	output, err := formatOutput(result, config.OutputFormat)
	if err != nil {
		return fmt.Errorf("error formatting output: %w", err)
	}

	fmt.Println(output)
	return nil
}

func loadConfig(config *Config) error {
	// Load from environment variables
	if tokenLimit := os.Getenv("CTX_TOKEN_LIMIT"); tokenLimit != "" {
		fmt.Sscanf(tokenLimit, "%d", &config.TokenLimit)
	}
	if plugins := os.Getenv("CTX_PLUGINS"); plugins != "" {
		config.Plugins = strings.Split(plugins, ",")
	}
	if outputFormat := os.Getenv("CTX_OUTPUT_FORMAT"); outputFormat != "" {
		config.OutputFormat = outputFormat
	}

	// Load from config file if specified
	if config.ConfigFilePath != "" {
		data, err := os.ReadFile(config.ConfigFilePath)
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("error parsing config file: %w", err)
		}
	}

	return nil
}

func discoverEnvironment() (map[string]string, error) {
	// Placeholder implementation
	return map[string]string{
		"working_directory": os.Getenv("PWD"),
		"os":                os.Getenv("GOOS"),
		"arch":              os.Getenv("GOARCH"),
	}, nil
}

func discoverPlugins() ([]Plugin, error) {
	var plugins []Plugin
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "ctx-") && !file.IsDir() {
				plugin, err := getPluginInfo(filepath.Join(path, file.Name()))
				if err != nil {
					log.Printf("Error getting plugin info for %s: %v", file.Name(), err)
					continue
				}
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins, nil
}

func getPluginInfo(pluginPath string) (Plugin, error) {
	cmd := exec.Command(pluginPath, "--capabilities")
	output, err := cmd.Output()
	if err != nil {
		return Plugin{}, err
	}

	var plugin Plugin
	if err := json.Unmarshal(output, &plugin); err != nil {
		return Plugin{}, err
	}

	return plugin, nil
}

func planExecution(env map[string]string, plugins []Plugin, config *Config) []string {
	// Placeholder implementation
	var plan []string
	for _, plugin := range plugins {
		plan = append(plan, plugin.Name)
	}
	return plan
}

func printExecutionPlan(plan []string) error {
	fmt.Println("Execution plan:")
	for i, plugin := range plan {
		fmt.Printf("%d. %s\n", i+1, plugin)
	}
	return nil
}

func executePlugins(ctx context.Context, plan []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, pluginName := range plan {
		cmd := exec.CommandContext(ctx, pluginName)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("error executing plugin %s: %w", pluginName, err)
		}

		result[pluginName] = string(output)
	}
	return result, nil
}

func applySecurityFilters(result map[string]interface{}, securityLevel string) error {
	// Placeholder implementation
	return nil
}

func manageTokenBudget(result map[string]interface{}, tokenLimit int) error {
	// Placeholder implementation
	return nil
}

func formatOutput(result map[string]interface{}, format string) (string, error) {
	var output []byte
	var err error

	switch format {
	case "json":
		output, err = json.MarshalIndent(result, "", "  ")
	case "yaml":
		output, err = yaml.Marshal(result)
	case "text":
		// Implement text formatting
		return "Text output not implemented yet", nil
	case "markdown":
		// Implement markdown formatting
		return "Markdown output not implemented yet", nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return "", fmt.Errorf("error formatting output: %w", err)
	}

	return string(output), nil
}

func listAvailablePlugins() error {
	plugins, err := discoverPlugins()
	if err != nil {
		return fmt.Errorf("error discovering plugins: %w", err)
	}

	fmt.Println("Available plugins:")
	for _, plugin := range plugins {
		fmt.Printf("- %s (version %s)\n", plugin.Name, plugin.Version)
		fmt.Printf("  Capabilities: %s\n", strings.Join(plugin.Capabilities, ", "))
	}

	return nil
}

func managePlugins(action string) error {
	// Placeholder implementation
	fmt.Printf("Plugin management action: %s\n", action)
	fmt.Println("Plugin management not implemented yet")
	return nil
}

func printPluginSpecification() error {
	specFile, err := docs.All.Open("PLUGIN_SPEC.md")
	if err != nil {
		return fmt.Errorf("error opening plugin specification: %w", err)
	}
	defer specFile.Close()

	_, err = io.Copy(os.Stdout, specFile)
	if err != nil {
		return fmt.Errorf("error printing plugin specification: %w", err)
	}

	return nil
}
