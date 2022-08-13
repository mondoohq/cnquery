package events

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WatcherTester struct {
	mock    providers.Transport
	watcher *Watcher
}

func SetupWatcherTest() *WatcherTester {
	filepath, _ := filepath.Abs("testdata/watcher_test.toml")
	trans, _ := mock.NewFromTomlFile(filepath)
	return &WatcherTester{watcher: NewWatcher(trans), mock: trans}
}

func TeardownWatcherTest(wt *WatcherTester) {
	wt.watcher.TearDown()
}

func TestCommandSubscribe(t *testing.T) {
	var wg sync.WaitGroup
	wt := SetupWatcherTest()
	w := wt.watcher

	var res *CommandObservable
	wg.Add(1)
	w.Subscribe("command", "hostname", func(co providers.Observable) {
		switch x := co.(type) {
		case *CommandObservable:
			defer wg.Done()
			res = x
		default:
		}
	})
	wg.Wait()

	stdout, err := ioutil.ReadAll(res.Result.Stdout)
	assert.Nil(t, err, "could extract stdout")
	assert.Equal(t, "mockland.local", string(stdout), "get the expected command output")
	TeardownWatcherTest(wt)
}

func TestFileSubscribe(t *testing.T) {
	var wg sync.WaitGroup
	wt := SetupWatcherTest()
	w := wt.watcher

	var res *FileObservable

	wg.Add(1)
	err := w.Subscribe("file", "/tmp/test", func(fo providers.Observable) {
		switch x := fo.(type) {
		case *FileObservable:
			defer wg.Done()
			res = x
		default:
		}
	})
	require.NoError(t, err)
	wg.Wait()
	content, err := ioutil.ReadAll(res.File)
	assert.Nil(t, err, "file content was returned without any error")
	assert.Equal(t, "test", string(content), "get the expected command output")

	TeardownWatcherTest(wt)
}

func TestFileChangeEvents(t *testing.T) {
	var waitInitialRead sync.WaitGroup
	var waitFileUpdate sync.WaitGroup
	var waitSecondRead sync.WaitGroup

	wt := SetupWatcherTest()
	w := wt.watcher
	// wait 500ms
	w.SleepDuration = time.Duration(2 * time.Millisecond)

	res := []string{}
	readCount := 0
	waitInitialRead.Add(1)
	waitFileUpdate.Add(1)
	waitSecondRead.Add(1)

	err := w.Subscribe("file", "/tmp/test", func(fo providers.Observable) {
		switch x := fo.(type) {
		case *FileObservable:
			if readCount == 0 {
				defer waitInitialRead.Done()
			} else if readCount == 1 {
				waitFileUpdate.Wait()
				defer waitSecondRead.Done()
			} else {
				return
			}
			content, err := ioutil.ReadAll(x.File)
			if err == nil {
				res = append(res, string(content))
			}
			readCount++
		default:
		}
	})
	require.NoError(t, err)

	waitInitialRead.Wait()

	// change file content
	mt := wt.mock.(*mock.Provider)
	mt.Fs.Files["/tmp/test"].Content = "newtest"
	waitFileUpdate.Done()

	waitSecondRead.Wait()

	assert.Equal(t, []string{"test", "newtest"}, res, "detect file change")

	TeardownWatcherTest(wt)
}
