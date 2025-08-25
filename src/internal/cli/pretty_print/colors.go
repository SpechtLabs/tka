package pretty_print

import (
	"os"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

type Theme string

const (
	AsciiStyle      Theme = "ascii"
	DarkStyle       Theme = "dark"
	DraculaStyle    Theme = "dracula"
	TokyoNightStyle Theme = "tokyo-night"
	LightStyle      Theme = "light"
	NoTTYStyle      Theme = "notty"
)

func AllThemes() []Theme {
	return []Theme{
		AsciiStyle,
		DarkStyle,
		DraculaStyle,
		TokyoNightStyle,
		LightStyle,
		NoTTYStyle,
	}
}

func AllThemeNames() []string {
	themes := AllThemes()
	names := make([]string, len(themes))
	for i, theme := range themes {
		names[i] = string(theme)
	}
	return names
}

var (
	styleMap = map[Theme]ansi.StyleConfig{
		AsciiStyle:      styles.ASCIIStyleConfig,
		DarkStyle:       styles.DarkStyleConfig,
		DraculaStyle:    styles.DraculaStyleConfig,
		TokyoNightStyle: styles.TokyoNightStyleConfig,
		LightStyle:      styles.LightStyleConfig,
	}

	// defaultStyle     = styles.TokyoNightStyleConfig
	// defaultStyle = styles.DarkStyleConfig
	// baseStyle    = defaultStyle.CodeBlock.Chroma

	// bold   = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.Text.Color)).Bold(true)
	// green  = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.NameAttribute.Color))
	// red    = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.GenericDeleted.Color))
	// gray   = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.KeywordType.Color))
	// blue   = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.LiteralStringEscape.Color))
	// yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(*baseStyle.LiteralString.Color))
)

func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdout.Fd()) ||
		os.Getenv("TERM") == "dumb"
}

func styleColor(style ansi.StylePrimitive) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(*style.Color))
}

func boldStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.Text).Bold(true)
}

func normalStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.Text)
}

func secondaryStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.KeywordType)
}
func errStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.GenericDeleted)
}

func warnStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.LiteralString)
}

func infoStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.LiteralStringEscape)
}

func okStyle(theme Theme) lipgloss.Style {
	return styleColor(styleMap[theme].CodeBlock.Chroma.NameAttribute)
}

func okColor(theme Theme) lipgloss.Color {
	return lipgloss.Color(*styleMap[theme].CodeBlock.Chroma.NameAttribute.Color)
}
