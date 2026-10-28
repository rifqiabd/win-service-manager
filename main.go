package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"win-service-manager/internal/config"
	"win-service-manager/internal/elevate"
	"win-service-manager/internal/service"
	"win-service-manager/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	AppName = "Windows Service Manager"
)

var AppVersion = "dev"

var (
	configPath  = flag.String("config", "", "Path to configuration file")
	logLevel    = flag.String("log-level", "", "Log level (debug, info, warn, error)")
	showVersion = flag.Bool("version", false, "Show version information")
	showHelp    = flag.Bool("help", false, "Show help information")
	noElevate   = flag.Bool("no-elevate", false, "Do not request administrator privileges via UAC")
)

func main() {
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		os.Exit(0)
	}

	// Show help and exit
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Request administrator privileges through UAC before starting the TUI.
	// Version/help output is intentionally handled above so scripts do not
	// trigger a UAC prompt.
	elevated := elevate.IsElevated()
	if !elevated && !*noElevate {
		fmt.Fprintln(os.Stderr, "Requesting administrator privileges (UAC)...")
		if err := elevate.RelaunchElevated(); err != nil {
			if errors.Is(err, elevate.ErrCancelled) {
				fmt.Fprintln(os.Stderr, "Administrator privileges are required to manage services.")
				fmt.Fprintln(os.Stderr, "Use -no-elevate to browse services without elevation.")
			} else {
				fmt.Fprintf(os.Stderr, "Failed to request administrator privileges: %v\n", err)
			}
			os.Exit(1)
		}
		// The elevated instance is now running in its own console.
		return
	}

	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override log level if specified via command line
	if *logLevel != "" {
		cfg.Logging.Level = *logLevel
	}

	// Setup logger
	logger := setupLogger(cfg)

	// Log application start
	logger.Info("application starting", "version", AppVersion, "pid", os.Getpid())

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Check if running on Windows
	if err := checkPlatform(); err != nil {
		logger.Error("platform check failed", "error", err)
		os.Exit(1)
	}

	// Initialize service manager
	serviceManager, err := service.NewManager(logger)
	if err != nil {
		logger.Error("failed to initialize service manager", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := serviceManager.Close(); err != nil {
			logger.Error("failed to close service manager", "error", err)
		}
	}()

	// Create Bubble Tea model
	model := ui.NewModel(cfg, serviceManager, logger)
	model.SetElevated(elevated)

	// Setup Bubble Tea program options
	var opts []tea.ProgramOption

	// Enable mouse support if desired
	opts = append(opts, tea.WithMouseCellMotion())

	// Set alternate screen buffer
	opts = append(opts, tea.WithAltScreen())

	// Create and run the program
	program := tea.NewProgram(model, opts...)

	logger.Info("starting TUI application")

	// Handle graceful shutdown
	if err := program.Start(); err != nil {
		logger.Error("application error", "error", err)
		logger.Info("application stopping", "reason", "error")
		os.Exit(1)
	}

	logger.Info("application stopping", "reason", "normal")
}

// setupLogger creates a simple logger based on configuration
func setupLogger(cfg *config.Config) *slog.Logger {
	// Parse log level
	var level slog.Level
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use file logging to avoid conflicts with TUI
	logFile := "win-service-manager.log"
	if cfg.Logging.File != "" {
		logFile = cfg.Logging.File
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fallback to stderr if file can't be opened
		handler := slog.NewTextHandler(os.Stderr, opts)
		logger := slog.New(handler)
		slog.SetDefault(logger)
		return logger
	}

	// Create text handler for file output
	handler := slog.NewTextHandler(file, opts)
	logger := slog.New(handler)

	// Set as default
	slog.SetDefault(logger)

	return logger
}

// loadConfiguration loads the application configuration
func loadConfiguration() (*config.Config, error) {
	var configFile string

	if *configPath != "" {
		configFile = *configPath
	} else {
		configFile = config.GetDefaultConfigPath()
	}

	// Expand tilde to home directory
	if configFile[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(homeDir, configFile[1:])
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	return cfg, nil
}

// checkPlatform ensures the application is running on Windows
func checkPlatform() error {
	if os.Getenv("GOOS") == "windows" ||
		os.Getenv("OS") == "Windows_NT" ||
		filepath.Separator == '\\' {
		return nil
	}
	return fmt.Errorf("this application requires Windows operating system")
}

// printHelp prints the help information
func printHelp() {
	fmt.Printf(`%s v%s

A modern TUI-based Windows service manager built with Go and Bubble Tea.

USAGE:
    win-service-manager [OPTIONS]

OPTIONS:
    -config string
        Path to configuration file
        (default: ~/.config/win-service-manager/config.yaml)
    
    -log-level string
        Log level: debug, info, warn, error
        (overrides configuration file setting)
    
    -version
        Show version information and exit
    
    -help
        Show this help information and exit

    -no-elevate
        Do not request administrator privileges via UAC
        (the TUI still opens, but service changes may fail)

ELEVATION:
    By default the app requests administrator privileges through the
    Windows UAC prompt on startup. Version/help output never triggers UAC.

KEYBOARD SHORTCUTS:
    Navigation:
        j/↓         Move down
        k/↑         Move up
        g/G         Go to top/bottom
        n/p         Page down/up
        Tab         Switch focus (services <-> filters)
        ←/→         Switch focus (services <-> filters)
        1-6         Jump to filter

    Actions:
        Enter       Open service details
        s           Start selected service
        t           Stop selected service
        m           Change startup type

    View Options:
        w/F5        Refresh services list
        f           Cycle filter
        /           Search services
        h/?         Toggle help
        Esc         Back / clear search

    Other:
        q/Ctrl+C    Quit application

CONFIGURATION:
    The application uses a YAML configuration file that controls:
    - Display settings (theme, refresh interval)
    - Logging configuration (level, file output)
    - Service management defaults
    - Behavior settings (confirmation dialogs, auto-refresh)

    If no configuration file exists, a default one will be created.

EXAMPLES:
    # Run with default settings (prompts for UAC)
    win-service-manager

    # Use custom config file
    win-service-manager -config myconfig.yaml

    # Browse without elevation
    win-service-manager -no-elevate

    # Enable debug logging
    win-service-manager -log-level debug

REQUIREMENTS:
    - Windows operating system
    - Administrator privileges for service changes (requested automatically)

For more information, visit: https://github.com/rifqiabd/win-service-manager
`, AppName, AppVersion)
}

// handlePanic recovers from panics and logs them
func handlePanic(logger *slog.Logger) {
	if r := recover(); r != nil {
		logger.Error("application panic", "panic", r)
		logger.Info("application stopping", "reason", "panic")
		os.Exit(1)
	}
}

// init sets up panic recovery
func init() {
	// This will be called if logger is available
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Fatal error during initialization: %v\n", r)
			os.Exit(1)
		}
	}()
}
