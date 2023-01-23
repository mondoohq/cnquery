package progress

import (
	"fmt"
	"math"
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
	defaultProgressNumAssets = 10
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
	orderedKeys    []string
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
// orderedKeys is used to define the order of the progress bars.
func NewMultiProgressProgram(elements map[string]string, orderedKeys []string) (Program, error) {
	if len(elements) != len(orderedKeys) {
		return nil, fmt.Errorf("number of elements and orderedKeys must be equal")
	}
	m := newMultiProgress(elements)
	m.maxItemsToShow = defaultProgressNumAssets
	m.orderedKeys = orderedKeys
	return tea.NewProgram(m), nil
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

		cmds = append(cmds, m.updateOverallProgress())

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
		return m, m.updateOverallProgress()

	case MsgScore:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		if msg.Score != "" {
			m.Progress[msg.Index].lock.Lock()
			m.Progress[msg.Index].Score = msg.Score
			m.Progress[msg.Index].lock.Unlock()
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

func (m modelMultiProgress) updateOverallProgress() tea.Cmd {
	if _, ok := m.Progress[overallProgressIndexName]; !ok {
		return nil
	}
	overallPercent := 0.0
	m.Progress[overallProgressIndexName].lock.Lock()
	sumPercent := 0.0
	validAssets := 0
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}
		if m.Progress[k].Errored {
			continue
		}
		m.Progress[k].lock.Lock()
		sumPercent += m.Progress[k].model.Percent()
		m.Progress[k].lock.Unlock()
		validAssets++
	}
	overallPercent = math.Floor((sumPercent/float64(validAssets))*100) / 100
	cmd := m.Progress[overallProgressIndexName].model.SetPercent(overallPercent)
	m.Progress[overallProgressIndexName].lock.Unlock()
	return cmd
}

func (m modelMultiProgress) View() string {
	pad := strings.Repeat(" ", padding)
	output := ""

	completedAssets := 0
	erroredAssets := 0
	for _, k := range m.orderedKeys {
		if m.Progress[k].Errored {
			erroredAssets++
		}
		if m.Progress[k].Completed {
			completedAssets++
		}
	}
	outputFinished := ""
	numItemsFinished := 0
	for _, k := range m.orderedKeys {
		if !m.Progress[k].Errored && !m.Progress[k].Completed {
			continue
		}
		pad := strings.Repeat(" ", m.maxNameWidth-len(m.Progress[k].Name))
		if m.Progress[k].Errored {
			outputFinished += m.Progress[k].model.View() + "    X " + m.Progress[k].Name
		} else if m.Progress[k].Completed {
			outputFinished += m.Progress[k].model.ViewAs(m.Progress[k].model.Percent()) + " " + m.Progress[k].Name
		}
		if m.Progress[k].Score != "" {
			outputFinished += pad + " score: " + m.Progress[k].Score
		}
		outputFinished += "\n"
		numItemsFinished++
	}

	itemsInProgress := 0
	outputNotDone := ""
	for _, k := range m.orderedKeys {
		if m.Progress[k].Errored || m.Progress[k].Completed {
			continue
		}
		outputNotDone += m.Progress[k].model.ViewAs(m.Progress[k].model.Percent()) + " " + m.Progress[k].Name + "\n"
		itemsInProgress++
		if itemsInProgress == m.maxItemsToShow {
			break
		}
	}
	itemsUnfinished := len(m.orderedKeys) - itemsInProgress - numItemsFinished
	if m.maxItemsToShow > 0 && itemsUnfinished > 0 {
		outputNotDone += fmt.Sprintf("... %d more assets ...\n", itemsUnfinished)
	}

	output += outputFinished + outputNotDone
	if _, ok := m.Progress[overallProgressIndexName]; ok {
		output += "\n" + m.Progress[overallProgressIndexName].model.ViewAs(m.Progress[overallProgressIndexName].model.Percent()) + " " + m.Progress[overallProgressIndexName].Name
		output += fmt.Sprintf(" %d/%d scanned", completedAssets, len(m.Progress)-1)
		if erroredAssets > 0 {
			output += fmt.Sprintf(" %d/%d errored", erroredAssets, len(m.Progress)-1)
		}
		output += "\n"
	}

	return "\n" + pad + output + "\n"
}
