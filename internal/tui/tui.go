package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	inputAppName state = iota
	inputModuleName
	inputAddHTTP
	inputFullStack
	done
)

type Result struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
	FullStack  bool
}

type Model struct {
	state     state
	appInput  textinput.Model
	modInput  textinput.Model
	addHTTP   bool
	fullStack bool
	result    Result
	err       error
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
		state:     inputAppName,
		appInput:  appInput,
		modInput:  modInput,
		addHTTP:   true,
		fullStack: true,
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
				m.state = inputFullStack
				return m, nil

			case inputFullStack:
				m.result = Result{
					AppName:    m.appInput.Value(),
					ModuleName: m.modInput.Value(),
					AddHTTP:    m.addHTTP,
					FullStack:  m.fullStack,
				}
				m.state = done
				return m, tea.Quit
			}

		case tea.KeySpace, tea.KeyLeft, tea.KeyRight:
			switch m.state {
			case inputAddHTTP:
				m.addHTTP = !m.addHTTP
			case inputFullStack:
				m.fullStack = !m.fullStack
			}
			return m, nil

		case tea.KeyRunes:
			switch m.state {
			case inputAddHTTP:
				switch string(msg.Runes) {
				case "y", "Y":
					m.addHTTP = true
				case "n", "N":
					m.addHTTP = false
				}
				return m, nil
			case inputFullStack:
				switch string(msg.Runes) {
				case "y", "Y":
					m.fullStack = true
				case "n", "N":
					m.fullStack = false
				}
				return m, nil
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

var (
	titleStyle      = lipgloss.NewStyle().Bold(true)
	checkStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	cursorStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selectedStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	unselectedStyle = lipgloss.NewStyle().Faint(true)
)

func yesNoIndicator(v bool) string {
	if v {
		return selectedStyle.Render("[ Yes ]") + unselectedStyle.Render("   No  ")
	}
	return unselectedStyle.Render("  Yes  ") + selectedStyle.Render(" [ No ]")
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func stepDone(label, value string) string {
	return fmt.Sprintf("  %s %-11s %s", checkStyle.Render("✓"), label, value)
}

func stepCurrent(label string) string {
	return "  " + cursorStyle.Render("▸") + " " + label
}

func stepPending(label string) string {
	return unselectedStyle.Render("    " + label)
}

func (m Model) help() string {
	switch m.state {
	case inputAppName, inputModuleName:
		return "tab complete · enter continue · esc quit"
	case inputAddHTTP:
		return "←/→ y/n select · enter continue · esc quit"
	default:
		return "←/→ y/n select · enter generate · esc quit"
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Genesis — new Go project"))
	b.WriteString("\n\n")

	if m.state == done {
		b.WriteString(stepDone("App name", m.result.AppName) + "\n")
		b.WriteString(stepDone("Module", m.result.ModuleName) + "\n")
		b.WriteString(stepDone("HTTP", yesNo(m.result.AddHTTP)) + "\n")
		b.WriteString(stepDone("Full stack", yesNo(m.result.FullStack)) + "\n\n")
		b.WriteString("  Generating project...\n")
		return b.String()
	}

	if m.state == inputAppName {
		b.WriteString(stepCurrent("App name") + "\n")
		b.WriteString("    " + m.appInput.View() + "\n")
	} else {
		b.WriteString(stepDone("App name", m.appInput.Value()) + "\n")
	}

	switch {
	case m.state == inputModuleName:
		b.WriteString(stepCurrent("Module") + "\n")
		b.WriteString("    " + m.modInput.View() + "\n")
	case m.state > inputModuleName:
		b.WriteString(stepDone("Module", m.modInput.Value()) + "\n")
	default:
		b.WriteString(stepPending("Module") + "\n")
	}

	switch {
	case m.state == inputAddHTTP:
		b.WriteString(stepCurrent("Add HTTP scaffolding?   "+yesNoIndicator(m.addHTTP)) + "\n")
	case m.state > inputAddHTTP:
		b.WriteString(stepDone("HTTP", yesNo(m.addHTTP)) + "\n")
	default:
		b.WriteString(stepPending("HTTP scaffolding") + "\n")
	}

	if m.state == inputFullStack {
		b.WriteString(stepCurrent("Full stack?   "+yesNoIndicator(m.fullStack)) + "\n")
		app := m.appInput.Value()
		b.WriteString(unselectedStyle.Render(fmt.Sprintf(
			"    nests backend under services/%s-server, adds services/%s-web frontend", app, app)) + "\n")
	} else {
		b.WriteString(stepPending("Full stack") + "\n")
	}

	b.WriteString("\n" + unselectedStyle.Render("  "+m.help()) + "\n")
	return b.String()
}

func (m Model) Result() (Result, error) {
	if m.err != nil {
		return Result{}, m.err
	}
	return m.result, nil
}
