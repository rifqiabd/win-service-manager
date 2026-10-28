package ui

import (
	"fmt"
	"strings"

	"win-service-manager/internal/service"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Lazydocker-inspired palette.
var (
	colAccent     = lipgloss.Color("39")  // bright blue
	colAccentAlt  = lipgloss.Color("45")  // cyan
	colBorder     = lipgloss.Color("240") // gray
	colMuted      = lipgloss.Color("244")
	colText       = lipgloss.Color("252")
	colGreen      = lipgloss.Color("42")
	colRed        = lipgloss.Color("203")
	colYellow     = lipgloss.Color("214")
	colSelectedBg = lipgloss.Color("62")
	colWhite      = lipgloss.Color("231")
	colDark       = lipgloss.Color("16")
)

var (
	appStyle = lipgloss.NewStyle().Foreground(colText)

	titleBadgeStyle = lipgloss.NewStyle().
			Background(colAccent).
			Foreground(colWhite).
			Bold(true).
			Padding(0, 1)

	titleTextStyle = lipgloss.NewStyle().
			Foreground(colAccentAlt).
			Bold(true)

	mutedStyle  = lipgloss.NewStyle().Foreground(colMuted)
	textStyle   = lipgloss.NewStyle().Foreground(colText)
	greenStyle  = lipgloss.NewStyle().Foreground(colGreen)
	redStyle    = lipgloss.NewStyle().Foreground(colRed)
	yellowStyle = lipgloss.NewStyle().Foreground(colYellow)
	accentStyle = lipgloss.NewStyle().Foreground(colAccentAlt).Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder)

	panelActiveStyle = panelStyle.
				BorderForeground(colAccent)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colAccentAlt).
			Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Background(colSelectedBg).
				Foreground(colWhite).
				Bold(true)

	activeFilterStyle = lipgloss.NewStyle().
				Foreground(colAccentAlt).
				Bold(true)

	statusOKStyle  = lipgloss.NewStyle().Foreground(colGreen).Bold(true)
	statusErrStyle = lipgloss.NewStyle().Foreground(colRed).Bold(true)

	adminBadgeStyle = lipgloss.NewStyle().
			Background(colYellow).
			Foreground(colDark).
			Bold(true).
			Padding(0, 1)

	userBadgeStyle = lipgloss.NewStyle().
			Foreground(colMuted).
			Padding(0, 1)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(1, 2)

	errorDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colRed).
				Padding(1, 2)

	confirmDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colYellow).
				Padding(1, 2)

	helpDialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccentAlt).
			Padding(1, 2)

	searchPromptStyle = lipgloss.NewStyle().
				Foreground(colAccentAlt).
				Bold(true)
)

// View renders the UI (required by Bubble Tea).
func (m *Model) View() string {
	if !m.ready {
		return fmt.Sprintf("\n  Loading... (Terminal size: %dx%d)\n  Press 'q' to quit", m.width, m.height)
	}

	if m.showHelp && (m.state == StateServiceList || m.state == StateSearch) {
		return m.renderHelp()
	}

	switch m.state {
	case StateLoading:
		return m.renderLoading()
	case StateError:
		return m.renderError()
	case StateConfirmDialog:
		return m.renderConfirmDialog()
	case StateServiceDetails:
		return m.renderServiceDetailsScreen()
	case StateSearch:
		return m.renderLayout(true)
	case StateServiceList:
		return m.renderLayout(false)
	default:
		return fmt.Sprintf("Unknown state: %d\nPress 'q' to quit", m.state)
	}
}

