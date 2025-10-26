package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spechtlabs/tka/pkg/cluster"
)

// TUI model for displaying gossip state
type gossipModel struct {
	store       cluster.GossipStore
	lastData    []cluster.NodeDisplayData
	highlighted map[string]time.Time
	width       int
	height      int
}

// Update message types
type stateUpdateMsg struct {
	data []cluster.NodeDisplayData
}

type tickMsg time.Time

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#626262")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	localNodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

func newGossipModel(store cluster.GossipStore) *gossipModel {
	return &gossipModel{
		store:       store,
		lastData:    make([]cluster.NodeDisplayData, 0),
		highlighted: make(map[string]time.Time),
	}
}

func (m gossipModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.updateStateCmd(),
	)
}

func (m gossipModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			tickCmd(),
			m.updateStateCmd(),
		)

	case stateUpdateMsg:
		// Check for changes and highlight updated nodes
		for _, newNode := range msg.data {
			for _, oldNode := range m.lastData {
				if newNode.ID == oldNode.ID {
					if newNode.State != oldNode.State || newNode.Version != oldNode.Version {
						m.highlighted[newNode.ID] = time.Now()
					}
					break
				}
			}
		}
		m.lastData = msg.data
		return m, nil
	}

	return m, nil
}

func (m gossipModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Title
	title := fmt.Sprintf("Gossip Cluster Monitor - Node: %s", m.store.GetId())
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n\n")

	// Header
	header := fmt.Sprintf("%-12s %-20s %-8s %-20s %-25s",
		"ID", "Address", "Version", "State", "Last Seen")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	// Separator
	sb.WriteString(strings.Repeat("-", m.width))
	sb.WriteString("\n")

	// Node data
	for _, node := range m.lastData {
		// Check if this node should be highlighted
		isHighlighted := false
		if highlightTime, exists := m.highlighted[node.ID]; exists {
			if time.Since(highlightTime) < 3*time.Second {
				isHighlighted = true
			} else {
				delete(m.highlighted, node.ID)
			}
		}

		// Choose style based on node type and highlight status
		var style lipgloss.Style
		if node.IsLocal {
			style = localNodeStyle
		} else if isHighlighted {
			style = highlightStyle
		} else {
			style = normalStyle
		}

		// Format the node data
		nodeLine := fmt.Sprintf("%-12s %-20s %-8d %-20s %-25s",
			truncateString(node.ID, 12),
			truncateString(node.Address, 20),
			node.Version,
			truncateString(node.State, 20),
			node.LastSeen.Format("2006-01-02 15:04:05"))

		sb.WriteString(style.Render(nodeLine))
		sb.WriteString("\n")
	}

	// Help text
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("Press 'q' or Ctrl+C to quit"))
	sb.WriteString("\n")

	return sb.String()
}

// Helper functions
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m gossipModel) updateStateCmd() tea.Cmd {
	return func() tea.Msg {
		// Cast to TestGossipStore to access GetDisplayData method
		if testStore, ok := m.store.(*cluster.TestGossipStore); ok {
			return stateUpdateMsg{data: testStore.GetDisplayData()}
		}
		return stateUpdateMsg{data: []cluster.NodeDisplayData{}}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
