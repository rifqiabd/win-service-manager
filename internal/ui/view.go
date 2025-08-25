package ui

import (
	"fmt"
	"strings"
	"win-service-manager/internal/service"

	"github.com/charmbracelet/lipgloss"
)

// Color scheme - using basic ANSI colors for better compatibility
var (
	// Primary colors
	primaryColor   = lipgloss.Color("5") // Magenta
	secondaryColor = lipgloss.Color("4") // Blue
	successColor   = lipgloss.Color("2") // Green
	warningColor   = lipgloss.Color("3") // Yellow
	errorColor     = lipgloss.Color("1") // Red
	mutedColor     = lipgloss.Color("8") // Bright Black (Gray)

	// Background colors - using no background for better compatibility
	bgColor       = lipgloss.Color("")  // Default
	surfaceColor  = lipgloss.Color("0") // Black
	selectedColor = lipgloss.Color("6") // Cyan

	// Text colors
	textColor      = lipgloss.Color("7") // White
	mutedTextColor = lipgloss.Color("8") // Bright Black (Gray)
)

// Base styles - simplified for better terminal compatibility
var (
	baseStyle = lipgloss.NewStyle().Foreground(textColor)

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")). // Black text
				Background(lipgloss.Color("6")). // Cyan background
				Bold(true).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedTextColor).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(errorColor)

	confirmDialogStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(warningColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedTextColor).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor)
)

// Service status styles
var (
	runningStatusStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	stoppedStatusStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	pendingStatusStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	unknownStatusStyle = lipgloss.NewStyle().
				Foreground(mutedTextColor).
				Bold(true)
)

// View renders the UI (required by Bubble Tea)
func (m *Model) View() string {
	// Always show something, even if not ready
	if !m.ready {
		return fmt.Sprintf("\n  Loading... (Terminal size: %dx%d)\n  Press 'q' to quit", m.width, m.height)
	}

	switch m.state {
	case StateLoading:
		return m.renderLoading()
	case StateError:
		return m.renderError()
	case StateConfirmDialog:
		// Debug: Log that we're trying to render confirmation dialog
		if m.logger != nil {
			m.logger.Info("rendering confirmation dialog", "action", m.confirmAction)
		}
		return m.renderConfirmDialog()
	case StateServiceDetails:
		return m.renderServiceDetails()
	case StateServiceList:
		return m.renderServiceList()
	case StateSearch:
		return m.renderSearchInput()
	default:
		return fmt.Sprintf("Unknown state: %d\nPress 'q' to quit", m.state)
	}
}

// renderServiceList renders the main service list view
func (m *Model) renderServiceList() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("Windows Service Manager")
	b.WriteString(title + "\n\n")

	// Debug info
	debugInfo := fmt.Sprintf("Terminal: %dx%d | State: %d (%s) | Ready: %t",
		m.width, m.height, m.state, m.getStateName(), m.ready)
	b.WriteString(statusBarStyle.Render(debugInfo) + "\n")

	// Service count info with search status
	var serviceInfo string
	if m.searchActive {
		serviceInfo = fmt.Sprintf("Total Services: %d | Search: '%s' | Results: %d | Selected: %d",
			len(m.services), m.searchTerm, len(m.filteredServices), m.selectedIndex+1)
	} else {
		serviceInfo = fmt.Sprintf("Total Services: %d | Filter: %s | Selected: %d",
			len(m.services), m.currentFilter.String(), m.selectedIndex+1)
	}
	b.WriteString(headerStyle.Render(serviceInfo) + "\n")

	// Show message if no services loaded
	if len(m.services) == 0 {
		b.WriteString("\n" + statusBarStyle.Render("No services loaded yet. Press 'r' to refresh.") + "\n")
		b.WriteString("\nBasic Controls:\n")
		b.WriteString("  r/F5 - Refresh services\n")
		b.WriteString("  q    - Quit\n")
		b.WriteString("  h/?  - Toggle help\n")
		return baseStyle.Render(b.String())
	}

	// Service list header - using proper table format with column separators
	headerRow := fmt.Sprintf("  %-50s │ %-10s │ %-10s │ %-6s",
		"Service Name", "Status", "Startup", "PID")
	headerLine := headerStyle.Render(headerRow)

	// Add table separator line for clear visual separation
	separatorLine := "  " + strings.Repeat("─", 50) + "─┼─" + strings.Repeat("─", 10) + "─┼─" + strings.Repeat("─", 10) + "─┼─" + strings.Repeat("─", 6)
	b.WriteString(headerLine + "\n")
	b.WriteString(statusBarStyle.Render(separatorLine) + "\n")

	// Service list
	visibleServices := m.GetVisibleServices()
	if len(visibleServices) == 0 && len(m.services) > 0 {
		b.WriteString("\n" + statusBarStyle.Render("No services on current page. Use 'p' for previous page.") + "\n")
	} else {
		for i, svc := range visibleServices {
			globalIndex := m.currentPage*m.itemsPerPage + i
			isSelected := globalIndex == m.selectedIndex

			serviceLine := m.formatServiceLine(svc, isSelected)
			b.WriteString(serviceLine + "\n")
		}
	}

	// Selection info
	selectionInfo := fmt.Sprintf("Selected: %d | Page: %d | Items per page: %d",
		m.selectedIndex, m.currentPage, m.itemsPerPage)
	b.WriteString("\n" + statusBarStyle.Render(selectionInfo) + "\n")

	// Add empty lines if we have fewer services than items per page
	for i := len(visibleServices); i < m.itemsPerPage && i < 5; i++ {
		b.WriteString("\n")
	}

	// Pagination info
	currentPage, totalPages := m.GetPageInfo()
	if totalPages > 1 {
		pageInfo := fmt.Sprintf("Page %d of %d", currentPage, totalPages)
		b.WriteString("\n" + statusBarStyle.Render(pageInfo) + "\n")
	}

	// Status message
	if m.statusMessage != "" {
		b.WriteString("\n" + statusBarStyle.Render(m.statusMessage) + "\n")
	}

	// Basic controls info - split into two lines for readability
	controlsLine1 := "Controls: j/k=navigate, Enter=details, s=start, t=stop, m=startup"
	controlsLine2 := "          /=search, esc=clear search, w=refresh, f=filter, h=help, q=quit"
	if m.searchActive {
		controlsLine1 = fmt.Sprintf("Search active: '%s' | %s", m.searchTerm, controlsLine1)
	}
	b.WriteString("\n" + statusBarStyle.Render(controlsLine1) + "\n")
	b.WriteString(statusBarStyle.Render(controlsLine2) + "\n")

	// Help
	if m.showHelp {
		b.WriteString("\n" + m.renderHelp())
	}

	return baseStyle.Render(b.String())
}

