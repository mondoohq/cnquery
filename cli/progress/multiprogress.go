package progress

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// reduced tea.Program interface
type Program interface {
	Send(msg tea.Msg)
	Run() (tea.Model, error)
	Kill()
	Quit()
}

type NoopProgram struct{}

func (n NoopProgram) Send(msg tea.Msg)        {}
func (n NoopProgram) Run() (tea.Model, error) { return progress.Model{}, nil }
func (n NoopProgram) Kill()                   {}
func (n NoopProgram) Quit()                   {}

const (
	padding                  = 0
	maxWidth                 = 80
	defaultWidth             = 40
	overallProgressIndexName = "overall"
)

type MsgProgress struct {
	Index   string
	Percent float64
}

type MsgErrored struct {
	Index string
}

type MsgScore struct {
	Index string
	Score string
}

type modelProgress struct {
	model     *progress.Model
	Name      string
	Score     string
	Completed bool
	Errored   bool
	lock      sync.Mutex
}

type modelMultiProgress struct {
	Progress       map[string]*modelProgress
	maxNameWidth   int
	maxItemsToShow int
}

func newProgressBar() progress.Model {
	progressbar := progress.New(progress.WithScaledGradient("#5A56E0", "#EE6FF8"))
	progressbar.Width = defaultWidth
	progressbar.Full = '█'
	progressbar.FullColor = "#7571F9"
	progressbar.Empty = '░'
	progressbar.EmptyColor = "#606060"
	progressbar.ShowPercentage = true
	progressbar.PercentFormat = " %3.0f%%"
	return progressbar
}

// Creates a new progress bars for the given elements.
// This is a wrapper around a tea.Programm.
// The key of the map is used to identify the progress bar.
// The value of the map is used as the name displayed for the progress bar.
func NewMultiProgressProgram(elements map[string]string, progressNumAssets int) Program {
	m := newMultiProgress(elements)
	m.maxItemsToShow = progressNumAssets
	return tea.NewProgram(m)
}

// This is only needed for testing.
// This way we get the output without having a tty.
func newMultiProgressMockProgram(elements map[string]string, input io.Reader, output io.Writer) Program {
	m := newMultiProgress(elements)
	return tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output))
}

func newMultiProgress(elements map[string]string) *modelMultiProgress {
	numBars := len(elements)
	if numBars > 1 {
		numBars++
	}
	multiprogress := make(map[string]*modelProgress, numBars)

	m := &modelMultiProgress{
		Progress:     multiprogress,
		maxNameWidth: 0,
	}

	maxNameWidth := 0
	if numBars > 1 {
		m.add(overallProgressIndexName, "overall")
		maxNameWidth = len("overall")
	}

	for k, v := range elements {
		if len(v) > maxNameWidth {
			maxNameWidth = len(v)
		}
		m.add(k, v)
	}
	m.maxNameWidth = maxNameWidth

	return m
}

func (m modelMultiProgress) Init() tea.Cmd {
	return nil
}

func (m modelMultiProgress) add(key string, name string) {
	progressbar := newProgressBar()

	m.Progress[key] = &modelProgress{
		model:     &progressbar,
		Name:      name,
		Score:     "",
		Completed: false,
	}
}

func (m modelMultiProgress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}

	case tea.WindowSizeMsg:
		for k := range m.Progress {
			m.Progress[k].model.Update(msg)

			m.Progress[k].model.Width = msg.Width - padding*2 - 4 - m.maxNameWidth
			if m.Progress[k].model.Width > maxWidth {
				m.Progress[k].model.Width = maxWidth
			}

		}
		return m, nil

	case MsgProgress:
		var cmds []tea.Cmd
		var cmd tea.Cmd

		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}
		if msg.Percent != 0 {
			m.Progress[msg.Index].lock.Lock()
			cmd := m.Progress[msg.Index].model.SetPercent(msg.Percent)
			m.Progress[msg.Index].lock.Unlock()
			cmds = append(cmds, cmd)
		}
		if msg.Percent == 1.0 {
			m.Progress[msg.Index].lock.Lock()
			m.Progress[msg.Index].Completed = true
			m.Progress[msg.Index].lock.Unlock()
		}

		if _, ok := m.Progress[overallProgressIndexName]; ok {
			m.Progress[msg.Index].lock.Lock()
			sumPercent := 0.0
			validAssets := 0
			for k := range m.Progress {
				if k == overallProgressIndexName {
					continue
				}
				sumPercent += m.Progress[k].model.Percent()
				validAssets++
			}
			cmd = m.Progress[overallProgressIndexName].model.SetPercent(sumPercent / float64(validAssets))
			m.Progress[msg.Index].lock.Unlock()
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	case MsgErrored:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		m.Progress[msg.Index].lock.Lock()
		m.Progress[msg.Index].Errored = true
		m.Progress[msg.Index].model.ShowPercentage = false
		// settings ShowPercentage to false, expanse the progress bar to match the others
		// we need to manually reduce the width to match the others without the percentage
		m.Progress[msg.Index].model.Width -= 5
		m.Progress[msg.Index].lock.Unlock()
		return m, nil

	case MsgScore:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		if msg.Score != "" {
			m.Progress[msg.Index].lock.Lock()
			m.Progress[msg.Index].Score = msg.Score
			m.Progress[msg.Index].lock.Unlock()
			return m, nil
		}
		return m, nil

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		var cmds []tea.Cmd
		for k := range m.Progress {
			progressModel, cmd := m.Progress[k].model.Update(msg)
			cmds = append(cmds, cmd)
			if pModel, ok := progressModel.(progress.Model); ok {
				m.Progress[k].model = &pModel
			}
		}
		return m, tea.Batch(cmds...)

	default:
		return m, nil
	}
}

func (m modelMultiProgress) View() string {
	pad := strings.Repeat(" ", padding)
	output := ""

	completedAssets := 0
	erroredAssets := 0
	keys := []string{}
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}
		keys = append(keys, k)
		if m.Progress[k].Completed {
			completedAssets++
		} else if m.Progress[k].Errored {
			erroredAssets++
		}
	}

	sort.Strings(keys)
	keys = append(keys, overallProgressIndexName)

	i := 1
	for _, k := range keys {
		if k != overallProgressIndexName {
			if m.Progress[k].Errored {
				output += m.Progress[k].model.View() + "    X " + m.Progress[k].Name
			} else {
				output += m.Progress[k].model.ViewAs(m.Progress[k].model.Percent()) + " " + m.Progress[k].Name
			}
			if m.Progress[k].Completed && k != overallProgressIndexName && m.Progress[k].Score != "" {
				pad := strings.Repeat(" ", m.maxNameWidth-len(m.Progress[k].Name))
				output += pad + " score: " + m.Progress[k].Score
			}
		}
		output += "\n" + pad
		if i == m.maxItemsToShow {
			break
		}
		i++
	}
	if m.maxItemsToShow > 0 && len(keys) > m.maxItemsToShow+1 {
		output += fmt.Sprintf("... %d more assets ...\n%s", len(keys)-m.maxItemsToShow-1, pad)
	}

	if _, ok := m.Progress[overallProgressIndexName]; ok {
		output += "\n" + m.Progress[overallProgressIndexName].model.ViewAs(m.Progress[overallProgressIndexName].model.Percent()) + " " + m.Progress[overallProgressIndexName].Name
		output += fmt.Sprintf(" %d/%d assets", completedAssets, len(m.Progress)-1)
		if erroredAssets > 0 {
			output += fmt.Sprintf(" %d/%d errors", erroredAssets, len(m.Progress)-1)
		}
	}

	return "\n" + pad + output + "\n\n"
}
