package pretty_print

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spechtlabs/tka/pkg/service/auth/models"
)

func PrintLoginInformation(respBody *models.UserLoginResponse) {
	options := DefaultOptions()

	// Parse time (optional formatting)
	untilTime, _ := time.Parse(time.RFC3339, respBody.Until)
	formattedUntil := untilTime.Format(time.RFC1123)

	// Message content
	content := fmt.Sprintf("%s %s\n%s %s\n%s %s",
		boldStyle(options.Theme).Render("User: "), normalStyle(options.Theme).Render(respBody.Username),
		boldStyle(options.Theme).Render("Role: "), normalStyle(options.Theme).Render(respBody.Role),
		boldStyle(options.Theme).Render("Until:"), normalStyle(options.Theme).Render(formattedUntil),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(okColor(options.Theme)).
		Padding(0, 1).
		MarginTop(0).
		MarginBottom(0).
		MarginLeft(4)

	_, _ = fmt.Fprintln(os.Stdout, boxStyle.Render(content))
}

func PrintLoginInfoWithProvisioning(respBody *models.UserLoginResponse, httpCode int) {
	options := DefaultOptions()
	// Parse time (optional formatting)
	untilTime, _ := time.Parse(time.RFC3339, respBody.Until)
	formattedUntil := untilTime.Format(time.RFC1123)
	formattedProvisioned := "False"
	if httpCode == 200 {
		formattedProvisioned = "True"
	}

	// Message content
	content := fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s",
		boldStyle(options.Theme).Render("User:       "), normalStyle(options.Theme).Render(respBody.Username),
		boldStyle(options.Theme).Render("Role:       "), normalStyle(options.Theme).Render(respBody.Role),
		boldStyle(options.Theme).Render("Until:      "), normalStyle(options.Theme).Render(formattedUntil),
		boldStyle(options.Theme).Render("Provisioned:"), normalStyle(options.Theme).Render(formattedProvisioned),
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
