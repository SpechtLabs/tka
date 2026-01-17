package pretty_print

import (
	"fmt"
	"os"
	"strings"

	humane "github.com/sierrasoftworks/humane-errors-go"
)

// PrettyPrint prints a message with the global options.
func PrettyPrint(lvl PrintLevel, msg string, context ...string) (int, humane.Error) {
	return PrettyPrintWithOptions(lvl, msg, context)
}

// PrettyPrintWithOptions formats and prints a message with custom options.
func PrettyPrintWithOptions(lvl PrintLevel, msg string, context []string, opts ...Option) (int, humane.Error) {
	formatted := FormatWithOptions(lvl, msg, context, opts...)

	// Determine the output writer
	output := os.Stdout
	if lvl == ErrLvl {
		output = os.Stderr
	}

	if strings.HasPrefix(strings.ToLower(msg), "error") {
		output = os.Stderr
	}

	// Write to the appropriate output
	n, err := fmt.Fprint(output, formatted)
	if err != nil {
		return n, humane.Wrap(err, "failed to write formatted output", "check that stdout/stderr is writable")
	}
	return n, nil
}

// Convenience functions for common levels

// PrintOk prints a message at the "Ok" level with optional context.
func PrintOk(msg string, context ...string) {
	_, _ = PrettyPrint(OkLvl, msg, context...)
}

// PrintInfoIcon logs an informational message with optional context.
func PrintInfoIcon(icon, msg string, context ...string) {
	_, _ = PrettyPrintWithOptions(InfoLvl,
		msg,
		context,
		WithIcon(InfoLvl, icon),
	)
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