// renderServiceDetails renders the detailed view of a service
func (m *Model) renderServiceDetails() string {
	selectedService := m.GetSelectedService()
	if selectedService == nil {
		return m.renderError()
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("Service Details: " + selectedService.DisplayName)
	b.WriteString(title + "\n\n")

	// Service details
	details := [][]string{
		{"Name:", selectedService.Name},
		{"Display Name:", selectedService.DisplayName},
		{"Description:", selectedService.Description},
		{"Status:", m.applyStatusColor(selectedService.Status.String(), selectedService.Status)},
		{"Startup Type:", m.formatStartupType(selectedService.StartType)},
		{"Process ID:", m.formatProcessID(selectedService.ProcessID)},
	}

	for _, detail := range details {
		line := fmt.Sprintf("%-15s %s", detail[0], detail[1])
		b.WriteString(normalItemStyle.Render(line) + "\n")
	}

	// Dependencies
	if len(selectedService.Dependencies) > 0 {
		b.WriteString("\n" + headerStyle.Render("Dependencies:") + "\n")
		for _, dep := range selectedService.Dependencies {
			b.WriteString(normalItemStyle.Render("  • "+dep) + "\n")
		}
	}

	// Available actions
	b.WriteString("\n" + headerStyle.Render("Available Actions:") + "\n")
	actions := m.getAvailableActions(selectedService)
	for _, action := range actions {
		b.WriteString(normalItemStyle.Render("  "+action) + "\n")
	}

	// Help for details view
	b.WriteString("\n" + statusBarStyle.Render("Press ESC to go back, or use action keys (s/t/m)") + "\n")

	return baseStyle.Render(b.String())
}

// renderConfirmDialog renders the confirmation dialog
func (m *Model) renderConfirmDialog() string {
	if m.confirmAction == nil {
		return m.renderServiceList()
	}

	// Try a simpler approach first for better compatibility
	simpleDialog := fmt.Sprintf(`


    ========================================
    CONFIRMATION REQUIRED
    ========================================
    
    ACTION: %s
    SERVICE: %s
    
    %s
    
    [Y] YES, %s
    [N] NO, cancel
    
    Press Y to confirm or N/ESC to cancel
    
    DIALOG IS VISIBLE - SCREEN: %dx%d
    ========================================


`,
		m.confirmAction.Action,
		m.confirmAction.ServiceName,
		m.confirmAction.Message,
		strings.ToUpper(m.confirmAction.Action),
		m.width, m.height)

	// Use a simple centered approach
	return simpleDialog
}

// renderError renders the error state
func (m *Model) renderError() string {
	errorMsg := errorStyle.Render(
		"Error: " + m.errorMessage + "\n\n" +
			"Press any key to continue...")

	return lipgloss.Place(
		m.height, m.width,
		lipgloss.Center, lipgloss.Center,
		errorMsg)
}

// renderLoading renders the loading state
func (m *Model) renderLoading() string {
	loading := normalItemStyle.Render("Loading services...")
	return lipgloss.Place(
		m.height, m.width,
		lipgloss.Center, lipgloss.Center,
		loading)
}

// renderHelp renders the help text
func (m *Model) renderHelp() string {
	helpText := `Keyboard Shortcuts:
  Navigation:
    j/↓     - Move down
    k/↑     - Move up
    g       - Go to top
    G       - Go to bottom
    n/PgDn  - Next page
    p/PgUp  - Previous page

  Actions:
    Enter   - View details
    s       - Start service
    t       - Stop service (direct)
    m       - Change startup type (direct)

  Other:
    /       - Search services
    esc     - Clear search (if active)
    w/F5    - Refresh
    f       - Cycle filter (All/Running/Stopped/Manual/Disabled/Automatic)
    h/?     - Toggle help
    q       - Quit`

	return helpStyle.Render(helpText)
}

// renderSearchInput renders the search input interface
func (m *Model) renderSearchInput() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("Search Services")
	b.WriteString(title + "\n\n")

	// Search prompt
	searchPrompt := fmt.Sprintf("Search: %s", m.searchTerm)
	if len(m.searchTerm) == 0 {
		searchPrompt += "_" // Show cursor when empty
	} else {
		searchPrompt += "_" // Always show cursor
	}
	b.WriteString(headerStyle.Render(searchPrompt) + "\n\n")

	// Instructions
	instructions := `Type to search service names...

Controls:
  Enter    - Apply search
  Escape   - Cancel search
  Backspace - Delete character`

	b.WriteString(statusBarStyle.Render(instructions) + "\n")

	return baseStyle.Render(b.String())
}

