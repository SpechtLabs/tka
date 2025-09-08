package async_operation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sierrasoftworks/humane-errors-go"
)

type textPollModel[T any] struct {
	tea *teaPollModel[T]
}

func newTextSpinner[T any](pollFunc PollFunc[T], opts *spinnerOptions, model *spinnerModel[T]) *textPollModel[T] {
	WithSpinnerStyle(Line)(opts)

	return &textPollModel[T]{
		tea: newTeaSpinner(pollFunc, opts, model),
	}
}

func (m textPollModel[T]) Run(ctx context.Context) (*T, humane.Error) {
	m.tea.ctx = ctx
	m.tea.model.startedAt = time.Now()

	var finalModel teaPollModel[T]

	var msg tea.Msg = pollTriggerMsg{}

	for {
		var cmd tea.Cmd
		m.tea.s, cmd = m.tea.s.Update(m.tea.s.Tick())
		cmd()

		model, cmd := m.tea.Update(msg)
		txt := model.View()
		txt = strings.TrimSuffix(txt, "\n")
		if !m.tea.opts.quiet {
			fmt.Printf("\r%s", txt)
		}

		msg = cmd()
		if msg == tea.Quit() {
			finalModel = model.(teaPollModel[T])
			txt = finalModel.View()
			if !m.tea.opts.quiet {
				fmt.Printf("\r%s", txt)
			}
			break
		}
	}

	if finalModel.model.err != nil {
		var herr humane.Error
		if errors.As(finalModel.model.err, &herr) {
			return nil, herr
		} else {
			return nil, humane.Wrap(finalModel.model.err, "async operation failed")
		}
	}

	return &finalModel.model.result, nil
}
