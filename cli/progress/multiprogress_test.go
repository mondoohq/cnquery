package progress

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiProgressBar(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "2", "3"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "50% test2\r\n")
	assert.Contains(t, buf.String(), "50% overall 1/3 assets")
}

func TestMultiProgressBarSingleAsset(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1 score: F")
	assert.NotContains(t, buf.String(), "test2")
	assert.NotContains(t, buf.String(), "overall")
}

func TestMultiProgressBarFinished(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "2", "3"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "3", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Send(MsgScore{Index: "2", Score: "F"})
		progressProgram.Send(MsgScore{Index: "3", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 assets")
}

func TestMultiProgressBarErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "2", "3"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Send(MsgScore{Index: "2", Score: "F"})
		progressProgram.Send(MsgErrored{Index: "3"})
		progressProgram.Send(MsgScore{Index: "3", Score: "X"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "   X test3   score: X")
	assert.Contains(t, buf.String(), "67% overall 2/3 assets 1/3 errors")
}

func TestMultiProgressBarOnlyOneErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		// this should also end the tea program
		progressProgram.Send(MsgErrored{Index: "1"})
	}()
	_, err = progressProgram.Run()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "   X test1")
}

func TestMultiProgressBarLimitedNumber(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "2", "3"}, 1, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "3", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Send(MsgScore{Index: "2", Score: "F"})
		progressProgram.Send(MsgScore{Index: "3", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.NotContains(t, buf.String(), "100% test2   score: F")
	assert.NotContains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 assets")
	assert.Contains(t, buf.String(), "2 more assets")
}

func TestMultiProgressBarLimitedOneMore(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "2", "3"}, 2, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "3", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Send(MsgScore{Index: "2", Score: "F"})
		progressProgram.Send(MsgScore{Index: "3", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.NotContains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 assets")
	assert.Contains(t, buf.String(), "1 more assets")
}

func TestMultiProgressBarError(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	_, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "3"}, 1, &in, &buf)
	require.Error(t, err)
}

func TestMultiProgressBarOrdering(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	progressProgram, err := newMultiProgressMockProgram(progressBarElements, []string{"1", "3", "2"}, 30, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 1.0})
		progressProgram.Send(MsgProgress{Index: "3", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Send(MsgScore{Index: "2", Score: "F"})
		progressProgram.Send(MsgScore{Index: "3", Score: "F"})
		progressProgram.Quit()
	}()
	_, err = progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 assets")
	// regexp is not working, perhaps because of ansi escape characters???
	// ordering := regexp.MustCompile(`^.*test1.*test3.*test2.*$`)
	// m := ordering.FindString(buf.String())
	assert.Contains(t, buf.String(), "test1   score: F\r\n███████████████████████████████████ 100% test3   score: F\r\n███████████████████████████████████ 100% test2")
}
