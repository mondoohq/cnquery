// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"errors"
	"os"

	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

type inputErrMsg error

type inputModel struct {
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
func NewInputModel(prompt string, onComplete func(res string, aborted bool), password bool) inputModel {
	im := input.New()
	im.Prompt = prompt
	im.Focus()
	im.EchoMode = input.EchoNormal
	if password {
		im.EchoMode = input.EchoPassword
	}
	im.CharLimit = 156
	im.Width = 20

	return inputModel{
		textInput:  im,
		err:        nil,
		onComplete: onComplete,
	}
}

func (m inputModel) Init() tea.Cmd {
	return input.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case inputErrMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	return m.textInput.View() + "\n"
}

// AskInput will only prompt the user for input if they are on a TTY.
func askInput(prompt string, password bool) (string, error) {
	// check if input is set
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		return "", errors.New("asking input is only supported when used with an interactive terminal (TTY)")
	}

	// ask user for input
	var res string = ""
	inputModel := NewInputModel(prompt, func(userInput string, aborted bool) {
		res = userInput
		if aborted {
			os.Exit(1)
		}
	}, password)

	p := tea.NewProgram(inputModel, tea.WithInputTTY())
	if _, err := p.Run(); err != nil {
		return res, err
	}

	return res, nil
}

// AskInput will only prompt the user for input if they are on a TTY.
func AskInput(prompt string) (string, error) {
	return askInput(prompt, false)
}

// AskPassword will only prompt the user for input if they are on a TTY.
func AskPassword(prompt string) (string, error) {
	return askInput(prompt, true)
}
