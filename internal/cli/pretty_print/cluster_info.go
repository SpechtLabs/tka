package pretty_print

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spechtlabs/tka/pkg/service/models"
)

func PrintClusterInfo(respBody *models.TkaClusterInfo) {
	options := DefaultOptions()

	// print the first line with no leading spaces and the other lines with 2 leading spaces
	indent := ""
	labels := ""
	line := 0
	for k, v := range respBody.Labels {
		if line > 0 {
			indent = "            "
		}
		labels += fmt.Sprintf("%s%s=%s\n", indent, k, v)
		line++
	}
	labels = strings.TrimSuffix(labels, "\n")

	// Message content
	content := fmt.Sprintf("%s %s\n\n%s %s\n%s %s\n\n%s %s",
		boldStyle(options.Theme).Render("Server URL:"), normalStyle(options.Theme).Render(respBody.ServerURL),
		boldStyle(options.Theme).Render("Insecure:  "), italicStyle(options.Theme).Render(strconv.FormatBool(respBody.InsecureSkipTLSVerify)),
		boldStyle(options.Theme).Render("CA Data:   "), italicStyle(options.Theme).Render(respBody.CAData[:20]+"..."+respBody.CAData[len(respBody.CAData)-5:]),
		boldStyle(options.Theme).Render("Labels:    "), normalStyle(options.Theme).Render(labels),
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
