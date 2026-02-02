package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	inputAppName state = iota
	inputModuleName
	done
)

type Result struct {
	AppName    string
	ModuleName string
}

type Model struct {
	state      state
	appInput   textinput.Model
	modInput   textinput.Model
	result     Result
	err        error
}

type Options struct {
	SuggestedAppName    string
	SuggestedModuleName string
}

func New(opts Options) Model {
	appInput := textinput.New()
	appInput.Width = 40
	if opts.SuggestedAppName != "" {
		appInput.Placeholder = opts.SuggestedAppName
	} else {
		appInput.Placeholder = "myapp"
	}
	appInput.Focus()

	modInput := textinput.New()
	modInput.Width = 60
	if opts.SuggestedModuleName != "" {
		modInput.Placeholder = opts.SuggestedModuleName
	} else {
		modInput.Placeholder = "github.com/user/myapp"
	}

	return Model{
		state:    inputAppName,
		appInput: appInput,
		modInput: modInput,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit

		case tea.KeyTab:
			switch m.state {
			case inputAppName:
				if m.appInput.Value() == "" && m.appInput.Placeholder != "" {
					m.appInput.SetValue(m.appInput.Placeholder)
				}
				return m, nil

			case inputModuleName:
				if m.modInput.Value() == "" && m.modInput.Placeholder != "" {
					m.modInput.SetValue(m.modInput.Placeholder)
				}
				return m, nil
			}

		case tea.KeyEnter:
			switch m.state {
			case inputAppName:
				if m.appInput.Value() == "" {
					return m, nil
				}
				m.state = inputModuleName
				m.appInput.Blur()
				m.modInput.Focus()
				return m, textinput.Blink

			case inputModuleName:
				if m.modInput.Value() == "" {
					return m, nil
				}
				m.result = Result{
					AppName:    m.appInput.Value(),
					ModuleName: m.modInput.Value(),
				}
				m.state = done
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	switch m.state {
	case inputAppName:
		m.appInput, cmd = m.appInput.Update(msg)
	case inputModuleName:
		m.modInput, cmd = m.modInput.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.state {
	case inputAppName:
		return fmt.Sprintf(
			"App name:\n%s\n\n(Tab to complete, Enter to continue, Esc to quit)",
			m.appInput.View(),
		)

	case inputModuleName:
		return fmt.Sprintf(
			"App name: %s\n\nGo module name:\n%s\n\n(Tab to complete, Enter to generate, Esc to quit)",
			m.appInput.Value(),
			m.modInput.View(),
		)

	case done:
		return fmt.Sprintf(
			"Generating project...\n  App: %s\n  Module: %s\n",
			m.result.AppName,
			m.result.ModuleName,
		)
	}
	return ""
}

func (m Model) Result() (Result, error) {
	if m.err != nil {
		return Result{}, m.err
	}
	return m.result, nil
}
