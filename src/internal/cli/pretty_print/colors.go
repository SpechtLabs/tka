package pretty_print

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var (
	bold    = lipgloss.NewStyle().Bold(true)
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	gray    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)
	blue    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)

	// Check if stdout is a terminal
	isTerminal = isatty.IsTerminal(os.Stdout.Fd())
)

func IsTerminal() bool {
	return isTerminal
}
