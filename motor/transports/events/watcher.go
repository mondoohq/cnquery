package events

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
)

// Subscriptions is a map to store all watcher subscriptions
type Subscriptions struct{ sync.Map }

// Store a new subscription
func (c *Subscriptions) Store(k string, v *WatcherSubscription) {
	c.Map.Store(k, v)
}

// Load a subscription
func (c *Subscriptions) Load(k string) (*WatcherSubscription, bool) {
	res, ok := c.Map.Load(k)
	if !ok {
		return nil, ok
	}
	return res.(*WatcherSubscription), ok
}

func (c *Subscriptions) Delete(k string) {
	c.Map.Delete(k)
}

func (c *Subscriptions) Range(f func(string, *WatcherSubscription) bool) {
	c.Map.Range(func(key interface{}, value interface{}) bool {
		return f(key.(string), value.(*WatcherSubscription))
	})
}

type Watcher struct {
	transport     transports.Transport
	subscriptions *Subscriptions
	jm            *JobManager
	SleepDuration time.Duration
}

type WatcherSubscription struct {
	typ        string
	observable func(transports.Observable)
}

func NewWatcher(transport transports.Transport) *Watcher {
	w := &Watcher{transport: transport, subscriptions: &Subscriptions{}}
	w.transport = transport
	w.jm = NewJobManager(transport)
	w.SleepDuration = time.Duration(10 * time.Second)
	return w
}

// the internal unique id is a combination of the typ + id
func (w *Watcher) subscriberId(typ string, id string) string {
	sid := typ + id
	return sid
}

func (w *Watcher) Subscribe(typ string, id string, observable func(transports.Observable)) error {
	var job *Job

	log.Debug().Str("id", id).Str("typ", typ).Msg("motor.watcher> subscribe")
	sid := w.subscriberId(typ, id)

	// throw an error if the id is already registered
	_, ok := w.subscriptions.Load(sid)
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
			Callback: []func(o transports.Observable){
				func(o transports.Observable) {
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
			Callback: []func(o transports.Observable){
				func(o transports.Observable) {
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
	w.subscriptions.Store(sid, &WatcherSubscription{
		typ:        typ,
		observable: observable,
	})

	return nil
}

func (w *Watcher) Unsubscribe(typ string, id string) error {
	log.Debug().Str("id", id).Str("typ", typ).Msg("motor.watcher> unsubscribe")
	// gather internal id
	sid := w.subscriberId(typ, id)
	return w.unsubscribe(sid)
}

func (w *Watcher) unsubscribe(sid string) error {
	// stop jobs in flight
	w.jm.Delete(sid)

	// remove the subscription and un-register the jobs
	w.subscriptions.Delete(sid)
	return nil
}

func (w *Watcher) TearDown() error {
	log.Debug().Msg("motor.watcher> teardown")
	// remove all subscriptions
	w.subscriptions.Range(func(k string, v *WatcherSubscription) bool {
		if err := w.unsubscribe(k); err != nil {
			log.Warn().Str("sub", k).Err(err).Msg("motor.watch> teardown unscribe failed")
		}
		return true
	})

	// tear down job manager, all subscriptions should be stopped already
	w.jm.TearDown()

	return nil
}
