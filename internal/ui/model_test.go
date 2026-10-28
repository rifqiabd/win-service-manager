package ui

import (
	"fmt"
	"testing"

	"win-service-manager/internal/service"

	tea "github.com/charmbracelet/bubbletea"
)

func runeKey(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func TestFilterSidebarSelection(t *testing.T) {
	m := testModel(100, 30)
	m.focus = FocusFilters
	m.sidebarIndex = 0

	if _, _ = m.handleKeyPress(runeKey("j")); m.sidebarIndex != 1 {
		t.Fatalf("sidebar index = %d, want 1", m.sidebarIndex)
	}
	if _, _ = m.handleKeyPress(tea.KeyMsg{Type: tea.KeyEnter}); m.currentFilter != FilterRunning {
		t.Fatalf("filter = %v, want Running", m.currentFilter)
	}
	if m.focus != FocusServices {
		t.Fatalf("focus = %v, want services", m.focus)
	}
	if len(m.filteredServices) != 1 {
		t.Fatalf("filtered services = %d, want 1", len(m.filteredServices))
	}
}

func TestDirectFilterShortcut(t *testing.T) {
	m := testModel(100, 30)
	m.handleKeyPress(runeKey("3")) // Stopped

	if m.currentFilter != FilterStopped {
		t.Fatalf("filter = %v, want Stopped", m.currentFilter)
	}
	if len(m.filteredServices) != 2 {
		t.Fatalf("filtered services = %d, want 2", len(m.filteredServices))
	}
}

func TestLiveSearch(t *testing.T) {
	m := testModel(100, 30)
	m.enterSearch()

	for _, char := range "audio" {
		m.handleSearchKeys(runeKey(string(char)))
	}

	if !m.searchActive {
		t.Fatal("search should be active")
	}
	if len(m.filteredServices) != 1 {
		t.Fatalf("filtered services = %d, want 1", len(m.filteredServices))
	}
	if m.filteredServices[0].Name != "Audiosrv" {
		t.Fatalf("matched %q, want Audiosrv", m.filteredServices[0].Name)
	}
}

func TestScrollKeepsSelectionVisible(t *testing.T) {
	m := testModel(100, 24)
	m.services = make([]service.ServiceInfo, 100)
	for i := range m.services {
		m.services[i] = service.ServiceInfo{
			Name:        fmt.Sprintf("svc%03d", i),
			DisplayName: fmt.Sprintf("Service %03d", i),
			Status:      service.StatusStopped,
			StartType:   service.StartTypeManual,
		}
	}
	m.applyFilter()

	for i := 0; i < 40; i++ {
		m.navigateDown()
	}

	rows := m.listRows()
	if m.scrollOffset == 0 {
		t.Fatal("scroll offset should have advanced")
	}
	if m.selectedIndex < m.scrollOffset || m.selectedIndex >= m.scrollOffset+rows {
		t.Fatalf("selection %d is outside visible range %d..%d", m.selectedIndex, m.scrollOffset, m.scrollOffset+rows-1)
	}
}

func TestStatusPreservedOnRefresh(t *testing.T) {
	m := testModel(100, 30)
	m.selectedIndex = 2
	selected := m.filteredServices[2].Name

	m.Update(ServicesRefreshedMsg{Services: append([]service.ServiceInfo(nil), m.services...)})

	if m.GetSelectedService() == nil || m.GetSelectedService().Name != selected {
		t.Fatalf("selection was not preserved after refresh")
	}
}

func TestSearchModeSurvivesRefresh(t *testing.T) {
	m := testModel(100, 30)
	m.enterSearch()
	for _, char := range "audio" {
		m.handleSearchKeys(runeKey(string(char)))
	}

	m.Update(ServicesRefreshedMsg{Services: append([]service.ServiceInfo(nil), m.services...)})

	if m.state != StateSearch {
		t.Fatalf("state = %v, want StateSearch", m.state)
	}
	if len(m.filteredServices) != 1 || m.filteredServices[0].Name != "Audiosrv" {
		t.Fatalf("search results were not preserved: %#v", m.filteredServices)
	}
}
