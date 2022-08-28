package internal

type ProgressReporter interface {
	Progress(numCompleted int, total int)
}

type NoopProgressReporter struct {
}

func (NoopProgressReporter) Progress(numCompleted int, total int) {}

type ProgressReporterFunc func(numCompleted int, total int)

func (f ProgressReporterFunc) Progress(numCompleted int, total int) {
	f(numCompleted, total)
}
