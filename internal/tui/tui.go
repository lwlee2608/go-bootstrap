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
	inputGenReadme
	inputReadmeDesc
	done
)

type Result struct {
	AppName     string
	ModuleName  string
	AddHTTP     bool
	FullStack   bool
	GenReadme   bool
	Description string
}

type Model struct {
	state     state
	appInput  textinput.Model
	modInput  textinput.Model
	descInput textinput.Model
	addHTTP   bool
	fullStack bool
	genReadme bool
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

	descInput := textinput.New()
	descInput.Width = 60
	descInput.Placeholder = "what does this app do?"

	return Model{
		state:     inputAppName,
		appInput:  appInput,
		modInput:  modInput,
		descInput: descInput,
		addHTTP:   true,
		fullStack: true,
		genReadme: true,
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
				m.state = inputGenReadme
				return m, nil

			case inputGenReadme:
				if m.genReadme {
					m.state = inputReadmeDesc
					m.descInput.Focus()
					return m, textinput.Blink
				}
				return m.finish()

			case inputReadmeDesc:
				return m.finish()
			}

		case tea.KeySpace, tea.KeyLeft, tea.KeyRight:
			switch m.state {
			case inputAddHTTP:
				m.addHTTP = !m.addHTTP
				return m, nil
			case inputFullStack:
				m.fullStack = !m.fullStack
				return m, nil
			case inputGenReadme:
				m.genReadme = !m.genReadme
				return m, nil
			}

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
			case inputGenReadme:
				switch string(msg.Runes) {
				case "y", "Y":
					m.genReadme = true
				case "n", "N":
					m.genReadme = false
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
	case inputReadmeDesc:
		m.descInput, cmd = m.descInput.Update(msg)
	}
	return m, cmd
}

func (m Model) finish() (tea.Model, tea.Cmd) {
	m.result = Result{
		AppName:     m.appInput.Value(),
		ModuleName:  m.modInput.Value(),
		AddHTTP:     m.addHTTP,
		FullStack:   m.fullStack,
		GenReadme:   m.genReadme,
		Description: m.descInput.Value(),
	}
	m.state = done
	return m, tea.Quit
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
	case inputReadmeDesc:
		return "enter generate · esc quit"
	case inputGenReadme:
		if !m.genReadme {
			return "←/→ y/n select · enter generate · esc quit"
		}
		return "←/→ y/n select · enter continue · esc quit"
	default:
		return "←/→ y/n select · enter continue · esc quit"
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
		b.WriteString(stepDone("Full stack", yesNo(m.result.FullStack)) + "\n")
		b.WriteString(stepDone("AI README", yesNo(m.result.GenReadme)) + "\n\n")
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

	switch {
	case m.state == inputFullStack:
		b.WriteString(stepCurrent("Full stack?   "+yesNoIndicator(m.fullStack)) + "\n")
		app := m.appInput.Value()
		b.WriteString(unselectedStyle.Render(fmt.Sprintf(
			"    nests backend under services/%s-server, adds services/%s-web frontend", app, app)) + "\n")
	case m.state > inputFullStack:
		b.WriteString(stepDone("Full stack", yesNo(m.fullStack)) + "\n")
	default:
		b.WriteString(stepPending("Full stack") + "\n")
	}

	switch {
	case m.state == inputGenReadme:
		b.WriteString(stepCurrent("Generate AI README?   "+yesNoIndicator(m.genReadme)) + "\n")
	case m.state > inputGenReadme:
		b.WriteString(stepDone("AI README", yesNo(m.genReadme)) + "\n")
	default:
		b.WriteString(stepPending("AI README") + "\n")
	}

	if m.state == inputReadmeDesc {
		b.WriteString(stepCurrent("Short description") + "\n")
		b.WriteString("    " + m.descInput.View() + "\n")
	} else if m.state < inputGenReadme || m.genReadme {
		b.WriteString(stepPending("Description") + "\n")
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
