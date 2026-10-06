package async_operation

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/stretchr/testify/require"
)

// stopTimeout bounds how long a stopped spinner may take to return. The
// tests back off for an hour between polls, so a spinner that waits out the
// backoff fails them.
const stopTimeout = 5 * time.Second

// backoffSettle is how long the spinner gets to go from a failed poll into its
// backoff. The text spinner draws a frame (100ms) before it starts waiting.
const backoffSettle = 300 * time.Millisecond

// newTestSpinner builds a spinner that never needs a terminal: the text
// spinner, or the bubbletea one without input and with its output
// discarded.
func newTestSpinner[T any](t *testing.T, terminal bool, pollFunc PollFunc[T], opts ...PollModelOption) *spinnerImpl[T] {
	t.Helper()

	options := spinnerOptions{}
	WithDefaultOptions()(&options)
	WithQuiet(true)(&options)
	for _, opt := range opts {
		opt(&options)
	}

	s := &spinnerImpl[T]{}
	if terminal {
		m := newTeaSpinner(pollFunc, &options, &s.model)
		m.programOptions = []tea.ProgramOption{tea.WithInput(nil), tea.WithOutput(io.Discard)}
		s.spinner = m
	} else {
		s.spinner = newTextSpinner(pollFunc, &options, &s.model)
	}
	return s
}

// neverReady is a poll that always fails with a retryable error, and
// signals polled on every attempt.
func neverReady(polled chan<- struct{}) PollFunc[string] {
	return func() (string, humane.Error) {
		select {
		case polled <- struct{}{}:
		default:
		}
		return "", humane.New("not ready yet")
	}
}

// runAsync runs s in a goroutine and returns a channel that receives its
// error.
func runAsync(ctx context.Context, s Spinner[string]) <-chan humane.Error {
	done := make(chan humane.Error, 1)
	go func() {
		_, err := s.Run(ctx)
		done <- err
	}()
	return done
}

func waitForRun(t *testing.T, done <-chan humane.Error) humane.Error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(stopTimeout):
		require.FailNow(t, "the spinner kept running after it was stopped")
		return nil
	}
}

func TestSpinnerStopsWhenContextCanceled(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(map[bool]string{false: "text spinner", true: "terminal spinner"}[terminal], func(t *testing.T) {
			polled := make(chan struct{}, 1)
			s := newTestSpinner(t, terminal, neverReady(polled),
				WithMaxAttempts(100),
				WithDelay(time.Hour),
			)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			done := runAsync(ctx, s)
			<-polled
			// Give the spinner time to take the failed poll in and start its
			// hour-long backoff, so that the cancellation has to cut that
			// short.
			time.Sleep(backoffSettle)
			cancel()

			err := waitForRun(t, done)
			require.ErrorIs(t, err, context.Canceled)
			require.ErrorContains(t, err, "operation canceled")
		})
	}
}

func TestSpinnerTimesOut(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(map[bool]string{false: "text spinner", true: "terminal spinner"}[terminal], func(t *testing.T) {
			s := newTestSpinner(t, terminal, neverReady(make(chan struct{}, 1)),
				WithMaxAttempts(100),
				WithDelay(time.Hour),
			)

			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()

			err := waitForRun(t, runAsync(ctx, s))
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.ErrorContains(t, err, "Operation timed out!")
		})
	}
}

func TestSpinnerResult(t *testing.T) {
	unauthorized := humane.New("request failed with status 401")
	notReady := humane.New("not ready yet")

	tests := []struct {
		name string
		// poll is called with the number of the attempt, starting at 1.
		poll         func(attempt int32) (string, humane.Error)
		opts         []PollModelOption
		want         string
		wantErr      error
		wantAttempts int32
	}{
		{
			name:         "ready on the first poll",
			poll:         func(int32) (string, humane.Error) { return "kubeconfig", nil },
			want:         "kubeconfig",
			wantAttempts: 1,
		},
		{
			name: "ready after retries",
			poll: func(attempt int32) (string, humane.Error) {
				if attempt < 3 {
					return "", notReady
				}
				return "kubeconfig", nil
			},
			want:         "kubeconfig",
			wantAttempts: 3,
		},
		{
			name:         "unauthorized ends the poll without retrying",
			poll:         func(int32) (string, humane.Error) { return "", unauthorized },
			wantErr:      unauthorized,
			wantAttempts: 1,
		},
		{
			name:         "out of attempts",
			poll:         func(int32) (string, humane.Error) { return "", notReady },
			opts:         []PollModelOption{WithMaxAttempts(2)},
			wantErr:      notReady,
			wantAttempts: 3,
		},
	}

	for _, tt := range tests {
		for _, terminal := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, terminal=%t", tt.name, terminal), func(t *testing.T) {
				var attempts atomic.Int32
				poll := func() (string, humane.Error) { return tt.poll(attempts.Add(1)) }
				s := newTestSpinner(t, terminal, poll, append([]PollModelOption{WithDelay(time.Millisecond)}, tt.opts...)...)

				got, err := s.Run(t.Context())
				require.Equal(t, tt.wantAttempts, attempts.Load())
				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
					require.Nil(t, got)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.want, *got)
			})
		}
	}
}

func TestCtrlCCancelsThePoll(t *testing.T) {
	options := spinnerOptions{}
	WithDefaultOptions()(&options)
	m := newTeaSpinner(neverReady(make(chan struct{}, 1)), &options, &spinnerModel[string]{})
	m.ctx, m.cancel = context.WithCancel(t.Context())

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	require.ErrorIs(t, m.ctx.Err(), context.Canceled)
	require.Equal(t, tea.Quit(), cmd())

	got, err := model.(teaPollModel[string]).outcome()
	require.Nil(t, got)
	require.ErrorIs(t, err, context.Canceled)
}
