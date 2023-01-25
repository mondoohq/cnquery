package progress

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// This way we get the output without having a tty.
func newMultiProgressMockProgram(elements map[string]string, orderedKeys []string, input io.Reader, output io.Writer) (*tea.Program, error) {
	if len(elements) != len(orderedKeys) {
		return nil, fmt.Errorf("number of elements and orderedKeys must be equal")
	}
	m := newMultiProgress(elements)
	m.orderedKeys = orderedKeys
	m.maxItemsToShow = defaultProgressNumAssets
	return tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output)), nil
}

func newMultiProgressBarsMock(elements map[string]string, orderedKeys []string, input io.Reader, output io.Writer) (*multiProgressBars, error) {
	program, err := newMultiProgressMockProgram(elements, orderedKeys, input, output)
	if err != nil {
		return nil, err
	}
	return &multiProgressBars{program: program}, nil
}
