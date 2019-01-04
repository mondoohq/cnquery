package events

import (
	"errors"
	"time"

	"go.mondoo.io/mondoo/motor/types"
)

type Watcher struct {
	transport     types.Transport
	subscriptions map[string]*WatcherSubscription
	jm            *JobManager
	SleepDuration time.Duration
}

type WatcherSubscription struct {
	typ        string
	observable func(types.Observable)
}

func NewWatcher(transport types.Transport) *Watcher {
	w := &Watcher{transport: transport}
	w.transport = transport
	w.subscriptions = make(map[string]*WatcherSubscription)
	w.jm = NewJobManager(transport)
	w.SleepDuration = time.Duration(10 * time.Second)
	return w
}

// the internal unique id is a combination of the typ + id
func (w *Watcher) subscriberId(typ string, id string) string {
	sid := typ + id
	return sid
}

func (w *Watcher) Subscribe(typ string, id string, observable func(types.Observable)) error {
	var job *Job

	sid := w.subscriberId(typ, id)

	// throw an error if the id is already registered
	_, ok := w.subscriptions[sid]
	if ok {
		return errors.New("resource " + typ + " with " + id + " is already registered")
	}

	// register the right job to gather the information
	switch typ {
	case "file":
		job = &Job{
			ID:           sid,
			ScheduledFor: time.Now(),
			Interval:     w.SleepDuration,
			Runnable:     NewFileRunnable(id),
			Repeat:       -1,
			Callback: []func(o types.Observable){
				func(o types.Observable) {
					observable(o)
				},
			},
		}
	case "command":
		job = &Job{
			ID:           sid,
			ScheduledFor: time.Now(),
			Interval:     w.SleepDuration,
			Runnable:     NewCommandRunnable(id),
			Repeat:       -1,
			Callback: []func(o types.Observable){
				func(o types.Observable) {
					observable(o)
				},
			},
		}
	default:
		return errors.New("unknown typ " + typ)
	}

	jobid, err := w.jm.Schedule(job)
	if err != nil {
		return err
	}

	// verify that the job id is our given id
	if jobid != sid {
		w.jm.Delete(jobid)
		return errors.New("something is wrong, the job ids are not identical")
	}

	// store the subscription
	w.subscriptions[sid] = &WatcherSubscription{
		typ:        typ,
		observable: observable,
	}

	return nil
}

func (w *Watcher) Unsubscribe(typ string, id string) error {
	// gather internal id
	sid := w.subscriberId(typ, id)
	return w.unsubscribe(sid)
}

func (w *Watcher) unsubscribe(sid string) error {
	// stop jobs in flight
	err := w.jm.Delete(sid)
	if err != nil {
		return err
	}

	// remove the subscription and un-register the jobs
	delete(w.subscriptions, sid)
	return nil
}

func (w *Watcher) TearDown() error {
	// remove all subscriptions
	for sid, _ := range w.subscriptions {
		w.unsubscribe(sid)
	}

	// tear down job manager, all subscriptions should be stopped already
	w.jm.TearDown()

	return nil
}
