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
	inputAddHTTP
	done
)

type Result struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
}

type Model struct {
	state    state
	appInput textinput.Model
	modInput textinput.Model
	addHTTP  bool
	result   Result
	err      error
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
				m.state = inputAddHTTP
				m.modInput.Blur()
				return m, nil

			case inputAddHTTP:
				m.result = Result{
					AppName:    m.appInput.Value(),
					ModuleName: m.modInput.Value(),
					AddHTTP:    m.addHTTP,
				}
				m.state = done
				return m, tea.Quit
			}

		case tea.KeyRunes:
			if m.state == inputAddHTTP {
				switch string(msg.Runes) {
				case "y", "Y":
					m.addHTTP = true
				case "n", "N":
					m.addHTTP = false
				}
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
		return fmt.Sprintf(`App name:
%s

(Tab to complete, Enter to continue, Esc to quit)`,
			m.appInput.View(),
		)

	case inputModuleName:
		return fmt.Sprintf(`App name: %s

Go module name:
%s

(Tab to complete, Enter to continue, Esc to quit)`,
			m.appInput.Value(),
			m.modInput.View(),
		)

	case inputAddHTTP:
		yesNo := "y/N"
		if m.addHTTP {
			yesNo = "Y/n"
		}
		return fmt.Sprintf(`App name: %s
Module: %s

Add http scaffolding? [%s]

(y/n to toggle, Enter to generate, Esc to quit)`,
			m.appInput.Value(),
			m.modInput.Value(),
			yesNo,
		)

	case done:
		return fmt.Sprintf(`Generating project...
  App: %s
  Module: %s
  HTTP Endpoint: %v
`,
			m.result.AppName,
			m.result.ModuleName,
			m.result.AddHTTP,
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
