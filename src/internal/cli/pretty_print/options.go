package pretty_print

import (
	"os"
	"slices"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// PrintOptions controls how messages are formatted
type PrintOptions struct {
	// Theme is the theme to use for the print options
	Theme Theme

	// IndentSize controls the number of spaces used for indentation
	IndentSize int

	// ShowTimestamp determines if a timestamp is shown before the message
	ShowTimestamp bool

	// TimeFormat specifies the format for timestamps (Go time format)
	TimeFormat string

	// NoColor disables colored output
	NoColor bool

	// LevelIcons maps print levels to their display icons
	LevelIcons map[PrintLevel]string

	// IconStyles maps print levels to their display styles
	IconStyles map[PrintLevel]themeStyleFunc

	// ContextStyle defines the style for context lines
	ContextStyle themeStyleFunc

	// MessageStyle defines the style for the main message
	MessageStyle themeStyleFunc

	// MarkdownRenderer defines the style for markdown
	MarkdownRenderer markdownRendererFunc

	// Error holds an error to be rendered with humane-errors formatting
	// This is only used for error printing and is not part of global options
	Error error

	// RenderHumaneError determines whether to use humane error rendering
	// This is true by default if an error is provided
	RenderHumaneError bool

	// NoNewline disables the newline at the end of the message
	NoNewline bool
}

type themeStyleFunc func(theme Theme) lipgloss.Style
type markdownRendererFunc func(theme Theme) *glamour.TermRenderer

// DefaultOptions returns the default print options
func DefaultOptions() *PrintOptions {
	options := &PrintOptions{
		Theme:         TokyoNightStyle,
		IndentSize:    4,
		ShowTimestamp: false,
		TimeFormat:    "15:04:05",
		NoColor:       false,
		LevelIcons: map[PrintLevel]string{
			OkLvl:    "✓",
			InfoLvl:  "ℹ", // "•"
			WarnLvl:  "!",
			ErrLvl:   "✗",
			DebugLvl: "D",
			NoOp:     "",
		},
		IconStyles: map[PrintLevel]themeStyleFunc{
			NoOp:     secondaryStyle,
			OkLvl:    okStyle,
			InfoLvl:  infoStyle,
			WarnLvl:  warnStyle,
			ErrLvl:   errStyle,
			DebugLvl: secondaryStyle,
		},
		ContextStyle: secondaryStyle,
		MessageStyle: normalStyle,
		MarkdownRenderer: func(theme Theme) *glamour.TermRenderer {
			var markdownRenderer *glamour.TermRenderer
			if IsTerminal() {
				markdownRenderer, _ = glamour.NewTermRenderer(
					glamour.WithStandardStyle(string(theme)),
					glamour.WithWordWrap(0),
				)
			} else {
				markdownRenderer, _ = glamour.NewTermRenderer(
					glamour.WithStandardStyle(string(NoTTYStyle)),
					glamour.WithWordWrap(0),
				)
			}
			return markdownRenderer
		},
		Error:             nil,
		RenderHumaneError: true,
		NoNewline:         false,
	}

	// Theme selection via config
	theme := viper.GetString("output.theme")
	if theme != "" && slices.Contains(AllThemeNames(), theme) {
		options.Theme = Theme(theme)
	}

	// Auto-select NoTTY theme when not in a TTY
	if !IsTerminal() {
		options.Theme = NoTTYStyle
	}

	// Auto-detect no color via NO_COLOR or non-TTY
	if _, hasNoColor := os.LookupEnv("NO_COLOR"); hasNoColor || !IsTerminal() {
		options.NoColor = true
	}

	return options
}

// Option is a function that modifies PrintOptions
type Option func(*PrintOptions)

// WithIndentSize sets the indent size
func WithIndentSize(size int) Option {
	return func(o *PrintOptions) {
		o.IndentSize = size
	}
}

// WithTimestamp enables timestamp display
func WithTimestamp(enabled bool) Option {
	return func(o *PrintOptions) {
		o.ShowTimestamp = enabled
	}
}

// WithTimeFormat sets the time format
func WithTimeFormat(format string) Option {
	return func(o *PrintOptions) {
		o.TimeFormat = format
	}
}

// WithNoColor enables or disables colors
func WithNoColor(noColor bool) Option {
	return func(o *PrintOptions) {
		o.NoColor = noColor
	}
}

// WithIcon sets a custom icon for a print level
func WithIcon(level PrintLevel, icon string) Option {
	return func(o *PrintOptions) {
		o.LevelIcons[level] = icon
	}
}

// WithStyle sets a custom style for a print level
func WithStyle(level PrintLevel, style themeStyleFunc) Option {
	return func(o *PrintOptions) {
		o.IconStyles[level] = style
	}
}

// WithContextStyle sets the style for context lines
func WithContextStyle(style themeStyleFunc) Option {
	return func(o *PrintOptions) {
		o.ContextStyle = style
	}
}

// WithMessageStyle sets the style for the main message
func WithMessageStyle(style themeStyleFunc) Option {
	return func(o *PrintOptions) {
		o.MessageStyle = style
	}
}

// WithError sets an error to be rendered using humane-errors formatting
// This option is only relevant for error printing functions
func WithError(err error) Option {
	return func(o *PrintOptions) {
		o.Error = err
		o.RenderHumaneError = true
	}
}

// WithoutHumaneErrorRendering disables humane error rendering even if an error is provided
func WithoutHumaneErrorRendering() Option {
	return func(o *PrintOptions) {
		o.RenderHumaneError = false
	}
}

// WithoutNewline disables the newline at the end of the message
func WithoutNewline() Option {
	return func(o *PrintOptions) {
		o.NoNewline = true
	}
}

// WithTheme sets the theme for the print options
func WithTheme(theme Theme) Option {
	return func(o *PrintOptions) {
		o.Theme = theme
	}
}
