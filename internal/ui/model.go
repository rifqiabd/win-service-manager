package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"win-service-manager/internal/config"
	"win-service-manager/internal/service"

	tea "github.com/charmbracelet/bubbletea"
)

// ApplicationState represents the current state of the application
type ApplicationState int

const (
	StateServiceList ApplicationState = iota
	StateServiceDetails
	StateConfirmDialog
	StateError
	StateLoading
	StateSearch
)

// FilterType represents different service filtering options
type FilterType int

const (
	FilterAll FilterType = iota
	FilterRunning
	FilterStopped
	FilterManual
	FilterDisabled
	FilterAutomatic
)

func (f FilterType) String() string {
	switch f {
	case FilterAll:
		return "All"
	case FilterRunning:
		return "Running"
	case FilterStopped:
		return "Stopped"
	case FilterManual:
		return "Manual"
	case FilterDisabled:
		return "Disabled"
	case FilterAutomatic:
		return "Automatic"
	default:
		return "All"
	}
}

// ConfirmAction represents actions that require confirmation
type ConfirmAction struct {
	Action      string
	ServiceName string
	Message     string
}

// Model represents the application state for Bubble Tea
type Model struct {
	// Application state
	state  ApplicationState
	width  int
	height int
	ready  bool

	// Configuration and dependencies
	config         *config.Config
	serviceManager *service.Manager
	logger         *slog.Logger

	// Service data
	services      []service.ServiceInfo
	selectedIndex int
	currentPage   int

	// UI state
	showHelp      bool
	errorMessage  string
	statusMessage string

	// Filtering
	currentFilter    FilterType
	filteredServices []service.ServiceInfo

	// Search functionality
	searchMode   bool
	searchTerm   string
	searchActive bool

	// Confirmation dialog
	confirmAction *ConfirmAction

	// Auto-refresh
	lastRefresh   time.Time
	refreshTicker *time.Ticker

	// Pagination
	itemsPerPage int
}

// NewModel creates a new Bubble Tea model
func NewModel(cfg *config.Config, svcMgr *service.Manager, logger *slog.Logger) *Model {
	return &Model{
		state:            StateLoading,
		width:            80, // Default width
		height:           24, // Default height
		ready:            false,
		config:           cfg,
		serviceManager:   svcMgr,
		logger:           logger,
		services:         make([]service.ServiceInfo, 0),
		selectedIndex:    0,
		currentPage:      0,
		showHelp:         false, // Start with help hidden
		itemsPerPage:     cfg.Display.ServicesPerPage,
		lastRefresh:      time.Now(),
		currentFilter:    FilterAll,
		filteredServices: make([]service.ServiceInfo, 0),
		searchMode:       false,
		searchTerm:       "",
		searchActive:     false,
	}
}

// Init initializes the model (required by Bubble Tea)
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshServicesCmd(),
		m.tickCmd(),
	)
}

// Update handles messages and updates the model (required by Bubble Tea)
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case ServicesRefreshedMsg:
		m.services = msg.Services
		m.applyFilter() // Apply current filter to new services
		m.state = StateServiceList
		m.statusMessage = fmt.Sprintf("Services refreshed - Filter: %s (%d services)", m.currentFilter.String(), len(m.filteredServices))
		m.lastRefresh = time.Now()
		return m, nil

	case ServiceOperationMsg:
		if msg.Error != nil {
			m.errorMessage = msg.Error.Error()
			m.state = StateError
		} else {
			m.statusMessage = msg.Message
			// Refresh services after operation
			return m, m.refreshServicesCmd()
		}
		return m, nil

	case StartupTypeChangedMsg:
		if msg.Error != nil {
			m.errorMessage = msg.Error.Error()
			m.state = StateError
		} else {
			m.statusMessage = msg.Message
			// Refresh services after startup type change
			return m, m.refreshServicesCmd()
		}
		return m, nil

	case ErrorMsg:
		m.errorMessage = msg.Error.Error()
		m.state = StateError
		return m, nil

	case TickMsg:
		if m.config.Behavior.AutoRefresh {
			if time.Since(m.lastRefresh) >= time.Duration(m.config.Display.RefreshInterval)*time.Second {
				return m, m.refreshServicesCmd()
			}
		}
		return m, m.tickCmd()
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateError:
		// Any key clears error state
		m.errorMessage = ""
		m.state = StateServiceList
		return m, nil

	case StateConfirmDialog:
		return m.handleConfirmDialogKeys(msg)

	case StateServiceList:
		return m.handleServiceListKeys(msg)

	case StateServiceDetails:
		return m.handleServiceDetailsKeys(msg)

	case StateSearch:
		return m.handleSearchKeys(msg)
	}

	return m, nil
}

