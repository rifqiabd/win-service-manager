package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Display settings
	Display DisplayConfig `yaml:"display"`

	// Logging settings
	Logging LoggingConfig `yaml:"logging"`

	// Service filtering and sorting
	Services ServiceConfig `yaml:"services"`

	// Application behavior
	Behavior BehaviorConfig `yaml:"behavior"`
}

// DisplayConfig contains UI display settings
type DisplayConfig struct {
	// Number of services to show per page
	ServicesPerPage int `yaml:"services_per_page"`

	// Whether to show service descriptions
	ShowDescriptions bool `yaml:"show_descriptions"`

	// Whether to show service dependencies
	ShowDependencies bool `yaml:"show_dependencies"`

	// Whether to show process IDs
	ShowProcessIDs bool `yaml:"show_process_ids"`

	// Color theme
	Theme string `yaml:"theme"` // "dark", "light", "auto"

	// Refresh interval in seconds
	RefreshInterval int `yaml:"refresh_interval"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	// Log level: "debug", "info", "warn", "error"
	Level string `yaml:"level"`

	// Log file path (empty for stdout)
	File string `yaml:"file"`

	// Whether to log to file and stdout
	EnableConsole bool `yaml:"enable_console"`

	// Maximum log file size in MB
	MaxSize int `yaml:"max_size"`

	// Number of old log files to retain
	MaxBackups int `yaml:"max_backups"`
}

// ServiceConfig contains service filtering and sorting settings
type ServiceConfig struct {
	// Default filter: "all", "running", "stopped", "system", "user"
	DefaultFilter string `yaml:"default_filter"`

	// Default sort: "name", "status", "startup_type"
	DefaultSort string `yaml:"default_sort"`

	// Whether to show system services by default
	ShowSystemServices bool `yaml:"show_system_services"`

	// Services to hide from the list
	HiddenServices []string `yaml:"hidden_services"`

	// Services to highlight/pin at the top
	FavoriteServices []string `yaml:"favorite_services"`
}

// BehaviorConfig contains application behavior settings
type BehaviorConfig struct {
	// Confirm dangerous operations
	ConfirmStop bool `yaml:"confirm_stop"`

	// Auto-refresh services list
	AutoRefresh bool `yaml:"auto_refresh"`

	// Remember window size and position
	RememberLayout bool `yaml:"remember_layout"`

	// Enable keyboard shortcuts help
	ShowHelp bool `yaml:"show_help"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Display: DisplayConfig{
			ServicesPerPage:  25,
			ShowDescriptions: true,
			ShowDependencies: false,
			ShowProcessIDs:   true,
			Theme:            "auto",
			RefreshInterval:  5,
		},
		Logging: LoggingConfig{
			Level:         "info",
			File:          "",
			EnableConsole: true,
			MaxSize:       10,
			MaxBackups:    3,
		},
		Services: ServiceConfig{
			DefaultFilter:      "all",
			DefaultSort:        "name",
			ShowSystemServices: true,
			HiddenServices:     []string{},
			FavoriteServices:   []string{},
		},
		Behavior: BehaviorConfig{
			ConfirmStop:    true,
			AutoRefresh:    true,
			RememberLayout: false,
			ShowHelp:       true,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// If config file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConf := DefaultConfig()
		if err := SaveConfig(defaultConf, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConf, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and set defaults for missing fields
	config.applyDefaults()

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "win-service-manager.yaml"
	}

	return filepath.Join(homeDir, ".config", "win-service-manager", "config.yaml")
}

// applyDefaults fills in missing configuration values with defaults
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	// Display defaults
	if c.Display.ServicesPerPage <= 0 {
		c.Display.ServicesPerPage = defaults.Display.ServicesPerPage
	}
	if c.Display.Theme == "" {
		c.Display.Theme = defaults.Display.Theme
	}
	if c.Display.RefreshInterval <= 0 {
		c.Display.RefreshInterval = defaults.Display.RefreshInterval
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = defaults.Logging.Level
	}
	if c.Logging.MaxSize <= 0 {
		c.Logging.MaxSize = defaults.Logging.MaxSize
	}
	if c.Logging.MaxBackups < 0 {
		c.Logging.MaxBackups = defaults.Logging.MaxBackups
	}

	// Services defaults
	if c.Services.DefaultFilter == "" {
		c.Services.DefaultFilter = defaults.Services.DefaultFilter
	}
	if c.Services.DefaultSort == "" {
		c.Services.DefaultSort = defaults.Services.DefaultSort
	}
	if c.Services.HiddenServices == nil {
		c.Services.HiddenServices = make([]string, 0)
	}
	if c.Services.FavoriteServices == nil {
		c.Services.FavoriteServices = make([]string, 0)
	}
}

// IsValidLogLevel checks if the log level is valid
func (c *Config) IsValidLogLevel() bool {
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

// IsValidTheme checks if the theme is valid
func (c *Config) IsValidTheme() bool {
	switch c.Display.Theme {
	case "dark", "light", "auto":
		return true
	default:
		return false
	}
}

// IsValidFilter checks if the service filter is valid
func (c *Config) IsValidFilter() bool {
	switch c.Services.DefaultFilter {
	case "all", "running", "stopped", "system", "user":
		return true
	default:
		return false
	}
}

// IsValidSort checks if the service sort option is valid
func (c *Config) IsValidSort() bool {
	switch c.Services.DefaultSort {
	case "name", "status", "startup_type":
		return true
	default:
		return false
	}
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if !c.IsValidLogLevel() {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}
	if !c.IsValidTheme() {
		return fmt.Errorf("invalid theme: %s", c.Display.Theme)
	}
	if !c.IsValidFilter() {
		return fmt.Errorf("invalid default filter: %s", c.Services.DefaultFilter)
	}
	if !c.IsValidSort() {
		return fmt.Errorf("invalid default sort: %s", c.Services.DefaultSort)
	}
	if c.Display.ServicesPerPage <= 0 {
		return fmt.Errorf("services per page must be positive")
	}
	if c.Display.RefreshInterval <= 0 {
		return fmt.Errorf("refresh interval must be positive")
	}
	return nil
}
