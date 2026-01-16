package pretty_print

import (
	"os"
	"regexp"
	"testing"

	humane "github.com/sierrasoftworks/humane-errors-go"
)

func setEnvForNoTTY(t *testing.T) {
	t.Helper()
	oldTerm := os.Getenv("TERM")
	oldNoColor, hadNoColor := os.LookupEnv("NO_COLOR")
	_ = os.Setenv("TERM", "dumb")
	_ = os.Setenv("NO_COLOR", "1")
	// restore
	t.Cleanup(func() {
		_ = os.Setenv("TERM", oldTerm)
		if hadNoColor {
			_ = os.Setenv("NO_COLOR", oldNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	})
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSI(t *testing.T, s string) string {
	t.Helper()
	return ansiRegex.ReplaceAllString(s, "")
}

func contains(t *testing.T, haystack, needle string) bool {
	t.Helper()
	return len(haystack) >= len(needle) && (len(needle) == 0 || indexOf(t, haystack, needle) >= 0)
}

func indexOf(t *testing.T, haystack, needle string) int {
	t.Helper()
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestFormatWithOptions(t *testing.T) {
	setEnvForNoTTY(t)

	tests := []struct {
		name    string
		lvl     PrintLevel
		msg     string
		context []string
		opts    []Option
		assert  func(t *testing.T, got string)
	}{
		{
			name:    "info_no_color_context",
			lvl:     InfoLvl,
			msg:     "Hello",
			context: []string{"ctx1", "ctx2"},
			opts:    []Option{WithNoColor(true)},
			assert: func(t *testing.T, got string) {
				want := "ℹ Hello\n    ctx1\n    ctx2\n"
				if got != want {
					if stripANSI(t, got) == want {
						t.Fatalf("unexpected ANSI in output: %q", got)
					}
					t.Fatalf("unexpected output.\nwant: %q\n got: %q", want, got)
				}
			},
		},
		{
			name:    "error_humane_non_tty",
			lvl:     ErrLvl,
			msg:     "",
			context: nil,
			opts:    []Option{WithError(humane.New("boom", "this is a test error"))},
			assert: func(t *testing.T, got string) {
				plain := stripANSI(t, got)
				if plain == "" {
					t.Fatalf("expected error message, got empty string")
				}
				if want := "✗ boom"; !contains(t, plain, want) {
					t.Fatalf("expected output to contain %q, got %q", want, plain)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatWithOptions(tc.lvl, tc.msg, tc.context, tc.opts...)
			tc.assert(t, got)
		})
	}
}
