// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/ansi"
	"go.mondoo.com/cnquery/v11/cli/components"
	"go.mondoo.com/cnquery/v11/cli/theme"
	"go.mondoo.com/cnquery/v11/logger"
)

type ProgressOption = func(*modelMultiProgress)

func WithScore() ProgressOption {
	return func(p *modelMultiProgress) {
		p.includeScore = true
	}
}

type MultiProgress interface {
	Open() error
	OnProgress(index string, percent float64)
	Score(index string, score string)
	Errored(index string)
	NotApplicable(index string)
	Completed(index string)
	Close()
}

type NoopMultiProgressBars struct{}

func (n NoopMultiProgressBars) Open() error                { return nil }
func (n NoopMultiProgressBars) OnProgress(string, float64) {}
func (n NoopMultiProgressBars) Score(string, string)       {}
func (n NoopMultiProgressBars) Errored(string)             {}
func (n NoopMultiProgressBars) NotApplicable(string)       {}
func (n NoopMultiProgressBars) Completed(string)           {}
func (n NoopMultiProgressBars) Close()                     {}

const (
	padding                  = 0
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
func (m *MultiProgressAdapter) NotApplicable()     { m.Multi.NotApplicable(m.Key) }
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

type MsgNotApplicable struct {
	Index string
}

type MsgScore struct {
	Index string
	Score string
}

type ProgressState int

const (
	ProgressStateUnknownProgressState = iota
	ProgressStateNotApplicable
	ProgressStateCompleted
	ProgressStateErrored
)

type modelProgress struct {
	model         *progress.Model
	percent       float64
	Name          string
	Score         string
	ProgressState ProgressState
}

type modelMultiProgress struct {
	Progress           map[string]*modelProgress
	maxNameWidth       int
	maxItemsToShow     int
	orderedKeys        []string
	lock               sync.Mutex
	maxProgressBarWith int
	includeScore       bool
}

type multiProgressBars struct {
	program        *tea.Program
	Progress       map[string]*modelProgress
	maxNameWidth   int
	maxItemsToShow int
	orderedKeys    []string
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
// includeScore indicates if the score should be displayed after the progress bar. This will only be used for spacing
func NewMultiProgressBars(elements map[string]string, orderedKeys []string, opts ...ProgressOption) (*multiProgressBars, error) {
	program, err := newMultiProgressProgram(elements, orderedKeys, opts...)
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

// This is called when an error occurs during the progress
func (m *multiProgressBars) NotApplicable(index string) {
	m.program.Send(MsgNotApplicable{
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
func newMultiProgressProgram(elements map[string]string, orderedKeys []string, opts ...ProgressOption) (*tea.Program, error) {
	if len(elements) != len(orderedKeys) {
		return nil, fmt.Errorf("number of elements and orderedKeys must be equal")
	}
	m := newMultiProgress(elements, opts...)
	m.maxItemsToShow = defaultProgressNumAssets
	m.orderedKeys = orderedKeys
	return tea.NewProgram(m), nil
}

func newMultiProgress(elements map[string]string, opts ...ProgressOption) *modelMultiProgress {
	numBars := len(elements)
	if numBars > 1 {
		numBars++
	}
	multiprogress := make(map[string]*modelProgress, numBars)

	m := &modelMultiProgress{
		Progress:           multiprogress,
		maxNameWidth:       0,
		maxProgressBarWith: defaultWidth,
	}
	for _, opt := range opts {
		opt(m)
	}

	if numBars > 1 {
		// add overall with max possible length, so we do not have to move progress bars later on
		overallName := fmt.Sprintf("%d/%d scanned %d/%d errored %d/%d n/a", numBars, numBars, numBars, numBars, numBars, numBars)
		m.add(overallProgressIndexName, overallName, m.maxProgressBarWith)
	}

	w := m.calculateMaxProgressBarWidth()
	if w > 10 {
		m.maxProgressBarWith = w
	}

	for k, v := range elements {
		m.add(k, v, m.maxProgressBarWith)
	}

	maxNameWidth := 0
	for k := range m.Progress {
		if len(m.Progress[k].Name) > maxNameWidth {
			maxNameWidth = ansi.PrintableRuneWidth(m.Progress[k].Name)
		}
	}
	m.maxNameWidth = maxNameWidth

	return m
}

func (m *modelMultiProgress) Init() tea.Cmd {
	return nil
}

func (m *modelMultiProgress) calculateMaxProgressBarWidth() int {
	w := 0
	terminalWidth, err := components.TerminalWidth(os.Stdout)
	if err == nil {
		w = terminalWidth - m.maxNameWidth - 8 // 5 for percentage + space
		// space for " score: F"
		if m.includeScore {
			w -= 9
		}
	}
	return w
}

func (m *modelMultiProgress) add(key string, name string, width int) {
	progressbar := newProgressBar()
	progressbar.Width = width
	m.Progress[key] = &modelProgress{
		model: &progressbar,
		Name:  name,
		Score: "",
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
		w := m.calculateMaxProgressBarWidth()
		if w > 10 {
			m.maxProgressBarWith = w
		}
		for k := range m.Progress {
			m.Progress[k].model.Width = m.maxProgressBarWith
		}
		return m, nil

	case MsgCompleted:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}
		m.lock.Lock()
		m.Progress[msg.Index].ProgressState = ProgressStateCompleted
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

	case MsgNotApplicable:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		m.lock.Lock()
		m.Progress[msg.Index].ProgressState = ProgressStateNotApplicable
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
	case MsgErrored:
		if _, ok := m.Progress[msg.Index]; !ok {
			return m, nil
		}

		m.lock.Lock()
		m.Progress[msg.Index].ProgressState = ProgressStateErrored
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
	m.lock.Lock()
	defer m.lock.Unlock()
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}
		if m.Progress[k].ProgressState == ProgressStateErrored ||
			m.Progress[k].ProgressState == ProgressStateNotApplicable ||
			m.Progress[k].ProgressState == ProgressStateCompleted {
			finished++
		}
	}
	allDone := false
	if _, ok := m.Progress[overallProgressIndexName]; ok {
		if finished == len(m.Progress)-1 {
			m.Progress[overallProgressIndexName].ProgressState = ProgressStateCompleted
		}
		allDone = m.Progress[overallProgressIndexName].ProgressState == ProgressStateCompleted
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
	defer m.lock.Unlock()
	sumPercent := 0.0
	validAssets := 0
	erroredAssets := 0
	notApplicableAssets := 0
	for k := range m.Progress {
		if k == overallProgressIndexName {
			continue
		}

		switch m.Progress[k].ProgressState {
		case ProgressStateErrored:
			erroredAssets++
			continue
		case ProgressStateNotApplicable:
			notApplicableAssets++
			continue
		}

		sumPercent += m.Progress[k].percent
		validAssets++
	}
	if validAssets > 0 {
		overallPercent = math.Floor((sumPercent/float64(validAssets))*100) / 100
	}
	_, ok := m.Progress[overallProgressIndexName]
	if ok && erroredAssets+notApplicableAssets == len(m.Progress)-1 {
		overallPercent = 1.0
	}
	m.Progress[overallProgressIndexName].percent = overallPercent

	return
}

func (m *modelMultiProgress) View() string {
	pad := strings.Repeat(" ", padding)
	output := ""

	m.lock.Lock()
	defer m.lock.Unlock()
	completedAssets := 0
	erroredAssets := 0
	notApplicableAssets := 0
	for _, k := range m.orderedKeys {
		switch m.Progress[k].ProgressState {
		case ProgressStateErrored:
			erroredAssets++
		case ProgressStateNotApplicable:
			notApplicableAssets++
		case ProgressStateCompleted:
			completedAssets++
		}
	}
	outputFinished := ""
	numItemsFinished := 0
	for _, k := range m.orderedKeys {
		progressState := m.Progress[k].ProgressState
		if progressState != ProgressStateErrored && progressState != ProgressStateCompleted && progressState != ProgressStateNotApplicable {
			continue
		}
		name := m.Progress[k].Name
		pad := strings.Repeat(" ", m.maxNameWidth-len(name))
		switch progressState {
		case ProgressStateErrored:
			outputFinished += " " + theme.DefaultTheme.Error(name) + pad + " " + m.Progress[k].model.View() + theme.DefaultTheme.Error("    X")
		case ProgressStateNotApplicable:
			outputFinished += " " + name + pad + " " + m.Progress[k].model.View() + "  n/a"
		case ProgressStateCompleted:
			percent := m.Progress[k].percent
			outputFinished += " " + name + pad + " " + m.Progress[k].model.ViewAs(percent)
		}

		score := m.Progress[k].Score
		if score != "" {
			switch progressState {
			case ProgressStateErrored:
				outputFinished += theme.DefaultTheme.Error(" score: " + score)
			case ProgressStateNotApplicable:
				outputFinished += " score: " + score
			default:
				outputFinished += " score: " + score
			}
		}
		outputFinished += "\n"
		numItemsFinished++
	}

	itemsInProgress := 0
	outputNotDone := ""
	for _, k := range m.orderedKeys {
		progressState := m.Progress[k].ProgressState
		if progressState == ProgressStateErrored || progressState == ProgressStateNotApplicable || progressState == ProgressStateCompleted {
			continue
		}
		name := m.Progress[k].Name
		pad := strings.Repeat(" ", m.maxNameWidth-len(name))
		percent := m.Progress[k].percent
		outputNotDone += " " + name + pad + " " + m.Progress[k].model.ViewAs(percent) + "\n"
		itemsInProgress++
		if itemsInProgress == m.maxItemsToShow {
			break
		}
	}
	itemsUnfinished := len(m.orderedKeys) - itemsInProgress - numItemsFinished
	if m.maxItemsToShow > 0 && itemsUnfinished > 0 {
		label := "asset"
		if itemsUnfinished > 1 {
			label = "assets"
		}
		outputNotDone += fmt.Sprintf("... %d more %s ...\n", itemsUnfinished, label)
	}

	output += outputFinished + outputNotDone
	if _, ok := m.Progress[overallProgressIndexName]; ok {
		percent := m.Progress[overallProgressIndexName].percent
		stats := fmt.Sprintf("%d/%d scanned", completedAssets, len(m.Progress)-1)

		if erroredAssets > 0 {
			stats += fmt.Sprintf(" %d/%d errored", erroredAssets, len(m.Progress)-1)
		}

		if notApplicableAssets > 0 {
			stats += fmt.Sprintf(" %d/%d n/a", notApplicableAssets, len(m.Progress)-1)
		}

		repeat := m.maxNameWidth - len(stats)
		if repeat < 0 {
			repeat = 0
		}
		pad := strings.Repeat(" ", repeat)
		output += "\n"
		output += " " + stats + pad + " " + m.Progress[overallProgressIndexName].model.ViewAs(percent)
	}

	return "\n" + pad + output + "\n\n"
}
