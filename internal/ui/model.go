package ui

import (
	"fmt"
	"log/slog"
	"sort"
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

// filterOrder is the order shown in the sidebar.
var filterOrder = []FilterType{
	FilterAll,
	FilterRunning,
	FilterStopped,
	FilterAutomatic,
	FilterManual,
	FilterDisabled,
}

// FocusArea identifies which panel currently owns keyboard navigation.
type FocusArea int

const (
	FocusServices FocusArea = iota
	FocusFilters
)

func filterOrderIndex(f FilterType) int {
	for i, candidate := range filterOrder {
		if candidate == f {
			return i
		}
	}
	return 0
}

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

	// Layout and navigation
	focus         FocusArea
	scrollOffset  int
	sidebarIndex  int
	elevated      bool
	statusIsError bool

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
		showHelp:         false, // Start with help hidden
		lastRefresh:      time.Now(),
		currentFilter:    FilterAll,
		filteredServices: make([]service.ServiceInfo, 0),
		searchMode:       false,
		searchTerm:       "",
		searchActive:     false,
		focus:            FocusServices,
		scrollOffset:     0,
		sidebarIndex:     0,
	}
}

// SetElevated records whether the process is running as administrator.
func (m *Model) SetElevated(elevated bool) {
	m.elevated = elevated
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
		m.ensureVisible()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case ServicesRefreshedMsg:
		previous := m.GetSelectedService()
		previousName := ""
		if previous != nil {
			previousName = previous.Name
		}

		m.services = msg.Services
		m.applyFilter()
		if previousName != "" {
			for i := range m.filteredServices {
				if m.filteredServices[i].Name == previousName {
					m.selectedIndex = i
					break
				}
			}
		}
		m.clampSelection()
		m.ensureVisible()
		if m.state == StateLoading || m.state == StateError {
			m.state = StateServiceList
		}
		m.lastRefresh = time.Now()
		return m, nil

	case ServiceOperationMsg:
		m.statusIsError = msg.Error != nil
		m.statusMessage = msg.Message
		if msg.Error != nil {
			return m, nil
		}
		// Refresh services after operation
		return m, m.refreshServicesCmd()

	case StartupTypeChangedMsg:
		m.statusIsError = msg.Error != nil
		m.statusMessage = msg.Message
		if msg.Error != nil {
			return m, nil
		}
		// Refresh services after startup type change
		return m, m.refreshServicesCmd()

	case ErrorMsg:
		if len(m.services) == 0 {
			m.errorMessage = msg.Error.Error()
			m.state = StateError
		} else {
			m.statusIsError = true
			m.statusMessage = msg.Error.Error()
		}
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
	if m.focus == FocusFilters {
		return m.handleFilterKeys(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "w", "F5":
		m.statusIsError = false
		m.statusMessage = "Refreshing services..."
		return m, m.refreshServicesCmd()

	case "h", "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "j", "down":
		m.navigateDown()

	case "k", "up":
		m.navigateUp()

	case "g", "home":
		m.selectedIndex = 0
		m.scrollOffset = 0

	case "G", "end":
		if len(m.filteredServices) > 0 {
			m.selectedIndex = len(m.filteredServices) - 1
			m.ensureVisible()
		}

	case "n", "pgdn", "ctrl+d":
		m.pageDown()

	case "p", "pgup", "ctrl+u":
		m.pageUp()

	case "enter":
		if len(m.filteredServices) > 0 {
			m.state = StateServiceDetails
		}

	case "s":
		return m.startSelectedService()

	case "t":
		return m.stopSelectedServiceDirect()

	case "m":
		return m.changeSelectedServiceStartupTypeDirect()

	case "f":
		return m.cycleFilter()

	case "/":
		m.enterSearch()

	case "esc":
		if m.searchActive {
			m.clearSearch()
		}

	case "tab", "left":
		m.focus = FocusFilters
		m.sidebarIndex = filterOrderIndex(m.currentFilter)

	case "right":
		m.focus = FocusServices

	case "1", "2", "3", "4", "5", "6":
		m.selectFilterByNumber(msg.String())
	}

	return m, nil
}

// handleFilterKeys handles keys while the filter sidebar is focused.
func (m *Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab", "right", "esc":
		m.focus = FocusServices
		return m, nil

	case "enter":
		if m.sidebarIndex >= 0 && m.sidebarIndex < len(filterOrder) {
			m.applyFilterSelection(filterOrder[m.sidebarIndex])
		}
		m.focus = FocusServices
		return m, nil

	case "j", "down":
		m.sidebarIndex = (m.sidebarIndex + 1) % len(filterOrder)

	case "k", "up":
		m.sidebarIndex--
		if m.sidebarIndex < 0 {
			m.sidebarIndex = len(filterOrder) - 1
		}

	case "g", "home":
		m.sidebarIndex = 0

	case "G", "end":
		m.sidebarIndex = len(filterOrder) - 1

	case "f":
		return m.cycleFilter()

	case "1", "2", "3", "4", "5", "6":
		m.selectFilterByNumber(msg.String())
		m.focus = FocusServices

	case "/":
		m.enterSearch()

	case "h", "?":
		m.showHelp = !m.showHelp
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
	if m.selectedIndex < len(m.filteredServices)-1 {
		m.selectedIndex++
	}
	m.ensureVisible()
}

func (m *Model) navigateUp() {
	if len(m.filteredServices) == 0 {
		return
	}
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
	m.ensureVisible()
}

func (m *Model) pageDown() {
	if len(m.filteredServices) == 0 {
		return
	}
	m.selectedIndex += m.listRows()
	if m.selectedIndex >= len(m.filteredServices) {
		m.selectedIndex = len(m.filteredServices) - 1
	}
	m.ensureVisible()
}

func (m *Model) pageUp() {
	if len(m.filteredServices) == 0 {
		return
	}
	m.selectedIndex -= m.listRows()
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	m.ensureVisible()
}

// listRows returns how many service rows fit in the list panel.
func (m *Model) listRows() int {
	// 1 header line + 2 footer lines, 2 border lines, then the list
	// panel's title, column header and separator.
	rows := m.height - 8
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m *Model) clampSelection() {
	if len(m.filteredServices) == 0 {
		m.selectedIndex = 0
		m.scrollOffset = 0
		return
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	if m.selectedIndex >= len(m.filteredServices) {
		m.selectedIndex = len(m.filteredServices) - 1
	}
}

func (m *Model) ensureVisible() {
	m.clampSelection()
	rows := m.listRows()
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
	if m.selectedIndex >= m.scrollOffset+rows {
		m.scrollOffset = m.selectedIndex - rows + 1
	}
	maxOffset := len(m.filteredServices) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// Service operation methods
func (m *Model) startSelectedService() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanStart() {
		m.statusIsError = false
		m.statusMessage = fmt.Sprintf("Service '%s' is already %s", selectedService.DisplayName, selectedService.Status.String())
		return m, nil
	}

	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Starting %s...", selectedService.DisplayName)
	return m, m.performServiceOperation("start", selectedService.Name)
}

func (m *Model) stopSelectedService() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusIsError = true
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]
	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Stopping %s...", selectedService.DisplayName)

	if !selectedService.CanStop() {
		m.statusIsError = true
		m.statusMessage = fmt.Sprintf("Service '%s' cannot be stopped (current status: %s)", selectedService.DisplayName, selectedService.Status.String())
		return m, nil
	}

	if m.config.Behavior.ConfirmStop {
		m.confirmAction = &ConfirmAction{
			Action:      "stop",
			ServiceName: selectedService.Name,
			Message:     fmt.Sprintf("Stop service '%s'?", selectedService.DisplayName),
		}
		m.state = StateConfirmDialog
		return m, nil
	}

	return m, m.performServiceOperation("stop", selectedService.Name)
}

