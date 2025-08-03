package async_operation

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/internal/cli/pretty_print"
)

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

type PollFunc[T any] func() (T, humane.Error)

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

	if pretty_print.IsTerminal() {
		s.spinner = newTeaSpinner(pollFunc, &options, &s.model)
	} else {
		s.spinner = newTextSpinner(pollFunc, &options, &s.model)
	}

	return s
}

func (s *spinnerImpl[T]) Run(ctx context.Context) (*T, humane.Error) {
	return s.spinner.Run(ctx)
}