// renderLayout draws the lazydocker-style three-pane layout.
func (m *Model) renderLayout(searchMode bool) string {
	width := m.width
	height := m.height
	if width < 30 {
		width = 30
	}
	if height < 10 {
		height = 10
	}

	header := m.renderHeader(width)
	footer := m.renderFooter(width, searchMode)

	bodyHeight := height - 3 // header (1) + footer (2)
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	sidebarWidth := 0
	if width >= 78 {
		sidebarWidth = 22
	}

	detailsWidth := 0
	if width >= 112 {
		detailsWidth = 38
	}

	listWidth := width - sidebarWidth - detailsWidth
	if listWidth < 24 {
		listWidth = 24
	}

	panels := make([]string, 0, 3)
	if sidebarWidth > 0 {
		panels = append(panels, m.renderSidebar(sidebarWidth-2, bodyHeight-2))
	}
	panels = append(panels, m.renderList(listWidth-2, bodyHeight-2))
	if detailsWidth > 0 {
		panels = append(panels, m.renderDetails(detailsWidth-2, bodyHeight-2, false))
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, panels...)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// renderHeader draws the top status bar.
func (m *Model) renderHeader(width int) string {
	left := titleBadgeStyle.Render("wsm") + " " + titleTextStyle.Render("Windows Service Manager")

	running := m.countStatus(service.StatusRunning)
	stopped := m.countStatus(service.StatusStopped)

	right := greenStyle.Render(fmt.Sprintf("● %d running", running)) +
		mutedStyle.Render("  ") +
		redStyle.Render(fmt.Sprintf("● %d stopped", stopped)) +
		mutedStyle.Render(fmt.Sprintf("  │  %d services  ", len(m.services)))

	if m.elevated {
		right += adminBadgeStyle.Render("ADMIN")
	} else {
		right += userBadgeStyle.Render("USER")
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncatePlain("wsm  Windows Service Manager", width)
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderFooter draws the status line and the key hints.
func (m *Model) renderFooter(width int, searchMode bool) string {
	var status string
	switch {
	case searchMode:
		term := truncatePlain(m.searchTerm, width-4)
		status = searchPromptStyle.Render(" / ") + textStyle.Render(term) + accentStyle.Render("_")
	case m.statusMessage != "":
		message := truncatePlain(m.statusMessage, width-2)
		if m.statusIsError {
			status = statusErrStyle.Render("✗ " + message)
		} else {
			status = statusOKStyle.Render("✓ " + message)
		}
	default:
		status = mutedStyle.Render(fmt.Sprintf("%d services · filter %s", len(m.filteredServices), m.currentFilter.String()))
	}

	keys := "j/k move · enter details · s start · t stop · m startup · / search · f filter · tab panels · w refresh · ? help · q quit"
	if searchMode {
		keys = "type to filter · enter apply · esc cancel"
	}

	return status + "\n" + mutedStyle.Render(truncatePlain(keys, width))
}

// renderSidebar draws the filter navigation panel.
func (m *Model) renderSidebar(innerWidth, innerHeight int) string {
	if innerWidth < 10 {
		innerWidth = 10
	}

	lines := []string{panelTitleStyle.Render(" FILTERS")}
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", innerWidth)))

	for i, filter := range filterOrder {
		count := m.countForFilter(filter)
		labelWidth := innerWidth - 6
		if labelWidth < 1 {
			labelWidth = 1
		}
		line := fmt.Sprintf("%-*s %3d", labelWidth, filter.String(), count)
		line = truncatePlain(line, innerWidth-2)

		switch {
		case m.focus == FocusFilters && i == m.sidebarIndex:
			lines = append(lines, selectedRowStyle.Width(innerWidth).Render("▸ "+line))
		case filter == m.currentFilter:
			lines = append(lines, activeFilterStyle.Render("▸ "+line))
		default:
			lines = append(lines, textStyle.Render("  "+line))
		}
	}

	lines = fitLines(lines, innerHeight)

	style := panelStyle
	if m.focus == FocusFilters {
		style = panelActiveStyle
	}
	return style.Width(innerWidth).Height(innerHeight).Render(strings.Join(lines, "\n"))
}

// renderList draws the service table.
func (m *Model) renderList(innerWidth, innerHeight int) string {
	if innerWidth < 20 {
		innerWidth = 20
	}

	title := panelTitleStyle.Render(fmt.Sprintf(" SERVICES · %s", m.currentFilter.String()))
	position := ""
	if len(m.filteredServices) > 0 {
		position = mutedStyle.Render(fmt.Sprintf("%d/%d ", m.selectedIndex+1, len(m.filteredServices)))
	}
	lines := []string{padLine(title, position, innerWidth)}

	columns := newServiceColumns(innerWidth)
	header := columns.renderHeader()
	lines = append(lines, mutedStyle.Render(header))
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", innerWidth)))

	rows := innerHeight - len(lines)
	if rows < 1 {
		rows = 1
	}

	visible := m.GetVisibleServices()
	if len(visible) == 0 {
		message := "No services match this filter."
		if len(m.services) == 0 {
			message = "Loading services..."
		}
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render(centerPlain(message, innerWidth)))
	} else {
		for i, svc := range visible {
			index := m.scrollOffset + i
			lines = append(lines, m.renderServiceRow(svc, index == m.selectedIndex, columns))
		}
	}

	if len(m.filteredServices) > rows {
		scrollInfo := fmt.Sprintf(" %d-%d of %d ",
			m.scrollOffset+1,
			min(m.scrollOffset+rows, len(m.filteredServices)),
			len(m.filteredServices),
		)
		if len(lines) > 0 {
			lines[0] = padLine(title, mutedStyle.Render(scrollInfo), innerWidth)
		}
	}

	lines = fitLines(lines, innerHeight)

	style := panelStyle
	if m.focus == FocusServices {
		style = panelActiveStyle
	}
	return style.Width(innerWidth).Height(innerHeight).Render(strings.Join(lines, "\n"))
}

// serviceColumns computes column widths for the current terminal width.
type serviceColumns struct {
	innerWidth   int
	nameWidth    int
	statusWidth  int
	startupWidth int
	pidWidth     int
	showStartup  bool
	showPID      bool
}

func newServiceColumns(innerWidth int) serviceColumns {
	columns := serviceColumns{
		innerWidth:  innerWidth,
		statusWidth: 12,
		showStartup: innerWidth >= 46,
		showPID:     innerWidth >= 60,
	}
	if columns.showStartup {
		columns.startupWidth = 9
	}
	if columns.showPID {
		columns.pidWidth = 7
	}

	fixed := 4 + columns.statusWidth // leading indicator + gap after name + status
	if columns.showStartup {
		fixed += 2 + columns.startupWidth
	}
	if columns.showPID {
		fixed += 2 + columns.pidWidth
	}
	columns.nameWidth = innerWidth - fixed
	if columns.nameWidth < 8 {
		columns.nameWidth = 8
	}
	return columns
}

func (c serviceColumns) renderHeader() string {
	line := fmt.Sprintf("  %-*s  %-*s", c.nameWidth, "SERVICE", c.statusWidth, "STATUS")
	if c.showStartup {
		line += fmt.Sprintf("  %-*s", c.startupWidth, "STARTUP")
	}
	if c.showPID {
		line += fmt.Sprintf("  %-*s", c.pidWidth, "PID")
	}
	return truncatePlain(line, c.innerWidth)
}

func (m *Model) renderServiceRow(svc service.ServiceInfo, selected bool, columns serviceColumns) string {
	name := svc.DisplayName
	if name == "" {
		name = svc.Name
	}
	name = truncatePlain(name, columns.nameWidth)

	statusText := svc.Status.String()
	statusPlain := "● " + statusText
	statusPlain = padRightPlain(statusPlain, columns.statusWidth)

	line := fmt.Sprintf("  %-*s  ", columns.nameWidth, name)

	if selected {
		// Keep the row readable as a single highlighted block.
		line += statusPlain
		if columns.showStartup {
			line += fmt.Sprintf("  %-*s", columns.startupWidth, shortStartup(svc.StartType))
		}
		if columns.showPID {
			line += fmt.Sprintf("  %-*s", columns.pidWidth, formatPID(svc.ProcessID))
		}
		return selectedRowStyle.Width(columns.innerWidth).Render(truncatePlain(line, columns.innerWidth))
	}

	line += m.statusStyle(svc.Status).Render(statusPlain)
	if columns.showStartup {
		line += fmt.Sprintf("  %-*s", columns.startupWidth, shortStartup(svc.StartType))
	}
	if columns.showPID {
		line += fmt.Sprintf("  %-*s", columns.pidWidth, formatPID(svc.ProcessID))
	}
	return line
}

func (m *Model) statusStyle(status service.ServiceStatus) lipgloss.Style {
	switch status {
	case service.StatusRunning:
		return greenStyle
	case service.StatusStopped:
		return redStyle
	case service.StatusStartPending, service.StatusStopPending:
		return yellowStyle
	default:
		return mutedStyle
	}
}

// renderDetails draws the right-hand service details pane.
func (m *Model) renderDetails(innerWidth, innerHeight int, fullScreen bool) string {
	if innerWidth < 16 {
		innerWidth = 16
	}

	title := " DETAILS"
	if fullScreen {
		title = " SERVICE DETAILS"
	}
	lines := []string{panelTitleStyle.Render(title), mutedStyle.Render(strings.Repeat("─", innerWidth))}

	svc := m.GetSelectedService()
	if svc == nil {
		lines = append(lines, "", mutedStyle.Render("Select a service to inspect it."))
	} else {
		lines = append(lines, m.renderField("Name", svc.Name, innerWidth)...)
		lines = append(lines, m.renderField("Display", svc.DisplayName, innerWidth)...)
		lines = append(lines, m.renderStatusField(svc.Status, innerWidth)...)
		lines = append(lines, m.renderField("Startup", svc.GetStartTypeString(), innerWidth)...)
		lines = append(lines, m.renderField("PID", formatPID(svc.ProcessID), innerWidth)...)

		if svc.Description != "" {
			lines = append(lines, "")
			lines = append(lines, mutedStyle.Render(" Description"))
			for _, wrapped := range wrapPlain(svc.Description, innerWidth-2) {
				lines = append(lines, textStyle.Render(" "+wrapped))
			}
		}

		if len(svc.Dependencies) > 0 {
			lines = append(lines, "")
			lines = append(lines, mutedStyle.Render(" Dependencies"))
			for _, dep := range svc.Dependencies {
				lines = append(lines, textStyle.Render(" • "+truncatePlain(dep, innerWidth-3)))
			}
		}

		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render(" Actions"))
		if svc.CanStart() {
			lines = append(lines, greenStyle.Render(" s  start service"))
		}
		if svc.CanStop() {
			lines = append(lines, redStyle.Render(" t  stop service"))
		}
		lines = append(lines, accentStyle.Render(fmt.Sprintf(" m  startup → %s", m.getStartTypeString(svc.GetNextStartType()))))
	}

	lines = fitLines(lines, innerHeight)
	style := panelStyle
	if fullScreen {
		style = panelActiveStyle
	}
	return style.Width(innerWidth).Height(innerHeight).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderField(label, value string, width int) []string {
	if value == "" {
		value = "—"
	}
	labelWidth := 10
	if width < 24 {
		labelWidth = 8
	}
	wrapped := wrapPlain(value, width-labelWidth-2)
	if len(wrapped) == 0 {
		wrapped = []string{"—"}
	}

	lines := make([]string, 0, len(wrapped))
	lines = append(lines, mutedStyle.Render(" "+padRightPlain(label, labelWidth))+" "+textStyle.Render(wrapped[0]))
	for _, extra := range wrapped[1:] {
		lines = append(lines, strings.Repeat(" ", labelWidth+1)+textStyle.Render(extra))
	}
	return lines
}

func (m *Model) renderStatusField(status service.ServiceStatus, width int) []string {
	label := padRightPlain("Status", 10)
	return []string{mutedStyle.Render(" "+label) + " " + m.statusStyle(status).Render("● "+status.String())}
}

// renderServiceDetailsScreen is the zoomed details view opened with Enter.
func (m *Model) renderServiceDetailsScreen() string {
	header := m.renderHeader(m.width)
	footer := m.renderFooter(m.width, false)
	bodyHeight := m.height - 3
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	panel := m.renderDetails(m.width-2, bodyHeight-2, true)
	return lipgloss.JoinVertical(lipgloss.Left, header, panel, footer)
}

// renderConfirmDialog draws a centered confirmation modal.
func (m *Model) renderConfirmDialog() string {
	if m.confirmAction == nil {
		return m.renderLayout(false)
	}

	width := m.width - 8
	if width > 64 {
		width = 64
	}
	if width < 24 {
		width = 24
	}

	messageLines := wrapPlain(m.confirmAction.Message, width-2)
	if len(messageLines) == 0 {
		messageLines = []string{m.confirmAction.Message}
	}

	body := fmt.Sprintf("%s\n\n%s\n\n%s",
		accentStyle.Render(strings.ToUpper(m.confirmAction.Action)),
		textStyle.Render(strings.Join(messageLines, "\n")),
		mutedStyle.Render("y / enter  confirm     n / esc  cancel"),
	)

	dialog := confirmDialogStyle.Width(width).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// renderError renders the fatal error state.
func (m *Model) renderError() string {
	width := m.width - 8
	if width > 72 {
		width = 72
	}
	if width < 24 {
		width = 24
	}

	messageLines := wrapPlain("Error: "+m.errorMessage, width-2)
	if len(messageLines) == 0 {
		messageLines = []string{"Error: " + m.errorMessage}
	}
	body := redStyle.Render(strings.Join(messageLines, "\n")) + "\n\n" + mutedStyle.Render("Press any key to continue...")
	dialog := errorDialogStyle.Width(width).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// renderLoading renders the initial loading state.
func (m *Model) renderLoading() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		accentStyle.Render("Loading services..."))
}

// renderHelp renders the help modal.
func (m *Model) renderHelp() string {
	help := `KEYBOARD SHORTCUTS

Navigation
  j / ↓        move down
  k / ↑        move up
  g / G        top / bottom
  n / p        page down / up
  tab / ← / →  switch panel focus
  1 - 6        jump to filter

Actions
  s            start service
  t            stop service
  m            cycle startup type
  enter        open details
  esc          back / clear search

View
  /            search services
  f            cycle filter
  w / F5       refresh
  h / ?        toggle this help
  q / ctrl+c   quit`

	width := m.width - 8
	if width > 76 {
		width = 76
	}
	if width < 32 {
		width = 32
	}
	dialog := helpDialogStyle.Width(width).Render(help)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// Helpers

func (m *Model) countStatus(status service.ServiceStatus) int {
	count := 0
	for _, svc := range m.services {
		if svc.Status == status {
			count++
		}
	}
	return count
}

func shortStartup(startType uint32) string {
	switch startType {
	case service.StartTypeAutomatic:
		return "Auto"
	case service.StartTypeManual:
		return "Manual"
	case service.StartTypeDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

func formatPID(pid uint32) string {
	if pid == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", pid)
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(value) <= width {
		return value
	}
	return runewidth.Truncate(value, width, "…")
}

func padRightPlain(value string, width int) string {
	current := runewidth.StringWidth(value)
	if current >= width {
		return truncatePlain(value, width)
	}
	return value + strings.Repeat(" ", width-current)
}

func centerPlain(value string, width int) string {
	value = truncatePlain(value, width)
	padding := width - runewidth.StringWidth(value)
	if padding <= 0 {
		return value
	}
	left := padding / 2
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", padding-left)
}

func padLine(left, right string, width int) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		return truncatePlain(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

func fitLines(lines []string, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) > height {
		return lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func wrapPlain(value string, width int) []string {
	if width <= 0 {
		return nil
	}

	var result []string
	for _, paragraph := range strings.Split(value, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		current := ""
		for _, word := range words {
			word = truncatePlain(word, width)
			if current == "" {
				current = word
				continue
			}
			if runewidth.StringWidth(current)+1+runewidth.StringWidth(word) <= width {
				current += " " + word
			} else {
				result = append(result, current)
				current = word
			}
		}
		if current != "" {
			result = append(result, current)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