// handleServiceListKeys handles keys in the service list view
func (m *Model) handleServiceListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "w", "F5":
		return m, m.refreshServicesCmd()

	case "h", "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "j", "down":
		m.navigateDown()
		return m, nil

	case "k", "up":
		m.navigateUp()
		return m, nil

	case "g":
		m.selectedIndex = 0
		m.currentPage = 0
		return m, nil

	case "G":
		if len(m.filteredServices) > 0 {
			m.selectedIndex = len(m.filteredServices) - 1
			m.currentPage = m.selectedIndex / m.itemsPerPage
		}
		return m, nil

	case "n", "pgdn":
		m.nextPage()
		return m, nil

	case "p", "pgup":
		m.prevPage()
		return m, nil

	case "enter":
		if len(m.filteredServices) > 0 {
			m.state = StateServiceDetails
		}
		return m, nil

	case "s":
		return m.startSelectedService()

	case "t":
		// Direct stop without confirmation (using T implementation)
		return m.stopSelectedServiceDirect()

	case "m":
		// Direct startup type change without confirmation
		return m.changeSelectedServiceStartupTypeDirect()

	case "f":
		// Cycle through filters
		return m.cycleFilter()

	case "/":
		// Enter search mode
		m.searchMode = true
		m.state = StateSearch
		m.searchTerm = ""
		return m, nil

	case "esc":
		// Clear search if active
		if m.searchActive {
			m.clearSearch()
			return m, nil
		}
	}

	return m, nil
}

// handleServiceDetailsKeys handles keys in the service details view
func (m *Model) handleServiceDetailsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.state = StateServiceList
		return m, nil

	case "ctrl+c", "q":
		return m, tea.Quit

	case "s":
		return m.startSelectedService()

	case "t":
		// Direct stop without confirmation
		return m.stopSelectedServiceDirect()

	case "m":
		// Direct startup type change without confirmation
		return m.changeSelectedServiceStartupTypeDirect()

	case "w", "F5":
		return m, m.refreshServicesCmd()

	case "f":
		// Cycle through filters
		return m.cycleFilter()
	}

	return m, nil
}

// handleConfirmDialogKeys handles keys in the confirmation dialog
func (m *Model) handleConfirmDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if m.confirmAction != nil {
			cmd := m.executeConfirmedAction()
			m.confirmAction = nil
			m.state = StateServiceList
			return m, cmd
		}
		m.state = StateServiceList
		return m, nil

	case "n", "N", "esc":
		m.confirmAction = nil
		m.state = StateServiceList
		return m, nil
	}

	return m, nil
}

// Navigation methods
func (m *Model) navigateDown() {
	if len(m.filteredServices) == 0 {
		return
	}

	m.selectedIndex++
	if m.selectedIndex >= len(m.filteredServices) {
		m.selectedIndex = len(m.filteredServices) - 1
		return
	}

	// Update page if necessary
	newPage := m.selectedIndex / m.itemsPerPage
	if newPage != m.currentPage {
		m.currentPage = newPage
	}

	// Update status message for debugging
	m.statusMessage = fmt.Sprintf("Selected: %d/%d (Filter: %s)", m.selectedIndex+1, len(m.filteredServices), m.currentFilter.String())
}

