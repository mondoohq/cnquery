// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/utils/multierr"
)

type Progress interface {
	Open() error
	OnProgress(current int, total int)
	Score(score string)
	Errored()
	NotApplicable()
	Completed()
	Close()
}

type Noop struct{}

func (n Noop) Open() error         { return nil }
func (n Noop) OnProgress(int, int) {}
func (n Noop) Score(score string)  {}
func (n Noop) Errored()            {}
func (n Noop) NotApplicable()      {}
func (n Noop) Completed()          {}
func (n Noop) Close()              {}

type progressbar struct {
	id           string
	maxNameWidth int
	padding      int
	Data         progressData
	lock         sync.Mutex
	bar          *renderer
	isTTY        bool
	wg           sync.WaitGroup
}

type progressData struct {
	Names      []string
	Completion []float32
	complete   bool
}

func New(id string, name string) *progressbar {
	return NewMultiBar(id, progressData{
		Names:      []string{name},
		Completion: []float32{0},
		complete:   false,
	})
}

func NewMultiBar(id string, data progressData) *progressbar {
	maxNameWidth := 0
	for _, v := range data.Names {
		l := len(v)
		if l > maxNameWidth {
			maxNameWidth = l
		}
	}

	return &progressbar{
		id:           id,
		maxNameWidth: maxNameWidth,
		Data:         data,
		isTTY:        isatty.IsTerminal(os.Stdout.Fd()),
	}
}

func (p *progressbar) Errored()       {}
func (p *progressbar) NotApplicable() {}
func (p *progressbar) Score(string)   {}
func (p *progressbar) Completed()     {}

func (p *progressbar) Open() error {
	var err error
	p.bar, err = newRenderer()
	if err != nil {
		return multierr.Wrap(err, "failed to initialize progressbar renderer")
	}

	p.wg.Add(1)
	if p.isTTY {
		go func() {
			defer p.wg.Done()
			(logger.LogOutputWriter.(*logger.BufferedWriter)).Pause()
			defer (logger.LogOutputWriter.(*logger.BufferedWriter)).Resume()
			if _, err := tea.NewProgram(p).Run(); err != nil {
				fmt.Println(err.Error())
				panic(err)
			}
		}()
	} else {
		go func() {
			defer p.wg.Done()
			o := termenv.NewOutput(os.Stdout)
			for {
				time.Sleep(time.Second / progressPipedFps)
				o.ClearLines(2)
				o.WriteString(p.View())
				p.lock.Lock()
				complete := p.Data.complete
				p.lock.Unlock()
				if complete {
					break
				}
			}
		}()
	}

	return nil
}

func (p *progressbar) OnProgress(current int, total int) {
	p.lock.Lock()
	p.Data.Completion[0] = float32(current) / float32(total)
	p.lock.Unlock()
}

func (p *progressbar) Close() {
	p.lock.Lock()
	p.Data.complete = true
	p.lock.Unlock()
	p.wg.Wait()
}

const (
	progressDefaultFps   = 60
	progressDefaultWidth = 80
	progressPipedFps     = 1
)

type tickMsg time.Time

// Init is a required interface method for the underlying renderer
func (p *progressbar) Init() tea.Cmd {
	return tickCmd()
}

// Update is a required interface method for the underlying renderer
func (p *progressbar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return p, tea.Quit
		default:
			return p, nil
		}

	case tea.WindowSizeMsg:
		p.bar.Width = msg.Width - p.padding*2 - 4 - p.maxNameWidth
		if p.bar.Width > progressDefaultWidth {
			p.bar.Width = progressDefaultWidth
		}
		return p, nil

	case tickMsg:
		p.lock.Lock()
		complete := p.Data.complete
		p.lock.Unlock()
		if complete {
			return p, tea.Quit
		}
		return p, tickCmd()

	default:
		return p, nil
	}
}

// View is a required interface method for the underlying renderer
func (p *progressbar) View() string {
	pad := strings.Repeat(" ", p.padding)
	out := ""
	for i := range p.Data.Names {
		name := p.Data.Names[i]
		value := p.Data.Completion[i]
		out += "\n" + pad + p.bar.View(value) + " " + name
	}

	out += "\n"
	return out
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/progressDefaultFps, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
