package async_operation

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
)

// Spinner is an interface for running a polling operation with visual feedback.
// The generic type T represents the result type returned by the polling function.
type Spinner[T any] interface {
	Run(context.Context) (*T, humane.Error)
}

type spinnerModel[T any] struct {
	err          error
	ready        bool
	showReadyMsg bool
	startedAt    time.Time
	result       T
}

type spinnerImpl[T any] struct {
	model   spinnerModel[T]
	spinner Spinner[T]
}

// PollFunc is a function type that performs a polling operation and returns
// a result of type T or an error.
type PollFunc[T any] func() (T, humane.Error)

// NewSpinner creates a new Spinner instance configured with the given polling function
// and options. It automatically selects between a terminal-based or text-based spinner
// depending on whether the output is a TTY.
func NewSpinner[T any](pollFunc PollFunc[T], opts ...PollModelOption) Spinner[T] {
	s := &spinnerImpl[T]{
		spinner: nil,
		model: spinnerModel[T]{
			err:          nil,
			ready:        false,
			showReadyMsg: false,
			startedAt:    time.Now(),
		},
	}

	options := spinnerOptions{}
	WithDefaultOptions()(&options)
	for _, opt := range opts {
		opt(&options)
	}

	// Force the style to Silent if the quiet option is set
	if options.quiet {
		options.style = Silent
	}

	if pretty_print.IsTerminal() && !options.quiet {
		s.spinner = newTeaSpinner(pollFunc, &options, &s.model)
	} else {
		s.spinner = newTextSpinner(pollFunc, &options, &s.model)
	}

	return s
}

func (s *spinnerImpl[T]) Run(ctx context.Context) (*T, humane.Error) {
	return s.spinner.Run(ctx)
}
