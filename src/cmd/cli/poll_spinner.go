package main

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sierrasoftworks/humane-errors-go"
)

var (
	bold  = lipgloss.NewStyle().Bold(true)
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	gray  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)
	blue  = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
)

type pollTriggerMsg struct{}

type pollModel struct {
	ctx          context.Context
	cancel       context.CancelFunc
	spinner      spinner.Model
	attempt      int
	message      string
	result       any
	err          error
	ready        bool
	expectedCode int
	delay        time.Duration
	maxAttempts  int
	pollFunc     func() (any, humane.Error)
	startedAt    time.Time
	showReadyMsg bool
}

func (m pollModel) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		pollOnceCmd(m),
	)
}

func (m pollModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.QuitMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancel()
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case pollTriggerMsg:
		return m, pollOnceCmd(m)

	case pollResultMsg:
		m.attempt++
		if msg.err == nil {
			m.ready = true
			m.result = msg.result
			m.showReadyMsg = time.Since(m.startedAt) > 250*time.Millisecond
			return m, tea.Quit
		}

		// Retry on retryStatus
		if msg.shouldRetry {
			m.delay *= 2
			return m, tea.Tick(m.delay, func(t time.Time) tea.Msg {
				return pollTriggerMsg{}
			})
		}

		// Terminal error
		m.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func (m pollModel) View() string {
	switch {
	case m.ready:
		if m.showReadyMsg {
			return fmt.Sprintf("%s %s\n", green.Render("✓"), bold.Render("kubeconfig is ready"))
		}
		return ""

	case m.err != nil:
		return fmt.Sprintf("%s %s\n\n%s\n", red.Render("✗"), bold.Render("fetching kubeconfig failed!"), m.err.Error())

	default:
		return fmt.Sprintf("%s %s", blue.Render(m.spinner.View()), gray.Render(m.message))
	}
}

type pollResultMsg struct {
	result      any
	err         error
	attempt     int
	shouldRetry bool
}

func pollOnceCmd(m pollModel) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
			return pollResultMsg{err: humane.New("operation timed out")}
		default:
		}

		result, err := m.pollFunc()
		return pollResultMsg{
			result:      result,
			err:         err,
			shouldRetry: err != nil && m.attempt < m.maxAttempts,
		}
	}
}
