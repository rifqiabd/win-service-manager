package ui

import (
	"fmt"
	"strings"
	"time"
	"win-service-manager/internal/service"

	tea "github.com/charmbracelet/bubbletea"
)

// Message types for Bubble Tea
type (
	// ServicesRefreshedMsg is sent when services list is refreshed
	ServicesRefreshedMsg struct {
		Services []service.ServiceInfo
	}

	// ServiceOperationMsg is sent when a service operation completes
	ServiceOperationMsg struct {
		Operation   string
		ServiceName string
		Message     string
		Error       error
	}

	// StartupTypeChangedMsg is sent when a service startup type is changed
	StartupTypeChangedMsg struct {
		ServiceName string
		OldType     string
		NewType     string
		Message     string
		Error       error
	}

	// ErrorMsg is sent when an error occurs
	ErrorMsg struct {
		Error error
	}

	// TickMsg is sent by the ticker for auto-refresh
	TickMsg time.Time
)

// refreshServicesCmd returns a command to refresh the services list
func (m *Model) refreshServicesCmd() tea.Cmd {
	return func() tea.Msg {
		services, err := m.serviceManager.GetServices()
		if err != nil {
			return ErrorMsg{Error: err}
		}
		return ServicesRefreshedMsg{Services: services}
	}
}

// performServiceOperation returns a command to perform a service operation
func (m *Model) performServiceOperation(operation, serviceName string) tea.Cmd {
	return func() tea.Msg {
		var err error
		var message string

		// Log the operation attempt
		m.logger.Info("attempting service operation", "operation", operation, "service", serviceName)

		switch operation {
		case "start":
			err = m.serviceManager.StartService(serviceName)
			if err == nil {
				message = fmt.Sprintf("Service '%s' started successfully", serviceName)
			} else {
				message = fmt.Sprintf("Failed to start service '%s'", serviceName)
			}
		case "stop":
			err = m.serviceManager.StopService(serviceName)
			if err == nil {
				message = fmt.Sprintf("Service '%s' stopped successfully", serviceName)
			} else {
				message = fmt.Sprintf("Failed to stop service '%s'", serviceName)
			}
		default:
			err = fmt.Errorf("unknown operation: %s", operation)
			message = "Unknown operation"
		}

		// Enhanced error message for common permission issues
		if err != nil {
			m.logger.Error("service operation failed", "operation", operation, "service", serviceName, "error", err)

			// Check for common permission-related errors
			errorStr := err.Error()
			if strings.Contains(errorStr, "Access is denied") || strings.Contains(errorStr, "access denied") {
				message += " - Access denied. Try running as Administrator."
			} else if strings.Contains(errorStr, "service does not exist") {
				message += " - Service not found."
			} else {
				message += fmt.Sprintf(" - %s", err.Error())
			}
		} else {
			m.logger.Info("service operation completed", "operation", operation, "service", serviceName)
		}

		return ServiceOperationMsg{
			Operation:   operation,
			ServiceName: serviceName,
			Message:     message,
			Error:       err,
		}
	}
}

// tickCmd returns a command for the auto-refresh ticker
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// changeStartupTypeCmd returns a command to change a service's startup type
func (m *Model) changeStartupTypeCmd(serviceName string, newStartType uint32) tea.Cmd {
	return func() tea.Msg {
		// Get current service info to capture old type
		serviceInfo, err := m.serviceManager.GetServiceInfo(serviceName)
		if err != nil {
			return StartupTypeChangedMsg{
				ServiceName: serviceName,
				Message:     fmt.Sprintf("Failed to get service info for '%s'", serviceName),
				Error:       err,
			}
		}

		oldTypeString := serviceInfo.GetStartTypeString()
		newTypeString := m.getStartTypeString(newStartType)

		// Log the operation attempt
		m.logger.Info("attempting startup type change",
			"service", serviceName,
			"oldType", oldTypeString,
			"newType", newTypeString)

		// Perform the startup type change
		err = m.serviceManager.ChangeStartupType(serviceName, newStartType)

		var message string
		if err == nil {
			message = fmt.Sprintf("Service '%s' startup type changed from %s to %s",
				serviceName, oldTypeString, newTypeString)
			m.logger.Info("startup type changed successfully",
				"service", serviceName,
				"oldType", oldTypeString,
				"newType", newTypeString)
		} else {
			message = fmt.Sprintf("Failed to change startup type for '%s'", serviceName)
			m.logger.Error("startup type change failed",
				"service", serviceName,
				"error", err)

			// Enhanced error message for common permission issues
			errorStr := err.Error()
			if strings.Contains(errorStr, "Access is denied") || strings.Contains(errorStr, "access denied") {
				message += " - Access denied. Try running as Administrator."
			} else if strings.Contains(errorStr, "service does not exist") {
				message += " - Service not found."
			} else {
				message += fmt.Sprintf(" - %s", err.Error())
			}
		}

		return StartupTypeChangedMsg{
			ServiceName: serviceName,
			OldType:     oldTypeString,
			NewType:     newTypeString,
			Message:     message,
			Error:       err,
		}
	}
}

// getStartTypeString returns a human-readable string for start type
func (m *Model) getStartTypeString(startType uint32) string {
	switch startType {
	case 2: // StartTypeAutomatic
		return "Automatic"
	case 3: // StartTypeManual
		return "Manual"
	case 4: // StartTypeDisabled
		return "Disabled"
	default:
		return "Unknown"
	}
}
