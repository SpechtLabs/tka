package tui

import (
	"fmt"
	"os"
)

// PrettyPrint prints a message with the global options
func PrettyPrint(lvl PrintLevel, msg string, context ...string) (int, error) {
	return PrettyPrintWithOptions(lvl, msg, context)
}

// PrettyPrintWithOptions formats and prints a message with custom options
func PrettyPrintWithOptions(lvl PrintLevel, msg string, context []string, opts ...Option) (int, error) {
	formatted := FormatWithOptions(lvl, msg, context, opts...)

	// Determine the output writer
	output := os.Stdout
	if lvl == ErrLvl {
		output = os.Stderr
	}

	// Write to the appropriate output
	return fmt.Fprint(output, formatted)
}

// Convenience functions for common levels

// PrintOk prints a message at the "Ok" level with optional context.
func PrintOk(msg string, context ...string) {
	_, _ = PrettyPrint(OkLvl, msg, context...)
}

// PrintInfo logs an informational message with optional context.
func PrintInfo(msg string, context ...string) {
	_, _ = PrettyPrint(InfoLvl, msg, context...)
}

// PrintWarn prints a warning message with the given context using a pre-defined warning level.
func PrintWarn(msg string, context ...string) {
	_, _ = PrettyPrint(WarnLvl, msg, context...)
}

// PrintErrorMessage logs an error message to the standard error stream with optional context.
func PrintErrorMessage(msg string, context ...string) {
	_, _ = PrettyPrint(ErrLvl, msg, context...)
}

// PrintError prints an error message with a humane-errors formatted error.
func PrintError(err error, context ...string) {
	_, _ = PrettyPrintWithOptions(ErrLvl, "", context, WithError(err))
}

// PrintDebug logs a debug-level message with optional context.
func PrintDebug(msg string, context ...string) {
	_, _ = PrettyPrint(DebugLvl, msg, context...)
}