func (m *Model) stopSelectedServiceDirect() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusIsError = true
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanStop() {
		m.statusIsError = false
		m.statusMessage = fmt.Sprintf("Service '%s' is already %s", selectedService.DisplayName, selectedService.Status.String())
		return m, nil
	}

	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Stopping %s...", selectedService.DisplayName)
	return m, m.performServiceOperation("stop", selectedService.Name)
}

func (m *Model) changeSelectedServiceStartupType() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusIsError = true
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanChangeStartupType() {
		m.statusIsError = true
		m.statusMessage = fmt.Sprintf("Startup type for service '%s' cannot be changed", selectedService.DisplayName)
		return m, nil
	}

	newStartType := selectedService.GetNextStartType()
	oldTypeString := selectedService.GetStartTypeString()
	newTypeString := m.getStartTypeString(newStartType)

	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Changing startup type: %s (%s -> %s)",
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
		return m, nil
	}

	return m, m.changeStartupTypeCmd(selectedService.Name, newStartType)
}

func (m *Model) changeSelectedServiceStartupTypeDirect() (tea.Model, tea.Cmd) {
	if len(m.filteredServices) == 0 {
		m.statusIsError = true
		m.statusMessage = "No services available"
		return m, nil
	}

	selectedService := m.filteredServices[m.selectedIndex]

	if !selectedService.CanChangeStartupType() {
		m.statusIsError = true
		m.statusMessage = fmt.Sprintf("Startup type for service '%s' cannot be changed", selectedService.DisplayName)
		return m, nil
	}

	newStartType := selectedService.GetNextStartType()
	oldTypeString := selectedService.GetStartTypeString()
	newTypeString := m.getStartTypeString(newStartType)

	m.statusIsError = false
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
	rows := m.listRows()
	start := m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + rows

	if start >= len(m.filteredServices) {
		return []service.ServiceInfo{}
	}

	if end > len(m.filteredServices) {
		end = len(m.filteredServices)
	}

	return m.filteredServices[start:end]
}

