package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
)

func PrintLoginSuccess(respBody tailscale.UserLoginResponse) {
	// Parse time (optional formatting)
	untilTime, _ := time.Parse(time.RFC3339, respBody.Until)
	formattedUntil := untilTime.Format(time.RFC1123)

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true).
		PaddingBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Width(10).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	// Message content
	content := fmt.Sprintf("%s %s\n%s %s\n%s %s",
		labelStyle.Render("User:"),
		valueStyle.Render(respBody.Username),
		labelStyle.Render("Role:"),
		valueStyle.Render(respBody.Role),
		labelStyle.Render("Until:"),
		valueStyle.Render(formattedUntil),
	)

	// Full render
	output := headerStyle.Render("✓ Successfully signed in!") + "\n" +
		boxStyle.Render(content)

	fmt.Fprintln(os.Stdout, output)
}
