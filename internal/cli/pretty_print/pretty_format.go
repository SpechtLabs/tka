package pretty_print

import (
	"fmt"
	"strings"
	"time"
)

type PrintLevel int

const (
	NoOp PrintLevel = iota
	DebugLvl
	InfoLvl
	OkLvl
	WarnLvl
	ErrLvl
)

// FormatWithOptions formats a message with custom options and returns it as a string
func FormatWithOptions(lvl PrintLevel, msg string, context []string, opts ...Option) string {
	// Get a copy of the global options (thread-safe)
	options := DefaultOptions()

	// Apply any function-specific options
	for _, opt := range opts {
		opt(options)
	}

	// If this is an error message, and we have an error object with humane rendering enabled,
	// use the humane error renderer
	if lvl == ErrLvl && options.Error != nil && options.RenderHumaneError {
		return renderHumaneError(options.Error)
	}

	// Get the appropriate icon and style it
	icon, ok := options.LevelIcons[lvl]
	if !ok {
		icon = options.LevelIcons[InfoLvl]
	}

	style, ok := options.IconStyles[lvl]
	if !ok {
		style = options.IconStyles[InfoLvl]
	}

	// Apply the styling if colors are enabled
	var status string
	if options.NoColor {
		status = icon
	} else {
		status = style(options.Theme).Render(icon)
	}

	// Apply message styling
	var message string
	if options.NoColor {
		message = msg
	} else {
		message = options.MessageStyle(options.Theme).Render(msg)
	}

	// Add timestamp if requested
	timestamp := ""
	if options.ShowTimestamp {
		timestamp = time.Now().Format(options.TimeFormat) + " "
	}

	// Process context with proper indentation
	indent := strings.Repeat(" ", options.IndentSize)
	var additionalContext string
	for _, c := range context {
		var contextText string
		if options.NoColor {
			contextText = c
		} else {
			contextText = options.ContextStyle(options.Theme).Render(c)
		}
		additionalContext += fmt.Sprintf("\n%s%s", indent, contextText)
	}

	// Create the complete log line

	newline := "\n"
	if options.NoNewline {
		newline = ""
	}

	return fmt.Sprintf("%s%s %s%s%s", timestamp, status, message, additionalContext, newline)
}

// Format formats a message with the global options and returns it as a string
func Format(lvl PrintLevel, msg string, context ...string) string {
	return FormatWithOptions(lvl, msg, context)
}

// Formatting convenience functions

// FormatOk formats a message at the "Ok" level with optional context.
func FormatOk(msg string, context ...string) string {
	return Format(OkLvl, msg, context...)
}

// FormatInfo formats an informational message with optional context.
func FormatInfo(msg string, context ...string) string {
	return Format(InfoLvl, msg, context...)
}

// FormatWarn formats a warning message with the given context.
func FormatWarn(msg string, context ...string) string {
	return Format(WarnLvl, msg, context...)
}

// FormatErrorMessage formats an error message with optional context.
func FormatErrorMessage(msg string, context ...string) string {
	return Format(ErrLvl, msg, context...)
}

// FormatError formats an error using humane-errors formatting.
func FormatError(err error, context ...string) string {
	return FormatWithOptions(ErrLvl, "", context, WithError(err))
}

// FormatDebug formats a debug-level message with optional context.
func FormatDebug(msg string, context ...string) string {
	return Format(DebugLvl, msg, context...)
}
