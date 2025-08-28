package pretty_print

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type templateData struct {
	*cobra.Command
	ShowUsage bool
}

var Template = `
# Usage ` + "`{{ .Name }}`" + `
` + "```bash" + `
{{if .Runnable}}{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}{{.CommandPath}} [command]{{end}}
` + "```" + `

{{if and .ShowUsage (gt (len .Aliases) 0)}}
## Aliases
- ` + "`{{ .Name }}`, `{{ join .Aliases \"`, `\" }}` " + `
{{end}}

## Description
{{if .ShowUsage}}
{{if gt (len .Long) 0}}
{{.Long }}
{{else}}
{{.Short}}
{{end}}
{{else}}
{{.Short}}
{{end}}

{{if and .ShowUsage .HasExample}}
## Examples
` + "```bash" + `
{{.Example}}
` + "```" + `
{{end}}

{{if .HasAvailableSubCommands}}
{{$cmds := .Commands}}{{if eq (len .Groups) 0}}
## Available Commands

> [!TIP]
> Use ` + "`{{.CommandPath}} [command] --help`" + ` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|{{range $cmds}}{{if and .IsAvailableCommand (ne .Name "help")}}
| **` + "`{{.Name}}`" + `** | {{.Short}} |{{end}}{{end}}
{{else}}
{{range $group := .Groups}}
### {{.Title}}
| **Command** | **Description** |
|:------------|:----------------|{{range $cmds}}{{if (and (eq .GroupID $group.ID) and .IsAvailableCommand (ne .Name "help"))}}
| **{{.Name}}** | {{.Short}} |{{end}}{{end}}
{{end}}

{{if not .AllChildCommandsHaveGroup}}
### Additional Commands
{{range $cmds}}{{if (and (eq .GroupID "") and .IsAvailableCommand (ne .Name "help"))}}
- **{{.Name}}**: {{.Short}}
{{end}}{{end}}
{{end}}{{end}}
{{end}}

{{if and .ShowUsage }}
{{if .HasAvailableLocalFlags}}
{{$localFlags := .LocalFlags | FlagUsages}}
## Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|{{range $localFlags}}
| ` + "`{{.Flag}}`" + ` | {{.Type}} | {{.Usage}} |{{end}}
{{end}}

{{if .HasAvailableInheritedFlags}}
{{$inheritedFlags := .InheritedFlags | FlagUsages}}
## Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|{{range $inheritedFlags}}
| ` + "`{{.Flag}}`" + ` | {{.Type}} | {{.Usage}} |{{end}}
{{end}}
{{else}}
{{if or .HasAvailableLocalFlags .HasAvailableInheritedFlags}}
{{$localFlags := .LocalFlags | FlagUsages}}
{{$inheritedFlags := .InheritedFlags | FlagUsages}}
## Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|{{range $localFlags}}
| ` + "`{{.Flag}}`" + ` | {{.Type}} | {{.Usage}} |{{end}}{{range $inheritedFlags}}
| ` + "`{{.Flag}}`" + ` | {{.Type}} | {{.Usage}} |{{end}}
{{end}}
{{end}}

{{if and .ShowUsage .HasHelpSubCommands}}
## Additional Help Topics
{{range .Commands}}
{{if .IsAdditionalHelpTopicCommand}}
- **{{.CommandPath}}**: {{.Short}}{{end}}{{end}}
{{end}}`

var templateFuncs = template.FuncMap{
	"gt":         Gt,
	"eq":         Eq,
	"FlagUsages": FlagUsages,
	"join":       JoinString,
}

func FormatHelpText(cmd *cobra.Command, _ []string) string {
	return render(cmd, true)
}

func PrintHelpText(cmd *cobra.Command, args []string) {
	fmt.Println(render(cmd, false))
}

func PrintUsageText(cmd *cobra.Command, _ []string) {
	fmt.Println(render(cmd, true))
}

func render(cmd *cobra.Command, showUsage bool) string {
	options := DefaultOptions()

	// if the user wants long output, show the usage text
	if viper.GetBool("output.long") {
		showUsage = true
	}

	tmpl, err := template.New("top").Funcs(templateFuncs).Parse(Template)
	if err != nil {
		return cmd.UsageString()
	}

	var buf bytes.Buffer
	data := templateData{Command: cmd, ShowUsage: showUsage}
	if err := tmpl.Execute(&buf, data); err != nil {
		return cmd.UsageString()
	}

	mdString := buf.String()

	// Return raw markdown without rendering for MarkdownStyle
	if options.Theme == MarkdownStyle {
		return mdString
	}

	// For non-Markdown themes, optionally format alerts then render once
	if options.Theme != NoTTYStyle && options.Theme != AsciiStyle {
		mdString = formatMarkdownAlerts(mdString)
	}

	// Render markdown
	if out, err := options.MarkdownRenderer(options.Theme).Render(mdString); err == nil {
		return out
	}

	// If rendering fails for any reason, return the raw markdown
	return mdString
}