func (m *Model) navigateUp() {
	if len(m.filteredServices) == 0 {
		return
	}

	m.selectedIndex--
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
		return
	}

	// Update page if necessary
	newPage := m.selectedIndex / m.itemsPerPage
	if newPage != m.currentPage {
		m.currentPage = newPage
	}

	// Update status message for debugging
	m.statusMessage = fmt.Sprintf("Selected: %d/%d (Filter: %s)", m.selectedIndex+1, len(m.filteredServices), m.currentFilter.String())
}

func (m *Model) nextPage() {
	if len(m.filteredServices) == 0 {
		return
	}

	totalPages := (len(m.filteredServices) + m.itemsPerPage - 1) / m.itemsPerPage
	if m.currentPage < totalPages-1 {
		m.currentPage++
		m.selectedIndex = m.currentPage * m.itemsPerPage
		// Ensure we don't go beyond the available services
		if m.selectedIndex >= len(m.filteredServices) {
			m.selectedIndex = len(m.filteredServices) - 1
		}
		m.statusMessage = fmt.Sprintf("Page %d/%d (Filter: %s)", m.currentPage+1, totalPages, m.currentFilter.String())
	}
}

func (m *Model) prevPage() {
	if m.currentPage > 0 {
		m.currentPage--
		m.selectedIndex = m.currentPage * m.itemsPerPage
		totalPages := (len(m.filteredServices) + m.itemsPerPage - 1) / m.itemsPerPage
		m.statusMessage = fmt.Sprintf("Page %d/%d (Filter: %s)", m.currentPage+1, totalPages, m.currentFilter.String())
	}
}

// Service operation methods
func (m *Model) startSelectedService() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanStart() {
		m.statusMessage = fmt.Sprintf("Service '%s' is already %s", selectedService.DisplayName, selectedService.Status.String())
		return m, nil
	}

	m.statusMessage = fmt.Sprintf("Attempting to start: %s", selectedService.DisplayName)
	return m, m.performServiceOperation("start", selectedService.Name)
}

func (m *Model) stopSelectedService() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]
	m.statusMessage = fmt.Sprintf("Attempting to stop: %s", selectedService.DisplayName)

	if !selectedService.CanStop() {
		m.errorMessage = fmt.Sprintf("Service '%s' cannot be stopped (current status: %s)", selectedService.DisplayName, selectedService.Status.String())
		m.state = StateError
		return m, nil
	}

	// Debug: Log configuration and state
	m.logger.Info("stop service requested", "service", selectedService.Name, "confirm_stop", m.config.Behavior.ConfirmStop, "current_state", m.state)

	if m.config.Behavior.ConfirmStop {
		m.confirmAction = &ConfirmAction{
			Action:      "stop",
			ServiceName: selectedService.Name,
			Message:     fmt.Sprintf("Stop service '%s'?", selectedService.DisplayName),
		}
		m.state = StateConfirmDialog
		m.statusMessage = fmt.Sprintf("DEBUG: Showing confirmation dialog for %s (State: %d)", selectedService.DisplayName, m.state)
		m.logger.Info("confirmation dialog requested", "service", selectedService.Name, "state", m.state)
		return m, nil
	}

	return m, m.performServiceOperation("stop", selectedService.Name)
}

func (m *Model) stopSelectedServiceDirect() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanStop() {
		m.statusMessage = fmt.Sprintf("Service '%s' is already %s", selectedService.DisplayName, selectedService.Status.String())
		return m, nil
	}

	m.statusMessage = fmt.Sprintf("Attempting to stop: %s", selectedService.DisplayName)
	return m, m.performServiceOperation("stop", selectedService.Name)
}

