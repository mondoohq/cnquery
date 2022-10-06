package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/logger"
)

type Progress interface {
	Open() error
	OnProgress(current int, total int)
	Close()
}

type Noop struct{}

func (n Noop) Open() error         { return nil }
func (n Noop) OnProgress(int, int) {}
func (n Noop) Close()              {}

type progressbar struct {
	id           string
	maxNameWidth int
	padding      int
	Data         progressData
	lock         sync.Mutex
	bar          *renderer
	isTTY        bool
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
		isTTY:        isatty.IsTerminal(os.Stdout.Fd()) && false, // FIXME: re-enable the detection
	}
}

func (p *progressbar) Open() error {
	var err error
	p.bar, err = newRenderer()
	if err != nil {
		return errors.Wrap(err, "failed to initialize progressbar renderer")
	}

	if p.isTTY {
		go func() {
			(logger.LogOutputWriter.(*logger.BufferedWriter)).Pause()
			defer (logger.LogOutputWriter.(*logger.BufferedWriter)).Resume()

			if err := tea.NewProgram(p).Start(); err != nil {
				panic(err)
			}
		}()
	} else {
		go func() {
			for {
				time.Sleep(time.Second / progressPipedFps)
				if p.Data.complete {
					break
				}
				fmt.Print(p.View() + "\r\033[2A")
			}

			fmt.Print(p.View())
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
	p.Data.complete = true
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
	if p.Data.complete {
		return p, tea.Quit
	}

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
		return p, tickCmd()

	default:
		return p, nil
	}
}

// View is a required interface method for the underlying renderer
func (p *progressbar) View() string {
	pad := strings.Repeat(" ", p.padding)

	out := ""
	p.lock.Lock()
	for i := range p.Data.Names {
		name := p.Data.Names[i]
		value := p.Data.Completion[i]
		out += "\n" + pad + p.bar.View(value) + " " + name
	}
	p.lock.Unlock()

	out += "\n"
	return out
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/progressDefaultFps, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
