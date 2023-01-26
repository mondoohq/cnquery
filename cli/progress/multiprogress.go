package progress

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"go.mondoo.com/cnquery/cli/theme"
	"go.mondoo.com/cnquery/logger"
)

type MultiProgress interface {
	Open() error
	OnProgress(index string, percent float64)
	Score(index string, score string)
	Errored(index string)
	Completed(index string)
	Close()
}

type NoopMultiProgressBars struct{}

func (n NoopMultiProgressBars) Open() error                { return nil }
func (n NoopMultiProgressBars) OnProgress(string, float64) {}
func (n NoopMultiProgressBars) Score(string, string)       {}
func (n NoopMultiProgressBars) Errored(string)             {}
func (n NoopMultiProgressBars) Completed(string)           {}
func (n NoopMultiProgressBars) Close()                     {}

const (
	padding                  = 0
	maxWidth                 = 80
	defaultWidth             = 40
	defaultProgressNumAssets = 1
	overallProgressIndexName = "overall"
)

type MultiProgressAdapter struct {
	Multi MultiProgress
	Key   string
}

func (m *MultiProgressAdapter) Open() error { return m.Multi.Open() }
func (m *MultiProgressAdapter) OnProgress(current int, total int) {
	percent := 0.0
	if total > 0 {
		percent = float64(current) / float64(total)
	}
	m.Multi.OnProgress(m.Key, percent)
}
func (m *MultiProgressAdapter) Score(score string) { m.Multi.Score(m.Key, score) }
func (m *MultiProgressAdapter) Errored()           { m.Multi.Errored(m.Key) }
func (m *MultiProgressAdapter) Completed()         { m.Multi.Completed(m.Key) }
func (m *MultiProgressAdapter) Close()             { m.Multi.Close() }

type MsgProgress struct {
	Index   string
	Percent float64
}

// For cnquery the progressbar is completed, when percent is 1.0
// But for cnspec we also need the score, which is displayed after the progressbar
// So we need a second message to indicate when the progressbar is completed
type MsgCompleted struct {
	Index string
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
	percent   float64
	Name      string
	Score     string
	Completed bool
	Errored   bool
}

type modelMultiProgress struct {
	Progress       map[string]*modelProgress
	maxNameWidth   int
	maxItemsToShow int
	orderedKeys    []string
	lock           sync.Mutex
}

type multiProgressBars struct {
	program *tea.Program
}

func newProgressBar() progress.Model {
	progressbar := progress.New(progress.WithScaledGradient("#5A56E0", "#EE6FF8"))
	progressbar.Width = defaultWidth
	progressbar.Full = '━'
	progressbar.FullColor = "#7571F9"
	progressbar.Empty = '─'
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
func NewMultiProgressBars(elements map[string]string, orderedKeys []string) (*multiProgressBars, error) {
	program, err := newMultiProgressProgram(elements, orderedKeys)
	if err != nil {
		return nil, err
	}
	return &multiProgressBars{program: program}, nil
}

// Start the progress bars
// Form now on the progress bars can be updated
func (m *multiProgressBars) Open() error {
	(logger.LogOutputWriter.(*logger.BufferedWriter)).Pause()
	defer (logger.LogOutputWriter.(*logger.BufferedWriter)).Resume()
	if _, err := m.program.Run(); err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	return nil
}

// Set the current progress of a progress bar
func (m *multiProgressBars) OnProgress(index string, percent float64) {
	m.program.Send(MsgProgress{
		Index:   index,
		Percent: percent,
	})
}

// Add a score to the progress bar
// This should be called before Completed is called
func (m *multiProgressBars) Score(index string, score string) {
	m.program.Send(MsgScore{
		Index: index,
		Score: score,
	})
}

// This is called when an error occurs during the progress
func (m *multiProgressBars) Errored(index string) {
	m.program.Send(MsgErrored{
		Index: index,
	})
}

// Set a single bar to completed
// For cnquery this should be called after the progress is 100%
// For cnspec this should be called after the score is set
func (m *multiProgressBars) Completed(index string) {
	m.program.Send(MsgCompleted{
		Index: index,
	})
}

// This ends the multiprogrssbar no matter the current progress
func (m *multiProgressBars) Close() {
	m.program.Quit()
}

// create the actual tea.Program
func newMultiProgressProgram(elements map[string]string, orderedKeys []string) (*tea.Program, error) {
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

func (m *modelMultiProgress) Init() tea.Cmd {
	return nil
}

func (m *modelMultiProgress) add(key string, name string) {
	progressbar := newProgressBar()

	m.Progress[key] = &modelProgress{
		model:     &progressbar,
		Name:      name,
		Score:     "",
		Completed: false,
	}
}

func (m *modelMultiProgress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case MsgCompleted:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}
		m.lock.Lock()
		m.Progress[msg.Index].Completed = true
		m.lock.Unlock()

		if m.allDone() {
			return m, tea.Quit
		}
		return m, nil

	case MsgProgress:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		if msg.Percent != 0 {
			m.lock.Lock()
			m.Progress[msg.Index].percent = msg.Percent
			m.lock.Unlock()
		}

		m.updateOverallProgress()

		return m, nil

	case MsgErrored:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		m.lock.Lock()
		m.Progress[msg.Index].Errored = true
		m.Progress[msg.Index].model.ShowPercentage = false
		// settings ShowPercentage to false, expanse the progress bar to match the others
		// we need to manually reduce the width to match the others without the percentage
		m.Progress[msg.Index].model.Width -= 5
		m.lock.Unlock()

		m.updateOverallProgress()

		if m.allDone() {
			return m, tea.Quit
		}

		return m, nil

	case MsgScore:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		if msg.Score != "" {
			m.lock.Lock()
			m.Progress[msg.Index].Score = msg.Score
			m.lock.Unlock()
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

func (m *modelMultiProgress) allDone() bool {
	finished := 0
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}
		m.lock.Lock()
		if m.Progress[k].Errored || m.Progress[k].Completed {
			finished++
		}
		m.lock.Unlock()
	}
	allDone := false
	if _, ok := m.Progress[overallProgressIndexName]; ok {
		if finished == len(m.Progress)-1 {
			m.lock.Lock()
			m.Progress[overallProgressIndexName].Completed = true
			m.lock.Unlock()
		}
		allDone = m.Progress[overallProgressIndexName].Completed
	} else {
		allDone = finished == len(m.Progress)
	}

	return allDone
}

func (m *modelMultiProgress) updateOverallProgress() {
	if _, ok := m.Progress[overallProgressIndexName]; !ok {
		return
	}
	overallPercent := 0.0
	m.lock.Lock()
	sumPercent := 0.0
	validAssets := 0
	erroredAssets := 0
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}
		errored := m.Progress[k].Errored
		if errored {
			erroredAssets++
			continue
		}
		sumPercent += m.Progress[k].percent
		validAssets++
	}
	if validAssets > 0 {
		overallPercent = math.Floor((sumPercent/float64(validAssets))*100) / 100
	}
	_, ok := m.Progress[overallProgressIndexName]
	if ok && erroredAssets == len(m.Progress)-1 {
		overallPercent = 1.0
	}
	m.Progress[overallProgressIndexName].percent = overallPercent
	m.lock.Unlock()
	return
}