// formatServiceLine formats a single service line for the list view
func (m *Model) formatServiceLine(svc service.ServiceInfo, isSelected bool) string {
	// Truncate long names to fit table column
	displayName := svc.DisplayName
	if len(displayName) > 48 {
		displayName = displayName[:45] + "..."
	}

	// Get plain text values for proper width calculation
	statusText := svc.Status.String()
	startupTypeText := m.formatStartupType(svc.StartType)
	pidText := m.formatProcessID(svc.ProcessID)

	// Add selection indicator
	var indicator string
	if isSelected {
		indicator = "► " // Clear arrow pointer
	} else {
		indicator = "  " // Two spaces for alignment
	}

	// Format each column with exact width and padding
	nameCol := fmt.Sprintf("%-50s", displayName)
	statusCol := fmt.Sprintf("%-10s", statusText)
	startupCol := fmt.Sprintf("%-10s", startupTypeText)
	pidCol := fmt.Sprintf("%-6s", pidText)

	// Apply color styling to status column only
	styledStatusCol := m.applyStatusColor(statusCol, svc.Status)

	// Combine columns with separators
	line := fmt.Sprintf("%s%s │ %s │ %s │ %s",
		indicator, nameCol, styledStatusCol, startupCol, pidCol)

	if isSelected {
		return selectedItemStyle.Render(line)
	}
	return normalItemStyle.Render(line)
}

// applyStatusColor applies color to status text while preserving width
func (m *Model) applyStatusColor(statusText string, status service.ServiceStatus) string {
	switch status {
	case service.StatusRunning:
		return runningStatusStyle.Render(statusText)
	case service.StatusStopped:
		return stoppedStatusStyle.Render(statusText)
	case service.StatusStartPending, service.StatusStopPending:
		return pendingStatusStyle.Render(statusText)
	default:
		return unknownStatusStyle.Render(statusText)
	}
}

// formatStartupType formats startup type with shorter labels for table
func (m *Model) formatStartupType(startType uint32) string {
	switch startType {
	case 2: // mgr.StartAutomatic
		return "Auto"
	case 3: // mgr.StartManual
		return "Manual"
	case 4: // mgr.StartDisabled
		return "Disabled"
	default:
		return "Unknown"
	}
}

// formatProcessID formats process ID
func (m *Model) formatProcessID(pid uint32) string {
	if pid == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", pid)
}

// getAvailableActions returns the available actions for a service
func (m *Model) getAvailableActions(svc *service.ServiceInfo) []string {
	var actions []string

	if svc.CanStart() {
		actions = append(actions, "s - Start")
	}
	if svc.CanStop() {
		actions = append(actions, "t - Stop")
	}
	if svc.CanChangeStartupType() {
		nextType := m.getStartTypeString(svc.GetNextStartType())
		actions = append(actions, fmt.Sprintf("m - Change startup type (current: %s, next: %s)",
			svc.GetStartTypeString(), nextType))
	}

	if len(actions) == 0 {
		actions = append(actions, "No actions available")
	}

	return actions
}

// getStateName returns a readable name for the current state
func (m *Model) getStateName() string {
	switch m.state {
	case StateServiceList:
		return "ServiceList"
	case StateServiceDetails:
		return "ServiceDetails"
	case StateConfirmDialog:
		return "ConfirmDialog"
	case StateError:
		return "Error"
	case StateLoading:
		return "Loading"
	case StateSearch:
		return "Search"
	default:
		return "Unknown"
	}
}
