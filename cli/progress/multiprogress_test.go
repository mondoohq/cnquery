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
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 0.5)
		multiprogress.OnProgress("2", 0.5)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "50% test2\r\n")
	assert.Contains(t, buf.String(), "50% overall 1/3 scanned")
}

func TestMultiProgressBarSingleAsset(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 0.5)
		multiprogress.OnProgress("2", 0.5)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1 score: F")
	assert.NotContains(t, buf.String(), "test2")
	assert.NotContains(t, buf.String(), "overall")
}

func TestMultiProgressBarFinished(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.OnProgress("2", 1.0)
		multiprogress.OnProgress("3", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Score("2", "F")
		multiprogress.Completed("2")
		multiprogress.Score("3", "F")
		multiprogress.Completed("3")
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 scanned")
}

func TestMultiProgressBarErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Score("2", "X")
		multiprogress.Errored("2")
		multiprogress.OnProgress("3", 1.0)
		multiprogress.Score("3", "F")
		multiprogress.Completed("3")
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "   X test2   score: X")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 2/3 scanned 1/3 errored")
}

func TestMultiProgressBarLastErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.OnProgress("2", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Score("2", "F")
		multiprogress.Completed("2")
		multiprogress.Score("3", "X")
		multiprogress.Errored("3")
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "   X test3   score: X")
	assert.Contains(t, buf.String(), "100% overall 2/3 scanned 1/3 errored")
}

func TestMultiProgressBarOnlyOneErrored(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		// this should also end the tea program
		multiprogress.Errored("1")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "   X test1")
}

func TestMultiProgressBarLimitedNumber(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.OnProgress("2", 0.1)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.NotContains(t, buf.String(), "100% test2   score: F")
	assert.NotContains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "36% overall 1/3 scanned")
	assert.Contains(t, buf.String(), "1 more assets")
}

func TestMultiProgressBarLimitedOneMore(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3", "4": "test4"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "2", "3", "4"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.OnProgress("2", 0.1)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "10% test2")
	assert.Contains(t, buf.String(), "27% overall 1/4 scanned")
	assert.Contains(t, buf.String(), "2 more assets")
}

func TestMultiProgressBarError(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	_, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "3"}, &in, &buf)
	require.Error(t, err)
}

func TestMultiProgressBarOrdering(t *testing.T) {
	var in bytes.Buffer
	var buf bytes.Buffer

	progressBarElements := map[string]string{"1": "test1", "2": "test2", "3": "test3"}
	multiprogress, err := newMultiProgressBarsMock(progressBarElements, []string{"1", "3", "2"}, &in, &buf)
	require.NoError(t, err)

	go func() {
		// we need to wait for tea to start the Program, otherwise these would be no-ops
		time.Sleep(1 * time.Millisecond)
		multiprogress.OnProgress("1", 1.0)
		multiprogress.OnProgress("2", 1.0)
		multiprogress.OnProgress("3", 1.0)
		multiprogress.Score("1", "F")
		multiprogress.Completed("1")
		multiprogress.Score("2", "F")
		multiprogress.Completed("2")
		multiprogress.Score("3", "F")
		multiprogress.Completed("3")
		multiprogress.Close()
	}()
	err = multiprogress.Open()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "100% test1   score: F")
	assert.Contains(t, buf.String(), "100% test2   score: F")
	assert.Contains(t, buf.String(), "100% test3   score: F")
	assert.Contains(t, buf.String(), "100% overall 3/3 scanned")
	// regexp is not working, perhaps because of ansi escape characters???
	// ordering := regexp.MustCompile(`^.*test1.*test3.*test2.*$`)
	// m := ordering.FindString(buf.String())
	assert.Contains(t, buf.String(), "test1   score: F\r\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 100% test3   score: F\r\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 100% test2")
}
