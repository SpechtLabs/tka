package pretty_print

import (
	"os"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// Theme represents a color theme for CLI output styling.
type Theme string

// Theme constants define the available color themes for CLI output.
const (
	// AsciiStyle uses ASCII-only characters without colors.
	AsciiStyle Theme = "ascii"
	// DarkStyle uses colors optimized for dark terminal backgrounds.
	DarkStyle Theme = "dark"
	// DraculaStyle uses the Dracula color scheme.
	DraculaStyle Theme = "dracula"
	// TokyoNightStyle uses the Tokyo Night color scheme.
	TokyoNightStyle Theme = "tokyo-night"
	// LightStyle uses colors optimized for light terminal backgrounds.
	LightStyle Theme = "light"
	// NoTTYStyle disables colors for non-TTY output.
	NoTTYStyle Theme = "notty"
	// MarkdownStyle outputs raw markdown without rendering.
	MarkdownStyle Theme = "markdown"
)

// AllThemes returns a slice containing all available themes.
func AllThemes() []Theme {
	return []Theme{
		AsciiStyle,
		DarkStyle,
		DraculaStyle,
		TokyoNightStyle,
		LightStyle,
		NoTTYStyle,
		MarkdownStyle,
	}
}

// AllThemeNames returns a slice of theme names as strings.
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
		NoTTYStyle:      styles.NoTTYStyleConfig,
		MarkdownStyle:   styles.NoTTYStyleConfig,
	}
)

// IsTerminal returns true if stdout is connected to an interactive terminal.
func IsTerminal() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd()) ||
		isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func styleColor(style ansi.StylePrimitive) lipgloss.Style {
	if style.Color == nil {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(*style.Color))
}

// styleFromTheme safely obtains a style using the provided getter. If any
// intermediate nil dereference would occur, it returns a neutral style.
func styleFromTheme(getter func() ansi.StylePrimitive) (st lipgloss.Style) {
	defer func() {
		if r := recover(); r != nil {
			st = lipgloss.NewStyle()
		}
	}()
	return styleColor(getter())
}

func boldStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.Text }).Bold(true)
}

func normalStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.Text })
}

func italicStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.Text }).Italic(true)
}

func secondaryStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.KeywordType })
}

func errStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.GenericDeleted })
}

func warnStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.LiteralString })
}

func infoStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.LiteralStringEscape })
}

func okStyle(theme Theme) lipgloss.Style {
	return styleFromTheme(func() ansi.StylePrimitive { return styleMap[theme].CodeBlock.Chroma.NameAttribute })
}

func okColor(theme Theme) (c lipgloss.Color) {
	defer func() {
		if r := recover(); r != nil {
			c = lipgloss.Color("10")
		}
	}()
	colorPtr := styleMap[theme].CodeBlock.Chroma.NameAttribute.Color
	if colorPtr == nil {
		return lipgloss.Color("10")
	}
	return lipgloss.Color(*colorPtr)
}
