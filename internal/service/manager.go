package service

import (
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceStatus represents the current status of a Windows service
type ServiceStatus int

const (
	StatusUnknown ServiceStatus = iota
	StatusStopped
	StatusStartPending
	StatusStopPending
	StatusRunning
)

// Service startup types
const (
	StartTypeAutomatic uint32 = 2 // mgr.StartAutomatic
	StartTypeManual    uint32 = 3 // mgr.StartManual
	StartTypeDisabled  uint32 = 4 // mgr.StartDisabled
)

func (s ServiceStatus) String() string {
	switch s {
	case StatusStopped:
		return "Stopped"
	case StatusStartPending:
		return "Starting"
	case StatusStopPending:
		return "Stopping"
	case StatusRunning:
		return "Running"
	default:
		return "Unknown"
	}
}

// ServiceInfo contains information about a Windows service
type ServiceInfo struct {
	Name         string
	DisplayName  string
	Description  string
	Status       ServiceStatus
	StartType    uint32
	ProcessID    uint32
	Dependencies []string
	ServiceType  uint32
}

// Manager provides methods for managing Windows services
type Manager struct {
	scm    *mgr.Mgr
	logger *slog.Logger
}

// NewManager creates a new service manager instance
func NewManager(logger *slog.Logger) (*Manager, error) {
	scm, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service control manager: %w", err)
	}

	return &Manager{
		scm:    scm,
		logger: logger,
	}, nil
}

// Close closes the connection to the service control manager
func (m *Manager) Close() error {
	if m.scm != nil {
		return m.scm.Disconnect()
	}
	return nil
}

// GetServices retrieves a list of all services on the system
func (m *Manager) GetServices() ([]ServiceInfo, error) {
	services, err := m.scm.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var serviceInfos []ServiceInfo
	for _, serviceName := range services {
		info, err := m.GetServiceInfo(serviceName)
		if err != nil {
			m.logger.Warn("failed to get service info", "service", serviceName, "error", err)
			// Continue with other services even if one fails
			continue
		}
		serviceInfos = append(serviceInfos, *info)
	}

	return serviceInfos, nil
}

// GetServiceInfo retrieves detailed information about a specific service
func (m *Manager) GetServiceInfo(serviceName string) (*ServiceInfo, error) {
	service, err := m.scm.OpenService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer service.Close()

	// Get service config
	config, err := service.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config for %s: %w", serviceName, err)
	}

	// Get service status
	status, err := service.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query service status for %s: %w", serviceName, err)
	}

	// Convert status
	serviceStatus := convertSvcState(status.State)

	// Get dependencies
	dependencies := config.Dependencies
	if dependencies == nil {
		dependencies = []string{}
	}

	return &ServiceInfo{
		Name:         serviceName,
		DisplayName:  config.DisplayName,
		Description:  config.Description,
		Status:       serviceStatus,
		StartType:    config.StartType,
		ProcessID:    status.ProcessId,
		Dependencies: dependencies,
		ServiceType:  config.ServiceType,
	}, nil
}

// StartService starts a Windows service
func (m *Manager) StartService(serviceName string) error {
	service, err := m.scm.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer service.Close()

	err = service.Start()
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}

	m.logger.Info("service started", "service", serviceName)
	return nil
}

// StopService stops a Windows service
func (m *Manager) StopService(serviceName string) error {
	service, err := m.scm.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer service.Close()

	status, err := service.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}

	// Wait for service to stop (with timeout)
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped && time.Now().Before(timeout) {
		time.Sleep(100 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status for %s: %w", serviceName, err)
		}
	}

	if status.State != svc.Stopped {
		return fmt.Errorf("service %s did not stop within timeout", serviceName)
	}

	m.logger.Info("service stopped", "service", serviceName)
	return nil
}

// ChangeStartupType changes the startup type of a Windows service
func (m *Manager) ChangeStartupType(serviceName string, startType uint32) error {
	service, err := m.scm.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service %s: %w", serviceName, err)
	}
	defer service.Close()

	// Get current config
	config, err := service.Config()
	if err != nil {
		return fmt.Errorf("failed to get service config for %s: %w", serviceName, err)
	}

	// Update the start type
	config.StartType = startType

	// Apply the new configuration
	err = service.UpdateConfig(config)
	if err != nil {
		return fmt.Errorf("failed to update service config for %s: %w", serviceName, err)
	}

	startTypeName := m.getStartTypeString(startType)
	m.logger.Info("service startup type changed", "service", serviceName, "startType", startTypeName)
	return nil
}

// getStartTypeString returns a human-readable string for start type
func (m *Manager) getStartTypeString(startType uint32) string {
	switch startType {
	case 2: // mgr.StartAutomatic
		return "Automatic"
	case 3: // mgr.StartManual
		return "Manual"
	case 4: // mgr.StartDisabled
		return "Disabled"
	default:
		return "Unknown"
	}
}

// convertSvcState converts Windows service state to our ServiceStatus enum
func convertSvcState(state svc.State) ServiceStatus {
	switch state {
	case svc.Stopped:
		return StatusStopped
	case svc.StartPending:
		return StatusStartPending
	case svc.StopPending:
		return StatusStopPending
	case svc.Running:
		return StatusRunning
	default:
		return StatusUnknown
	}
}

// IsServiceControllable checks if a service can be controlled (started/stopped)
func (s *ServiceInfo) IsServiceControllable() bool {
	// Services with automatic start type can usually be controlled
	// Some system services might not be controllable
	return s.StartType == 2 || s.StartType == 3 // StartAutomatic or StartManual
}

// CanStart returns true if the service can be started
func (s *ServiceInfo) CanStart() bool {
	return s.IsServiceControllable() && s.Status == StatusStopped
}

// CanStop returns true if the service can be stopped
func (s *ServiceInfo) CanStop() bool {
	return s.IsServiceControllable() && s.Status == StatusRunning
}

// CanChangeStartupType returns true if the service startup type can be changed
func (s *ServiceInfo) CanChangeStartupType() bool {
	// Most services can have their startup type changed
	// Some critical system services might be restricted
	return true
}

// GetStartTypeString returns a human-readable string for the service's start type
func (s *ServiceInfo) GetStartTypeString() string {
	switch s.StartType {
	case StartTypeAutomatic:
		return "Automatic"
	case StartTypeManual:
		return "Manual"
	case StartTypeDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// GetNextStartType returns the next startup type in cycle (Automatic -> Manual -> Disabled -> Automatic)
func (s *ServiceInfo) GetNextStartType() uint32 {
	switch s.StartType {
	case StartTypeAutomatic:
		return StartTypeManual
	case StartTypeManual:
		return StartTypeDisabled
	case StartTypeDisabled:
		return StartTypeAutomatic
	default:
		return StartTypeManual
	}
}
