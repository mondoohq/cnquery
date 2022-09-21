package components

import (
	"errors"
	"os"

	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

type passwordErrMsg error

type password struct {
	textInput  input.Model
	err        error
	onComplete func(res string, aborted bool)
}

// A bubbletea password input component
//
//	passwordModel := components.NewPasswordModel("root@192.168.178.141's password: ", func(res string, aborted bool) {
//		 // output the pwd
//		 fmt.Println(password)
//	})
//
// p := tea.NewProgram(passwordModel)
//
//	if err := p.Start(); err != nil {
//		 panic(err)
//	}
func NewPasswordModel(prompt string, onComplete func(res string, aborted bool)) password {
	inputModel := input.NewModel()
	inputModel.Prompt = prompt
	inputModel.Focus()
	inputModel.EchoMode = input.EchoNone
	inputModel.CharLimit = 156
	inputModel.Width = 20

	return password{
		textInput:  inputModel,
		err:        nil,
		onComplete: onComplete,
	}
}

func (m password) Init() tea.Cmd {
	return input.Blink
}

func (m password) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		aborted := false
		switch msg.Type {
		case tea.KeyCtrlC:
			aborted = true
			fallthrough
		case tea.KeyEsc:
			aborted = true
			fallthrough
		case tea.KeyEnter:
			m.onComplete(m.textInput.Value(), aborted)
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case passwordErrMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m password) View() string {
	return m.textInput.View() + "\n"
}

// AskPassword will only prompt the user for a password if they are on a TTY.
func AskPassword(prompt string) (string, error) {
	// check if password is set
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return "", errors.New("asking passwords is only supported when used with an interactive terminal (TTY)")
	}

	// ask user for password
	var res string = ""
	passwordModel := NewPasswordModel(prompt, func(userPassword string, aborted bool) {
		res = userPassword
		if aborted {
			os.Exit(1)
		}
	})

	p := tea.NewProgram(passwordModel)
	if err := p.Start(); err != nil {
		return res, err
	}

	return res, nil
}
