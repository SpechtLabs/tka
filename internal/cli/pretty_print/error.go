package pretty_print

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sierrasoftworks/humane-errors-go"
)

func renderHumaneError(err error) string {
	var he humane.Error
	if !errors.As(err, &he) {
		// Get a copy of the global options (thread-safe)
		options := DefaultOptions()
		return errStyle(options.Theme).Render(fmt.Sprintf("✗ %s", err.Error()))
	}

	var b strings.Builder

	// Collect error chain and advice
	var causes []string
	advice := make([]string, 0)
	cur := error(he)
	for cur != nil {
		causes = append(causes, cur.Error())

		if adv, ok := cur.(interface {
			Advice() []string
		}); ok {
			advice = append(adv.Advice(), advice...)
		}

		cur = errors.Unwrap(cur)
	}

	// IconStyles
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))   // red
	section := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8"))  // gray
	bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("•") // blue
	code := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("245"))

	// Message
	b.WriteString(header.Render("✖ " + he.Error()))
	b.WriteString("\n\n")

	// Advice
	if len(advice) > 0 {
		b.WriteString(section.Render("💡 What you can do:") + "\n")
		for _, tip := range advice {
			b.WriteString("  " + bullet + " " + tip + "\n")
		}
		b.WriteString("\n")
	}

	// Causes
	if len(causes) > 1 {
		b.WriteString(section.Render("🔎 Root causes:") + "\n")
		for _, c := range causes[1:] {
			b.WriteString("  " + bullet + " " + code.Render(c) + "\n")
		}
	}

	return b.String()
}
