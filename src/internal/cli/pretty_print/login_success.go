package pretty_print

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spechtlabs/tka/pkg/models"
)

var (
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Width(12).Bold(true)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

func PrintLoginInformation(respBody *models.UserLoginResponse) {
	// Parse time (optional formatting)
	untilTime, _ := time.Parse(time.RFC3339, respBody.Until)
	formattedUntil := untilTime.Format(time.RFC1123)

	// Message content
	content := fmt.Sprintf("%s %s\n%s %s\n%s %s",
		labelStyle.Render("User:"),
		valueStyle.Render(respBody.Username),
		labelStyle.Render("Role:"),
		valueStyle.Render(respBody.Role),
		labelStyle.Render("Until:"),
		valueStyle.Render(formattedUntil),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10")).
		Padding(0, 1).
		MarginTop(0).
		MarginBottom(0).
		MarginLeft(2)

	_, _ = fmt.Fprintln(os.Stdout, boxStyle.Render(content))
}

func PrintLoginInfoWithProvisioning(respBody *models.UserLoginResponse, httpCode int) {
	// Parse time (optional formatting)
	untilTime, _ := time.Parse(time.RFC3339, respBody.Until)
	formattedUntil := untilTime.Format(time.RFC1123)
	formattedProvisioned := "False"
	if httpCode == 200 {
		formattedProvisioned = "True"
	}

	// Message content
	content := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s",
		labelStyle.Render("User:"),
		valueStyle.Render(respBody.Username),
		labelStyle.Render("Role:"),
		valueStyle.Render(respBody.Role),
		labelStyle.Render("Until:"),
		valueStyle.Render(formattedUntil),
		labelStyle.Render("Provisioned:"),
		valueStyle.Render(formattedProvisioned),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10")).
		Padding(0, 1).
		MarginTop(0).
		MarginBottom(0).
		MarginLeft(2)

	_, _ = fmt.Fprintln(os.Stdout, boxStyle.Render(content))
}
