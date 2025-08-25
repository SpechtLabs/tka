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

	}

	return &teaPollModel[T]{
		ctx:      nil,
		cancel:   nil,
		s:        s,
		opts:     opts,
		model:    model,
		pollFunc: pollFunc,
	}
}

func (m teaPollModel[T]) Run(ctx context.Context) (*T, humane.Error) {
	m.ctx = ctx
	m.model.startedAt = time.Now()

	prog := tea.NewProgram(m)
	finalModel, err := prog.Run()

	if err != nil {
		return nil, humane.Wrap(err, "UI error while polling")
	}

	final := finalModel.(teaPollModel[T])
	if final.model.err != nil {
		var herr humane.Error
		if errors.As(final.model.err, &herr) {
			return nil, herr
		} else {
			return nil, humane.Wrap(final.model.err, "async operation failed")
		}
	}

	return &final.model.result, nil
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
			return m, tea.Tick(m.opts.delay, func(t time.Time) tea.Msg {
				return pollTriggerMsg{}
			})
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
		if m.model.showReadyMsg {
			return pretty_print.FormatOk(m.opts.doneMessage)
		}
		return ""

	case m.model.err != nil:
		return pretty_print.FormatError(m.model.err)

	default:
		s := strings.TrimSpace(m.s.View())
		lvl := pretty_print.InfoLvl
		return pretty_print.FormatWithOptions(lvl, m.opts.inProgressMessage, []string{}, pretty_print.WithIcon(lvl, s))
	}
}

// pollOnceCmd executes a single polling attempt based on the pollModel configuration and returns a pollResultMsg.
func pollOnceCmd[T any](m teaPollModel[T]) tea.Cmd {
	return func() tea.Msg {
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
			return pollResultMsg[T]{err: humane.New(m.opts.timeoutMessage)}
		case msg := <-resultCh:
			return msg
		}
	}
}
