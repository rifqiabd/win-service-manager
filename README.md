# Windows Service Manager (TUI)

> A modern, terminal-based Windows service management tool built with Go and Bubble Tea

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![Windows](https://img.shields.io/badge/Windows-10%2F11-0078D4?style=flat-square&logo=windows)](https://www.microsoft.com/windows/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

## Overview

Windows Service Manager is a powerful TUI (Terminal User Interface) replacement for the traditional Windows Services management console (`services.msc`). Built with Go and the elegant Bubble Tea framework, it provides a modern, keyboard-driven interface for managing Windows services efficiently.

## ✨ Features

### 🎯 Core Service Management
- **Start/Stop services** with immediate execution
- **Change startup type** (Automatic/Manual/Disabled) instantly
- **Real-time status updates** with auto-refresh
- **Detailed service information** including PID, startup type, and dependencies
- **Administrator privilege handling** with graceful error management

### 🎨 Modern Terminal Interface  
- **Beautiful TUI** styled with Lip Gloss
- **Vim-style navigation** for efficient keyboard control
- **Responsive design** that adapts to terminal size
- **Color-coded status indicators** for quick visual feedback

### ⚡ Advanced Capabilities
- **Powerful search** across service names and descriptions
- **Flexible filtering** (All/Running/Stopped/Manual/Disabled/Automatic)
- **YAML configuration** with sensible defaults
- **Structured logging** with configurable levels

## 🚀 Quick Start

### Prerequisites
- Windows 10 or 11
- Administrator privileges (recommended for service control operations)

### Installation

```bash
# Clone the repository
git clone https://github.com/sv222/win-service-manager.git
cd win-service-manager

# Build the executable
go build -o win-service-manager.exe .
```

### Usage
```bash
# Run with default settings
.\win-service-manager.exe

# Use custom configuration
.\win-service-manager.exe -config myconfig.yaml

# Enable debug logging
.\win-service-manager.exe -log-level debug
```

## ⌨️ Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | Go to top |
| `G` | Go to bottom |
| `n` / `PgDn` | Next page |
| `p` / `PgUp` | Previous page |

### Service Actions
| Key | Action |
|-----|--------|
| `Enter` | View service details |
| `s` | Start service |
| `t` | Stop service |
| `m` | Change startup type |

### View Controls
| Key | Action |
|-----|--------|
| `w` / `F5` | Refresh services |
| `f` | Cycle filter (All/Running/Stopped/Manual/Disabled/Automatic) |
| `/` | Enter search mode |
| `h` / `?` | Toggle help panel |
| `Esc` | Go back / Clear search |
| `q` / `Ctrl+C` | Quit |

### Search Mode
| Key | Action |
|-----|--------|
| `Enter` | Apply search and exit search mode |
| `Esc` | Exit search mode without applying |
| `Backspace` | Delete last character |
| Any letter/number | Add to search term |

## ⚙️ Configuration

Create `~/.config/win-service-manager/config.yaml`:

```yaml
display:
  services_per_page: 25
  show_descriptions: true
  show_process_ids: true
  theme: "auto"
  refresh_interval: 5

logging:
  level: "info"
  enable_console: true

services:
  default_filter: "all"    # Options: all, running, stopped, manual, disabled, automatic
  show_system_services: true

behavior:
  confirm_stop: false        # All operations execute immediately
  confirm_restart: false     # (Note: restart not implemented)
  auto_refresh: true
```

## 🏗️ Architecture

```
win-service-manager/
├── main.go                    # Application entry point
├── internal/
│   ├── service/              # Windows service management
│   │   └── manager.go       # SCM interface & operations  
│   ├── config/              # Configuration handling
│   │   └── config.go        # YAML config management
│   ├── logging/             # Structured logging
│   │   └── logger.go        # slog integration
│   └── ui/                  # Terminal interface
│       ├── model.go         # Bubble Tea state management
│       ├── view.go          # UI rendering with Lip Gloss
│       └── commands.go      # Async operations & messages
├── go.mod                   # Go module definition
└── sample-config.yaml       # Example configuration
```

## 📋 Requirements

- **OS**: Windows 10/11 
- **Go**: 1.21+ (for building from source)
- **Dependencies**:
  - [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
  - [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
  - [golang.org/x/sys/windows](https://pkg.go.dev/golang.org/x/sys/windows) - Windows API

## 🔒 Security

- Handles administrator privileges gracefully
- Operations execute immediately for efficiency
- Safe service operation timeouts
- Comprehensive error handling and logging

## 🐛 Troubleshooting

### Common Issues

**Access Denied Errors**
- Run as administrator or use "Run as administrator" context menu

**Services Not Loading**  
- Ensure Windows Service Control Manager is accessible
- Try running with administrator privileges

**Configuration Issues**
- Delete config file to reset to defaults: `~/.config/win-service-manager/config.yaml`

## Contributing
We welcome contributions from the community. If you have ideas, bug reports, or feature requests, please open an issue or submit a pull request.

## License
This project is licensed under the MIT License.