// Package async_operation provides utilities for displaying progress spinners
// during long-running asynchronous operations in the CLI.
package async_operation

import (
	"net/http"
	"time"
)

// SpinnerStyle represents the visual style of the progress spinner animation.
type SpinnerStyle int

// Spinner style constants define the available animation styles for the progress spinner.
const (
	// Line displays a rotating line animation.
	Line SpinnerStyle = iota
	// Dot displays a bouncing dot animation.
	Dot
	// MiniDot displays a smaller dot animation.
	MiniDot
	// Jump displays a jumping animation.
	Jump
	// Pulse displays a pulsing animation.
	Pulse
	// Points displays expanding dots animation.
	Points
	// Globe displays a spinning globe animation.
	Globe
	// Moon displays moon phases animation.
	Moon
	// Monkey displays a monkey emoji animation.
	Monkey
	// Meter displays a progress meter animation.
	Meter
	// Hamburger displays a hamburger menu animation.
	Hamburger
	// Ellipsis displays an ellipsis animation.
	Ellipsis
	// Silent disables the visual spinner animation.
	Silent
)

type spinnerOptions struct {
	attempt           int
	maxAttempts       int
	inProgressMessage string
	doneMessage       string
	failedMessage     string
	timeoutMessage    string
	keepProgressAfter time.Duration
	delay             time.Duration
	expectedCode      int
	style             SpinnerStyle
	quiet             bool
}

// PollModelOption is a functional option for configuring spinner behavior.
type PollModelOption func(*spinnerOptions)

// WithInProgressMessage sets the message displayed while the polling operation is in progress.
func WithInProgressMessage(msg string) PollModelOption {
	return func(s *spinnerOptions) {
		s.inProgressMessage = msg
	}
}

// WithDoneMessage sets the message displayed when the polling operation completes successfully.
func WithDoneMessage(msg string) PollModelOption {
	return func(s *spinnerOptions) {
		s.doneMessage = msg
	}
}

// WithFailedMessage sets the message displayed when the polling operation fails.
func WithFailedMessage(msg string) PollModelOption {
	return func(s *spinnerOptions) {
		s.failedMessage = msg
	}
}

// WithMaxAttempts sets the maximum number of attempts the spinnerOptions can perform.
func WithMaxAttempts(attempts int) PollModelOption {
	return func(s *spinnerOptions) {
		s.maxAttempts = attempts
	}
}

// WithDelay sets the delay duration between polling attempts for the spinnerOptions.
func WithDelay(delay time.Duration) PollModelOption {
	return func(s *spinnerOptions) {
		s.delay = delay
	}
}

// WithExpectedCode sets the expected HTTP status code to determine whether the polling operation was successful.
func WithExpectedCode(code int) PollModelOption {
	return func(s *spinnerOptions) {
		s.expectedCode = code
	}
}

// WithKeepProgressAfter sets the duration for keeping the progress message displayed after the polling process ends.
func WithKeepProgressAfter(duration time.Duration) PollModelOption {
	return func(s *spinnerOptions) {
		s.keepProgressAfter = duration
	}
}

// WithSpinnerStyle sets the spinner style for the spinner options using the provided SpinnerStyle.
func WithSpinnerStyle(style SpinnerStyle) PollModelOption {
	return func(s *spinnerOptions) {
		s.style = style
	}
}

// WithDefaultOptions returns a PollModelOption that sets sensible default values
// for all spinner options including max attempts, messages, delay, and style.
func WithDefaultOptions() PollModelOption {
	return func(s *spinnerOptions) {
		s.attempt = 0
		s.maxAttempts = 10
		s.inProgressMessage = "Waiting..."
		s.doneMessage = "Done!"
		s.failedMessage = "Failed!"
		s.timeoutMessage = "Operation timed out!"
		s.keepProgressAfter = 500 * time.Millisecond
		s.style = Dot
		s.delay = 100 * time.Millisecond
		s.expectedCode = http.StatusOK
	}
}

// WithQuiet sets the spinner to quiet mode, suppressing visual output
// and using the Silent spinner style.
func WithQuiet(quiet bool) PollModelOption {
	return func(s *spinnerOptions) {
		s.quiet = quiet
		s.style = Silent
	}
}