func (m *modelMultiProgress) View() string {
	pad := strings.Repeat(" ", padding)
	output := ""

	completedAssets := 0
	erroredAssets := 0
	for _, k := range m.orderedKeys {
		m.lock.Lock()
		if m.Progress[k].Errored {
			erroredAssets++
		}
		if m.Progress[k].Completed {
			completedAssets++
		}
		m.lock.Unlock()
	}
	outputFinished := ""
	numItemsFinished := 0
	for _, k := range m.orderedKeys {
		m.lock.Lock()
		errored := m.Progress[k].Errored
		completed := m.Progress[k].Completed
		m.lock.Unlock()
		if !errored && !completed {
			continue
		}
		pad := strings.Repeat(" ", m.maxNameWidth-len(m.Progress[k].Name))
		if errored {
			outputFinished += m.Progress[k].model.View() + theme.DefaultTheme.Error("    X "+m.Progress[k].Name)
		} else if completed {
			m.lock.Lock()
			percent := m.Progress[k].percent
			m.lock.Unlock()
			outputFinished += m.Progress[k].model.ViewAs(percent) + " " + m.Progress[k].Name
		}
		m.lock.Lock()
		score := m.Progress[k].Score
		m.lock.Unlock()
		if score != "" {
			if errored {
				outputFinished += pad + theme.DefaultTheme.Error(" score: "+score)
			} else {
				outputFinished += pad + " score: " + score
			}
		}
		outputFinished += "\n"
		numItemsFinished++
	}

	itemsInProgress := 0
	outputNotDone := ""
	for _, k := range m.orderedKeys {
		m.lock.Lock()
		errored := m.Progress[k].Errored
		completed := m.Progress[k].Completed
		m.lock.Unlock()
		if errored || completed {
			continue
		}
		m.lock.Lock()
		percent := m.Progress[k].percent
		m.lock.Unlock()
		outputNotDone += m.Progress[k].model.ViewAs(percent) + " " + m.Progress[k].Name + "\n"
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
		m.lock.Lock()
		percent := m.Progress[overallProgressIndexName].percent
		m.lock.Unlock()
		output += "\n" + m.Progress[overallProgressIndexName].model.ViewAs(percent) + " " + m.Progress[overallProgressIndexName].Name
		output += fmt.Sprintf(" %d/%d scanned", completedAssets, len(m.Progress)-1)
		if erroredAssets > 0 {
			output += fmt.Sprintf(" %d/%d errored", erroredAssets, len(m.Progress)-1)
		}
		output += "\n"
	}

	return "\n" + pad + output + "\n"
}
