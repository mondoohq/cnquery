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
	progressProgram := newMultiProgressMockProgram(progressBarElements, &in, &buf)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Quit()
	}()
	_, err := progressProgram.Run()
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
	progressProgram := newMultiProgressMockProgram(progressBarElements, &in, &buf)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		progressProgram.Send(MsgProgress{Index: "1", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "2", Percent: 0.5})
		progressProgram.Send(MsgProgress{Index: "1", Percent: 1.0})
		progressProgram.Send(MsgScore{Index: "1", Score: "F"})
		progressProgram.Quit()
	}()
	_, err := progressProgram.Run()
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
	progressProgram := newMultiProgressMockProgram(progressBarElements, &in, &buf)

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
	_, err := progressProgram.Run()
	defer progressProgram.Quit()
	require.NoError(t, err)
	progressProgram.Quit()
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 assets")
}