func (m *Model) changeSelectedServiceStartupType() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanChangeStartupType() {
		m.errorMessage = fmt.Sprintf("Startup type for service '%s' cannot be changed", selectedService.DisplayName)
		m.state = StateError
		return m, nil
	}

	newStartType := selectedService.GetNextStartType()
	oldTypeString := selectedService.GetStartTypeString()
	newTypeString := m.getStartTypeString(newStartType)

	m.statusMessage = fmt.Sprintf("Attempting to change startup type: %s (%s -> %s)",
		selectedService.DisplayName, oldTypeString, newTypeString)

	// Use confirmation dialog for startup type changes
	if m.config.Behavior.ConfirmStop { // Reuse the confirmation setting
		m.confirmAction = &ConfirmAction{
			Action:      "change_startup",
			ServiceName: selectedService.Name,
			Message: fmt.Sprintf("Change startup type for '%s' from %s to %s?",
				selectedService.DisplayName, oldTypeString, newTypeString),
		}
		m.state = StateConfirmDialog
		m.statusMessage = fmt.Sprintf("DEBUG: Showing startup type confirmation dialog for %s (State: %d)",
			selectedService.DisplayName, m.state)
		m.logger.Info("startup type confirmation dialog requested",
			"service", selectedService.Name,
			"oldType", oldTypeString,
			"newType", newTypeString,
			"state", m.state)
		return m, nil
	}

	return m, m.changeStartupTypeCmd(selectedService.Name, newStartType)
}

func (m *Model) changeSelectedServiceStartupTypeDirect() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanChangeStartupType() {
		m.errorMessage = fmt.Sprintf("Startup type for service '%s' cannot be changed", selectedService.DisplayName)
		m.state = StateError
		return m, nil
	}

	newStartType := selectedService.GetNextStartType()
	oldTypeString := selectedService.GetStartTypeString()
	newTypeString := m.getStartTypeString(newStartType)

	m.statusMessage = fmt.Sprintf("Changing startup type: %s (%s -> %s)",
		selectedService.DisplayName, oldTypeString, newTypeString)

	return m, m.changeStartupTypeCmd(selectedService.Name, newStartType)
}

// Helper methods
func (m *Model) executeConfirmedAction() tea.Cmd {
	if m.confirmAction == nil {
		return nil
	}

	switch m.confirmAction.Action {
	case "change_startup":
		// Get the service info to determine the new startup type
		if len(m.filteredServices) > 0 {
			selectedService := m.filteredServices[m.selectedIndex]
			if selectedService.Name == m.confirmAction.ServiceName {
				newStartType := selectedService.GetNextStartType()
				return m.changeStartupTypeCmd(selectedService.Name, newStartType)
			}
		}
		return nil
	default:
		return m.performServiceOperation(m.confirmAction.Action, m.confirmAction.ServiceName)
	}
}

func (m *Model) GetSelectedService() *service.ServiceInfo {
	if len(m.filteredServices) == 0 || m.selectedIndex < 0 || m.selectedIndex >= len(m.filteredServices) {
		return nil
	}
	return &m.filteredServices[m.selectedIndex]
}

func (m *Model) GetVisibleServices() []service.ServiceInfo {
	start := m.currentPage * m.itemsPerPage
	end := start + m.itemsPerPage

	if start >= len(m.filteredServices) {
		return []service.ServiceInfo{}
	}

	if end > len(m.filteredServices) {
		end = len(m.filteredServices)
	}

	return m.filteredServices[start:end]
}

func (m *Model) GetPageInfo() (currentPage, totalPages int) {
	if len(m.filteredServices) == 0 {
		return 0, 0
	}

	totalPages = (len(m.filteredServices) + m.itemsPerPage - 1) / m.itemsPerPage
	return m.currentPage + 1, totalPages
}

