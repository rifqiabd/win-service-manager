package ui

import (
	"strings"
	"testing"

	"win-service-manager/internal/config"
	"win-service-manager/internal/service"

	"github.com/charmbracelet/lipgloss"
)

func testModel(width, height int) *Model {
	services := []service.ServiceInfo{
		{
			Name:         "Audiosrv",
			DisplayName:  "Windows Audio",
			Description:  "Manages audio for Windows-based programs. If this service is stopped, audio devices and effects will not function properly.",
			Status:       service.StatusRunning,
			StartType:    service.StartTypeAutomatic,
			ProcessID:    1234,
			Dependencies: []string{"AudioEndpointBuilder"},
		},
		{
			Name:        "BITS",
			DisplayName: "Background Intelligent Transfer Service",
			Description: "Transfers files in the background using idle network bandwidth.",
			Status:      service.StatusStopped,
			StartType:   service.StartTypeManual,
		},
		{
			Name:        "DiagTrack",
			DisplayName: "Connected User Experiences and Telemetry",
			Status:      service.StatusStopped,
			StartType:   service.StartTypeDisabled,
		},
	}

	m := &Model{
		state:            StateServiceList,
		width:            width,
		height:           height,
		ready:            true,
		config:           config.DefaultConfig(),
		services:         services,
		filteredServices: services,
		currentFilter:    FilterAll,
		selectedIndex:    0,
		focus:            FocusServices,
		elevated:         true,
	}

	return m
}

func TestViewFitsTerminal(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{120, 35},
		{100, 30},
		{80, 24},
		{60, 20},
		{40, 15},
	}

	for _, size := range sizes {
		m := testModel(size.width, size.height)
		output := m.View()
		lines := strings.Split(output, "\n")

		if len(lines) > size.height {
			t.Errorf("%dx%d: rendered %d lines, want at most %d", size.width, size.height, len(lines), size.height)
		}

		for i, line := range lines {
			if got := lipgloss.Width(line); got > size.width {
				t.Errorf("%dx%d: line %d is %d cells wide, want at most %d: %q",
					size.width, size.height, i, got, size.width, line)
			}
		}
	}
}

func TestViewShowsServiceAndFilters(t *testing.T) {
	m := testModel(120, 35)
	output := m.View()

	for _, want := range []string{
		"Windows Service Manager",
		"SERVICES",
		"FILTERS",
		"DETAILS",
		"Windows Audio",
		"Running",
		"ADMIN",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
}

func TestSearchViewFitsTerminal(t *testing.T) {
	m := testModel(90, 25)
	m.state = StateSearch
	m.searchTerm = "a-very-long-search-term-that-should-not-overflow-the-footer"
	m.searchActive = true
	m.applyFilter()

	output := m.View()
	for i, line := range strings.Split(output, "\n") {
		if got := lipgloss.Width(line); got > m.width {
			t.Errorf("line %d is %d cells wide, want at most %d: %q", i, got, m.width, line)
		}
	}
}

func TestDetailsViewFitsTerminal(t *testing.T) {
	m := testModel(100, 30)
	m.state = StateServiceDetails

	output := m.View()
	for i, line := range strings.Split(output, "\n") {
		if got := lipgloss.Width(line); got > m.width {
			t.Errorf("line %d is %d cells wide, want at most %d: %q", i, got, m.width, line)
		}
	}
}

func TestHelpViewFitsTerminal(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},
		{60, 20},
		{40, 15},
	}

	for _, size := range sizes {
		m := testModel(size.width, size.height)
		m.showHelp = true
		output := m.View()

		for i, line := range strings.Split(output, "\n") {
			if got := lipgloss.Width(line); got > size.width {
				t.Errorf("%dx%d: help line %d is %d cells wide, want at most %d: %q",
					size.width, size.height, i, got, size.width, line)
			}
		}
	}
}

func TestConfirmDialogFitsTerminal(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},
		{40, 15},
	}

	for _, size := range sizes {
		m := testModel(size.width, size.height)
		m.state = StateConfirmDialog
		m.confirmAction = &ConfirmAction{
			Action:      "stop",
			ServiceName: "Audiosrv",
			Message:     "Stop service 'Windows Audio'? This is a deliberately long confirmation message used to verify that the modal wraps instead of overflowing the terminal.",
		}
		output := m.View()

		for i, line := range strings.Split(output, "\n") {
			if got := lipgloss.Width(line); got > size.width {
				t.Errorf("%dx%d: confirm line %d is %d cells wide, want at most %d: %q",
					size.width, size.height, i, got, size.width, line)
			}
		}
	}
}
