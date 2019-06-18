package events

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/mock/toml"
	"go.mondoo.io/mondoo/motor/motoros/types"

	"github.com/stretchr/testify/assert"
)

type WatcherTester struct {
	mock    types.Transport
	watcher *Watcher
}

func SetupWatcherTest() *WatcherTester {
	filepath, _ := filepath.Abs("./watcher_test.toml")
	trans, _ := toml.New(&types.Endpoint{Backend: "mock", Path: filepath})
	return &WatcherTester{watcher: NewWatcher(trans), mock: trans}
}

func TeardownWatcherTest(wt *WatcherTester) {

}

func TestCommandSubscribe(t *testing.T) {
	var wg sync.WaitGroup
	wt := SetupWatcherTest()
	w := wt.watcher

	var res *CommandObservable
	wg.Add(1)
	w.Subscribe("command", "hostname", func(co types.Observable) {
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
	w.Subscribe("file", "/tmp/test", func(fo types.Observable) {
		switch x := fo.(type) {
		case *FileObservable:
			defer wg.Done()
			res = x
		default:
		}
	})

	wg.Wait()
	content, err := ioutil.ReadAll(res.File)
	assert.Nil(t, err, "file content was returned without any error")
	assert.Equal(t, "test", string(content), "get the expected command output")

	TeardownWatcherTest(wt)
}

func TestFileChangeEvents(t *testing.T) {
	var wg sync.WaitGroup
	wt := SetupWatcherTest()
	w := wt.watcher
	// wait 500ms
	w.SleepDuration = time.Duration(2 * time.Millisecond)

	res := []string{}

	wg.Add(2)
	w.Subscribe("file", "/tmp/test", func(fo types.Observable) {
		switch x := fo.(type) {
		case *FileObservable:
			defer wg.Done()
			content, err := ioutil.ReadAll(x.File)
			if err == nil {
				res = append(res, string(content))
			}
		default:
		}
	})

	// wait a second to ensure the callback was called already
	time.AfterFunc(time.Duration(1*time.Millisecond), func() {
		// change file content
		mt := wt.mock.(*mock.Transport)
		mt.Files["/tmp/test"].Content = "newtest"
	})

	wg.Wait()

	assert.Equal(t, []string{"test", "newtest"}, res, "detect file change")

	TeardownWatcherTest(wt)
}
