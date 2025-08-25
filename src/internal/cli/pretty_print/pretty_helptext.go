package pretty_print

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func FormatHelpText(cmd *cobra.Command, _ []string) string {
	style := func(s string, render func(...string) string) string {
		if !IsTerminal() {
			return s
		}
		return render(s)
	}

	var b strings.Builder

	b.WriteString(style("Usage:", blue.Render))
	b.WriteString("\n")
	b.WriteString(style("  "+cmd.UseLine(), bold.Render))
	b.WriteString("\n\n")

	// Long description
	b.WriteString(style("Description:", blue.Render))
	b.WriteString("\n")
	if strings.TrimSpace(cmd.Long) != "" {
		for _, line := range strings.Split(cmd.Long, "\n") {
			b.WriteString("  ")
			if strings.HasPrefix(line, "$ ") || strings.HasPrefix(line, "  ") {
				b.WriteString(style(line, gray.Render))
			} else {
				for i, segment := range strings.Split(line, "\"") {
					if i%2 == 1 { // inside quotes
						b.WriteString(style(segment, italic.Render))
					} else {
						b.WriteString(segment)
					}
				}
			}
			b.WriteString("\n")
		}

		b.WriteString("\n\n")
	}

	// Examples
	if strings.TrimSpace(cmd.Example) != "" {
		b.WriteString(style("Examples:", blue.Render))
		b.WriteString("\n")
		for _, line := range strings.Split(strings.TrimRight(cmd.Example, "\n"), "\n") {
			if strings.TrimSpace(line) == "" {
				b.WriteString("\n")
				continue
			}

			if strings.HasPrefix(line, "# ") {
				b.WriteString(style("  "+line, gray.Render))
			} else {
				b.WriteString("  " + line)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Args / ValidArgs
	if len(cmd.ValidArgs) > 0 {
		b.WriteString(style("Valid Args:", blue.Render))
		b.WriteString("\n  ")
		b.WriteString(strings.Join(cmd.ValidArgs, ", "))
		b.WriteString("\n\n")
	}

	// Local flags
	if fs := cmd.Flags(); fs != nil && fs.HasFlags() {
		b.WriteString(style("Flags:", blue.Render))
		b.WriteString("\n")
		b.WriteString(formatFlagSet(fs))
		b.WriteString("\n")
	}

	// Inherited (persistent) flags
	if pfs := cmd.InheritedFlags(); pfs != nil && pfs.HasFlags() {
		b.WriteString(style("Global Flags:", blue.Render))
		b.WriteString("\n")
		b.WriteString(formatFlagSet(pfs))
	}

	return strings.TrimRight(b.String(), "\n")
}

func formatFlagSet(fs *pflag.FlagSet) string {
	// Collect and sort flags for stable output
	var entries []string
	fs.VisitAll(func(f *pflag.Flag) {
		var parts []string
		if f.Shorthand != "" {
			parts = append(parts, fmt.Sprintf("-%s", f.Shorthand))
		}
		parts = append(parts, fmt.Sprintf("--%s", f.Name))

		flagSpec := strings.Join(parts, ", ")
		displaySpec := flagSpec
		if IsTerminal() {
			displaySpec = bold.Render(flagSpec)
		}

		usage := strings.TrimSpace(f.Usage)
		if usage == "" {
			usage = ""
		}

		def := strings.TrimSpace(f.DefValue)
		var defText string
		if def != "" && def != "false" && def != "0" && def != "<nil>" {
			defText = fmt.Sprintf(" (default %q)", def)
		}

		entry := fmt.Sprintf("  %-22s  %s%s", displaySpec, usage, defText)
		entries = append(entries, entry)
	})

	sort.Strings(entries)
	return strings.Join(entries, "\n") + "\n"
}

func PrintHelpText(cmd *cobra.Command, args []string) {
	fmt.Println(FormatHelpText(cmd, args))
}
