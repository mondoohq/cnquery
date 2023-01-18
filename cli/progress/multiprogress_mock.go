package progress

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// This way we get the output without having a tty.
func newMultiProgressMockProgram(elements map[string]string, orderedKeys []string, progressNumAssets int, input io.Reader, output io.Writer) (Program, error) {
	if len(elements) != len(orderedKeys) {
		return nil, fmt.Errorf("number of elements and orderedKeys must be equal")
	}
	m := newMultiProgress(elements)
	m.maxItemsToShow = progressNumAssets
	m.orderedKeys = orderedKeys
	return tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output)), nil
}
