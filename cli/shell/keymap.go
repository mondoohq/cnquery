// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the shell
type KeyMap struct {
	// Basic navigation
	Submit key.Binding
	Exit   key.Binding
	Cancel key.Binding
	Clear  key.Binding

	// History navigation
	HistoryUp   key.Binding
	HistoryDown key.Binding

	// Completion navigation
	NextCompletion   key.Binding
	PrevCompletion   key.Binding
	AcceptCompletion key.Binding
	DismissPopup     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute query"),
		),
		Exit: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "exit shell"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "cancel/exit"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear screen"),
		),
		HistoryUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "previous history"),
		),
		HistoryDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "next history"),
		),
		NextCompletion: key.NewBinding(
			key.WithKeys("down", "tab"),
			key.WithHelp("tab/down", "next completion"),
		),
		PrevCompletion: key.NewBinding(
			key.WithKeys("up", "shift+tab"),
			key.WithHelp("shift+tab/up", "previous completion"),
		),
		AcceptCompletion: key.NewBinding(
			key.WithKeys("enter", "tab"),
			key.WithHelp("enter/tab", "accept completion"),
		),
		DismissPopup: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "dismiss completions"),
		),
	}
}

// ShortHelp returns keybindings to show in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Exit, k.Clear}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.Exit, k.Cancel, k.Clear},
		{k.HistoryUp, k.HistoryDown},
		{k.NextCompletion, k.PrevCompletion, k.AcceptCompletion, k.DismissPopup},
	}
}