// cycleFilter cycles through the different filter types
func (m *Model) cycleFilter() (tea.Model, tea.Cmd) {
	// Cycle to next filter type
	switch m.currentFilter {
	case FilterAll:
		m.currentFilter = FilterRunning
	case FilterRunning:
		m.currentFilter = FilterStopped
	case FilterStopped:
		m.currentFilter = FilterManual
	case FilterManual:
		m.currentFilter = FilterDisabled
	case FilterDisabled:
		m.currentFilter = FilterAutomatic
	case FilterAutomatic:
		m.currentFilter = FilterAll
	default:
		m.currentFilter = FilterAll
	}

	// Apply the new filter
	m.applyFilter()

	// Reset selection and pagination
	m.selectedIndex = 0
	m.currentPage = 0

	// Update status message
	m.statusMessage = fmt.Sprintf("Filter: %s (%d services)", m.currentFilter.String(), len(m.filteredServices))

	return m, nil
}

// applyFilter applies the current filter to the services list
func (m *Model) applyFilter() {
	m.filteredServices = make([]service.ServiceInfo, 0)

	for _, svc := range m.services {
		if m.matchesFilter(svc) {
			m.filteredServices = append(m.filteredServices, svc)
		}
	}
}

// matchesFilter checks if a service matches the current filter
func (m *Model) matchesFilter(svc service.ServiceInfo) bool {
	// First check if it matches the search term (if search is active)
	if m.searchActive && m.searchTerm != "" {
		if !m.matchesSearch(svc) {
			return false
		}
	}

	// Then check the regular filter
	switch m.currentFilter {
	case FilterAll:
		return true
	case FilterRunning:
		return svc.Status == service.StatusRunning
	case FilterStopped:
		return svc.Status == service.StatusStopped
	case FilterManual:
		return svc.StartType == service.StartTypeManual
	case FilterDisabled:
		return svc.StartType == service.StartTypeDisabled
	case FilterAutomatic:
		return svc.StartType == service.StartTypeAutomatic
	default:
		return true
	}
}

// matchesSearch checks if a service matches the search term
func (m *Model) matchesSearch(svc service.ServiceInfo) bool {
	if m.searchTerm == "" {
		return true
	}

	// Convert search term to lowercase for case-insensitive search
	searchLower := strings.ToLower(m.searchTerm)

	// Search in service name and display name
	return strings.Contains(strings.ToLower(svc.Name), searchLower) ||
		strings.Contains(strings.ToLower(svc.DisplayName), searchLower)
}

// handleSearchKeys handles keys in search mode
func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "ctrl+c":
		// Exit search mode without applying search
		m.exitSearchMode()
		return m, nil

	case "enter":
		// Apply search and exit search mode
		m.applySearch()
		m.exitSearchMode()
		return m, nil

	case "backspace":
		// Remove last character from search term
		if len(m.searchTerm) > 0 {
			m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
		}
		return m, nil

	default:
		// Add character to search term (only printable characters)
		if len(msg.String()) == 1 {
			r := []rune(msg.String())[0]
			if r >= 32 && r <= 126 { // Printable ASCII characters
				m.searchTerm += msg.String()
			}
		}
		return m, nil
	}
}

// Search-related helper methods
func (m *Model) applySearch() {
	if m.searchTerm == "" {
		m.clearSearch()
		return
	}

	m.searchActive = true
	m.applyFilter() // This will now include search filtering
	m.selectedIndex = 0
	m.currentPage = 0
	m.statusMessage = fmt.Sprintf("Search: '%s' | Results: %d", m.searchTerm, len(m.filteredServices))
}

func (m *Model) clearSearch() {
	m.searchActive = false
	m.searchTerm = ""
	m.applyFilter() // Re-apply filters without search
	m.selectedIndex = 0
	m.currentPage = 0
	m.statusMessage = fmt.Sprintf("Search cleared | Filter: %s (%d services)", m.currentFilter.String(), len(m.filteredServices))
}

func (m *Model) exitSearchMode() {
	m.searchMode = false
	m.state = StateServiceList
}
