package tui

import "github.com/charmbracelet/lipgloss"

// PrintOptions controls how messages are formatted
type PrintOptions struct {
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
	IconStyles map[PrintLevel]lipgloss.Style

	// ContextStyle defines the style for context lines
	ContextStyle lipgloss.Style

	// MessageStyle defines the style for the main message
	MessageStyle lipgloss.Style

	// Error holds an error to be rendered with humane-errors formatting
	// This is only used for error printing and is not part of global options
	Error error

	// RenderHumaneError determines whether to use humane error rendering
	// This is true by default if an error is provided
	RenderHumaneError bool
}

// DefaultOptions returns the default print options
func DefaultOptions() PrintOptions {
	return PrintOptions{
		IndentSize:    4,
		ShowTimestamp: false,
		TimeFormat:    "15:04:05",
		NoColor:       !isTerminal,
		LevelIcons: map[PrintLevel]string{
			OkLvl:    "✓",
			InfoLvl:  "•",
			WarnLvl:  "!",
			ErrLvl:   "✗",
			DebugLvl: "D",
			NoOp:     "",
		},
		IconStyles: map[PrintLevel]lipgloss.Style{
			NoOp:     gray,
			OkLvl:    green,
			InfoLvl:  blue,
			WarnLvl:  yellow,
			ErrLvl:   red,
			DebugLvl: magenta,
		},
		ContextStyle:      gray,
		MessageStyle:      bold,
		Error:             nil,
		RenderHumaneError: true,
	}
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
func WithStyle(level PrintLevel, style lipgloss.Style) Option {
	return func(o *PrintOptions) {
		o.IconStyles[level] = style
	}
}

// WithContextStyle sets the style for context lines
func WithContextStyle(style lipgloss.Style) Option {
	return func(o *PrintOptions) {
		o.ContextStyle = style
	}
}

// WithMessageStyle sets the style for the main message
func WithMessageStyle(style lipgloss.Style) Option {
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