// cycleFilter cycles through the different filter types
func (m *Model) cycleFilter() (tea.Model, tea.Cmd) {
	next := (filterOrderIndex(m.currentFilter) + 1) % len(filterOrder)
	m.applyFilterSelection(filterOrder[next])
	return m, nil
}

// applyFilterSelection switches to a filter and resets the list position.
func (m *Model) applyFilterSelection(f FilterType) {
	m.currentFilter = f
	m.sidebarIndex = filterOrderIndex(f)
	m.applyFilter()
	m.selectedIndex = 0
	m.scrollOffset = 0
	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("%s · %d services", f.String(), len(m.filteredServices))
}

func (m *Model) selectFilterByNumber(key string) {
	if len(key) != 1 {
		return
	}
	n := int(key[0] - '0')
	if n < 1 || n > len(filterOrder) {
		return
	}
	m.applyFilterSelection(filterOrder[n-1])
}

func (m *Model) enterSearch() {
	m.focus = FocusServices
	m.searchMode = true
	m.state = StateSearch
	m.searchTerm = ""
	m.searchActive = false
	m.applyFilter()
	m.selectedIndex = 0
	m.scrollOffset = 0
}

// applyFilter applies the current filter to the services list
func (m *Model) applyFilter() {
	m.filteredServices = make([]service.ServiceInfo, 0)

	for _, svc := range m.services {
		if m.matchesFilter(svc) {
			m.filteredServices = append(m.filteredServices, svc)
		}
	}

	sort.SliceStable(m.filteredServices, func(i, j int) bool {
		left := m.filteredServices[i].DisplayName
		if left == "" {
			left = m.filteredServices[i].Name
		}
		right := m.filteredServices[j].DisplayName
		if right == "" {
			right = m.filteredServices[j].Name
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})
}

// matchesFilter checks if a service matches the current filter
func (m *Model) matchesFilter(svc service.ServiceInfo) bool {
	// First check if it matches the search term (if search is active)
	if m.searchActive && m.searchTerm != "" {
		if !m.matchesSearch(svc) {
			return false
		}
	}

	return matchesFilterType(m.currentFilter, svc)
}

// matchesFilterType reports whether a service belongs to a filter category.
func matchesFilterType(f FilterType, svc service.ServiceInfo) bool {
	switch f {
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

// countForFilter returns the number of services in a filter category.
func (m *Model) countForFilter(f FilterType) int {
	count := 0
	for _, svc := range m.services {
		if matchesFilterType(f, svc) {
			count++
		}
	}
	return count
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
		m.clearSearch()
		m.exitSearchMode()
		return m, nil

	case "enter":
		// Apply search and exit search mode
		m.applySearch()
		m.exitSearchMode()
		return m, nil

	case "backspace":
		runes := []rune(m.searchTerm)
		if len(runes) > 0 {
			m.searchTerm = string(runes[:len(runes)-1])
		}
		m.refreshSearchResults()
		return m, nil

	default:
		// Add character to search term (only printable characters)
		if len(msg.String()) == 1 {
			r := []rune(msg.String())[0]
			if r >= 32 && r <= 126 { // Printable ASCII characters
				m.searchTerm += msg.String()
				m.refreshSearchResults()
			}
		}
		return m, nil
	}
}

// refreshSearchResults applies the current search term live while typing.
func (m *Model) refreshSearchResults() {
	m.searchActive = m.searchTerm != ""
	m.applyFilter()
	m.selectedIndex = 0
	m.scrollOffset = 0
	m.statusIsError = false
	if m.searchActive {
		m.statusMessage = fmt.Sprintf("Search: '%s' · %d results", m.searchTerm, len(m.filteredServices))
	} else {
		m.statusMessage = ""
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
	m.scrollOffset = 0
	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Search: '%s' · %d results", m.searchTerm, len(m.filteredServices))
}

func (m *Model) clearSearch() {
	m.searchActive = false
	m.searchTerm = ""
	m.applyFilter() // Re-apply filters without search
	m.selectedIndex = 0
	m.scrollOffset = 0
	m.statusIsError = false
	m.statusMessage = fmt.Sprintf("Search cleared · %s · %d services", m.currentFilter.String(), len(m.filteredServices))
}

func (m *Model) exitSearchMode() {
	m.searchMode = false
	m.state = StateServiceList
	m.focus = FocusServices
}
