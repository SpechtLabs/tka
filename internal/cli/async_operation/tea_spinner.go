package async_operation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
)

var silentSpinner = spinner.Spinner{
	Frames: []string{""},
	FPS:    0, //nolint:gomnd
}

type pollTriggerMsg struct{}

type pollResultMsg[T any] struct {
	result      T
	err         error
	shouldRetry bool
}

type teaPollModel[T any] struct {
	ctx      context.Context
	cancel   context.CancelFunc
	s        spinner.Model
	opts     *spinnerOptions
	model    *spinnerModel[T]
	pollFunc PollFunc[T]

	// programOptions are added to the bubbletea program's options. Tests use
	// them to run the program without a terminal.
	programOptions []tea.ProgramOption
}

func newTeaSpinner[T any](pollFunc PollFunc[T], opts *spinnerOptions, model *spinnerModel[T]) *teaPollModel[T] {
	s := spinner.New()

	switch opts.style {
	case Dot:
		s.Spinner = spinner.Dot
	case Line:
		s.Spinner = spinner.Line
	case MiniDot:
		s.Spinner = spinner.MiniDot
	case Jump:
		s.Spinner = spinner.Jump
	case Pulse:
		s.Spinner = spinner.Pulse
	case Points:
		s.Spinner = spinner.Points
	case Globe:
		s.Spinner = spinner.Globe
	case Moon:
		s.Spinner = spinner.Moon
	case Monkey:
		s.Spinner = spinner.Monkey
	case Meter:
		s.Spinner = spinner.Meter
	case Hamburger:
		s.Spinner = spinner.Hamburger
	case Ellipsis:
		s.Spinner = spinner.Ellipsis
	case Silent:
		s.Spinner = silentSpinner
	}

	return &teaPollModel[T]{
		ctx:      nil,
		cancel:   nil,
		s:        s,
		opts:     opts,
		model:    model,
		pollFunc: pollFunc,

		programOptions: nil,
	}
}

func (m teaPollModel[T]) run(ctx context.Context) (*T, humane.Error) {
	m.ctx, m.cancel = context.WithCancel(ctx)
	defer m.cancel()
	m.model.startedAt = time.Now()

	// WithContext kills the program as soon as the context is done, rather
	// than when the poll in flight notices.
	prog := tea.NewProgram(m, append(m.programOptions, tea.WithContext(m.ctx))...)
	finalModel, err := prog.Run()

	if err != nil {
		if m.ctx.Err() != nil {
			return nil, m.stoppedError()
		}
		return nil, humane.Wrap(err, "UI error while polling", "try running with --quiet flag to disable the spinner")
	}

	return finalModel.(teaPollModel[T]).outcome()
}

// outcome is what a finished poll returns: the result, the error the poll
// failed with, or why it was stopped before it finished.
func (m teaPollModel[T]) outcome() (*T, humane.Error) {
	if m.model.err != nil {
		if herr, ok := errors.AsType[humane.Error](m.model.err); ok {
			return nil, herr
		}
		return nil, humane.Wrap(m.model.err, "async operation failed", "check the server logs for more details")
	}

	// The program quit before a poll finished, which only ctrl+c does.
	if !m.model.ready {
		return nil, m.stoppedError()
	}

	return &m.model.result, nil
}

// stoppedError is the error a poll stopped by its context ends with.
func (m teaPollModel[T]) stoppedError() humane.Error {
	if err := m.ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
		return humane.Wrap(err, m.opts.timeoutMessage, "try increasing the timeout or check the server status")
	}
	return humane.Wrap(context.Canceled, "operation canceled", "run the command again to retry")
}

// Init initializes the poll model and starts the spinner and polling command routines.
func (m teaPollModel[T]) Init() tea.Cmd {
	return tea.Batch(
		m.s.Tick,
		pollOnceCmd(m),
	)
}

// Update handles incoming messages, updates the model state, and returns the updated model and command for processing.
func (m teaPollModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.QuitMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancel()
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.s, cmd = m.s.Update(msg)
		return m, cmd

	case pollTriggerMsg:
		return m, pollOnceCmd(m)

	case pollResultMsg[T]:
		m.opts.attempt++
		if msg.err == nil {
			m.model.ready = true
			m.model.result = msg.result
			m.model.showReadyMsg = time.Since(m.model.startedAt) > m.opts.keepProgressAfter
			return m, tea.Quit
		}

		// Retry on retryStatus
		if msg.shouldRetry {
			m.model.ready = false
			m.opts.delay *= 2
			return m, retryAfter(m.ctx, m.opts.delay)
		}

		// Terminal error
		m.model.ready = true
		m.model.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

// View returns a string representation of the current poll model's state, formatted based on readiness, error, or progress.
func (m teaPollModel[T]) View() string {
	switch {
	case m.model.ready:
		if !m.model.showReadyMsg || m.opts.quiet {
			return ""
		}

		return pretty_print.FormatOk(m.opts.doneMessage)

	case m.model.err != nil:
		return pretty_print.FormatError(m.model.err)

	default:
		if m.opts.quiet {
			return ""
		}

		s := strings.TrimSpace(m.s.View())
		lvl := pretty_print.InfoLvl
		return pretty_print.FormatWithOptions(lvl, m.opts.inProgressMessage, []string{}, pretty_print.WithIcon(lvl, s))
	}
}

// pollOnceCmd executes a single polling attempt based on the pollModel configuration and returns a pollResultMsg.
func pollOnceCmd[T any](m teaPollModel[T]) tea.Cmd {
	return func() tea.Msg {
		if m.ctx.Err() != nil {
			return pollResultMsg[T]{err: m.stoppedError()}
		}

		resultCh := make(chan pollResultMsg[T], 1)

		// Run pollFunc in a separate goroutine
		go func() {
			result, err := m.pollFunc()
			shouldRetry := err != nil && m.opts.attempt < m.opts.maxAttempts

			// if we get a forbidden or unauthorized from the API, we can terminate
			// early, because there is going to be no recovery for that in any sort
			// or form at all.
			if err != nil && (strings.Contains(err.Display(), fmt.Sprintf("%d", http.StatusUnauthorized)) ||
				strings.Contains(err.Display(), fmt.Sprintf("%d", http.StatusForbidden))) {
				shouldRetry = false
			}

			resultCh <- pollResultMsg[T]{
				result:      result,
				err:         err,
				shouldRetry: shouldRetry,
			}
		}()

		// Wait for either context done or pollFunc to complete
		select {
		case <-m.ctx.Done():
			return pollResultMsg[T]{err: m.stoppedError()}
		case msg := <-resultCh:
			return msg
		}
	}
}

// retryAfter triggers the next poll after delay, or as soon as ctx is done,
// so that a stopped spinner doesn't sit out a backoff that doubles with
// every attempt.
func retryAfter(ctx context.Context, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		return pollTriggerMsg{}
	}
}