func formatMarkdownAlerts(mdString string) string {
	mdString = strings.ReplaceAll(mdString, "[!NOTE]", "__ⓘ Note__")
	mdString = strings.ReplaceAll(mdString, "[!TIP]", "__➤ Tip__")
	mdString = strings.ReplaceAll(mdString, "[!IMPORTANT]", "__‼ Important__")
	mdString = strings.ReplaceAll(mdString, "[!WARNING]", "__⚠ Warning__")
	mdString = strings.ReplaceAll(mdString, "[!CRITICAL]", "__☠ Critical__")
	return mdString
}

// Gt takes two types and checks whether the first type is greater than the second. In case of types Arrays, Chans,
// Maps and Slices, Gt will compare their lengths. Ints are compared directly while strings are first parsed as
// ints and then compared.
func Gt(a interface{}, b interface{}) bool {
	var left, right int64
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = int64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = av.Int()
	case reflect.String:
		left, _ = strconv.ParseInt(av.String(), 10, 64)
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = int64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = bv.Int()
	case reflect.String:
		right, _ = strconv.ParseInt(bv.String(), 10, 64)
	}

	return left > right
}

// Eq takes two types and checks whether they are equal. Supported types are int and string. Unsupported types will panic.
func Eq(a interface{}, b interface{}) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		panic("Eq called on unsupported type")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() == bv.Int()
	case reflect.String:
		return av.String() == bv.String()
	}
	return false
}

func JoinString(a []string, sep string) string {
	return strings.Join(a, sep)
}

type FlagUsage struct {
	Flag  string
	Type  string
	Usage string
}

// FlagUsages returns a list of flag usages for a flag set.
func FlagUsages(f *pflag.FlagSet) []FlagUsage {
	lines := make([]FlagUsage, 0)

	f.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}

		flagStr := ""
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			flagStr = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
		} else {
			flagStr = fmt.Sprintf("    --%s", flag.Name)
		}

		varname, usage := pflag.UnquoteUsage(flag)
		if varname != "" && varname != flag.Value.Type() {
			flagStr = fmt.Sprintf("%s [%s]", flagStr, varname)
		}
		if flag.NoOptDefVal != "" {
			switch flag.Value.Type() {
			case "string":
				flagStr += fmt.Sprintf("[=\"%s\"]", flag.NoOptDefVal)
			case "bool":
				if flag.NoOptDefVal != "true" {
					flagStr += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			case "count":
				if flag.NoOptDefVal != "+1" {
					flagStr += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
				}
			default:
				flagStr += fmt.Sprintf("[=%s]", flag.NoOptDefVal)
			}
		}

		if !defaultIsZeroValue(flag) {
			if flag.Value.Type() == "string" {
				usage += fmt.Sprintf(" (_default: %q_)", flag.DefValue)
			} else {
				usage += fmt.Sprintf(" (_default: %s_)", flag.DefValue)
			}
		}
		if len(flag.Deprecated) != 0 {
			usage = fmt.Sprintf("(__DEPRECATED__: %s) %s", flag.Deprecated, usage)
		}

		lines = append(lines, FlagUsage{
			Flag:  flagStr,
			Type:  fmt.Sprintf("`%s`", flag.Value.Type()),
			Usage: usage,
		})
	})

	return lines
}

// defaultIsZeroValue returns true if the default value for this flag represents
// a zero value.
func defaultIsZeroValue(f *pflag.Flag) bool {
	switch f.Value.Type() {
	case "bool":
		return f.DefValue == "false"
	case "duration":
		return f.DefValue == "0" || f.DefValue == "0s"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "count", "float32", "float64":
		return f.DefValue == "0"
	case "string":
		return f.DefValue == ""
	case "ip", "ipMask", "ipNet":
		return f.DefValue == "<nil>"
	case "intSlice", "stringSlice", "stringArray":
		return f.DefValue == "[]"
	default:
		switch f.Value.String() {
		case "false":
			return true
		case "<nil>":
			return true
		case "":
			return true
		case "0":
			return true
		}
		return false
	}
}
