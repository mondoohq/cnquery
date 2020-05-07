// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/motoros/events"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func (s *lumiFile) id() (string, error) {
	return s.Path()
}

func (s *lumiFile) GetContent(path string) (string, error) {
	_, ok := s.Cache.Load("content")
	if ok {
		return "", lumi.NotReadyError{}
	}

	log.Debug().Msg("[file]> listen to file " + path)

	watcher := s.Runtime.Motor.Watcher()
	// TODO: overwrite sleepduration for now
	watcher.(*events.Watcher).SleepDuration = 1 * time.Second

	err := watcher.Subscribe("file", path, func(o types.Observable) {
		content := ""
		f := o.(*events.FileObservable)
		if f.FileOp != events.Eonet {
			// file is available, therefore we can stream the content
			c, err := ioutil.ReadAll(f.File)
			if err == nil {
				content = string(c)
			}
		} else {
			log.Debug().Str("file", path).Msg("[file]> file does not exist")
		}

		old, ok := s.Cache.Load("content")
		if ok && old.Valid && old.Data.(string) == content {
			// nothing to be done
			return
		}

		log.Debug().Str("file", path).Msg("[file]> update content")
		s.Cache.Store("content", &lumi.CacheEntry{Data: content, Valid: true, Timestamp: time.Now().Unix()})

		err := s.Runtime.Observers.Trigger(s.LumiResource().FieldUID("content"))
		if err != nil {
			log.Error().Err(err).Msg("[file]> failed to trigger content")
		}
	})
	if err != nil {
		return "", err
	}
	s.Cache.Store("content", &lumi.CacheEntry{})

	s.Runtime.Observers.OnUnwatch(s.FieldUID("content"), func() {
		s.Cache.Delete("content")
		log.Debug().Msg("[file]> unwatch")
		watcher.Unsubscribe("file", path)
	})

	return "", lumi.NotReadyError{}
}

func (s *lumiFile) GetExists() (bool, error) {
	path, _ := s.Path()

	fs := s.Runtime.Motor.Transport.FS()
	afs := &afero.Afero{Fs: fs}
	return afs.Exists(path)
}

func (s *lumiFile) GetPermissions() (FilePermissions, error) {
	// TODO: this is a one-off right now, turn it into a watcher
	path, err := s.Path()
	if err != nil {
		return nil, err
	}

	fs := s.Runtime.Motor.Transport.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return nil, err
	}
	mode := stat.Mode()

	res, err := s.Runtime.CreateResource("file.permissions",
		"user_readable", mode&00400 != 0,
		"user_writeable", mode&00200 != 0,
		"user_executable", mode&00100 != 0,
		"group_readable", mode&00040 != 0,
		"group_writeable", mode&00020 != 0,
		"group_executable", mode&00010 != 0,
		"other_readable", mode&00004 != 0,
		"other_writeable", mode&00002 != 0,
		"other_executable", mode&00001 != 0,
		"suid", mode&04000 != 0,
		"sgid", mode&02000 != 0,
		"sticky", mode&01000 != 0,
	)
	if err != nil {
		return nil, err
	}

	return res.(FilePermissions), nil
}

func (l *lumiFilePermissions) id() (string, error) {
	res := []byte("---------")

	if i, _ := l.User_readable(); i {
		res[0] = 'r'
	}
	if i, _ := l.User_writeable(); i {
		res[1] = 'w'
	}
	if i, _ := l.User_executable(); i {
		res[2] = 'x'
	}

	if i, _ := l.Group_readable(); i {
		res[3] = 'r'
	}
	if i, _ := l.Group_writeable(); i {
		res[4] = 'w'
	}
	if i, _ := l.Group_executable(); i {
		res[5] = 'x'
	}

	if i, _ := l.Other_readable(); i {
		res[6] = 'r'
	}
	if i, _ := l.Other_writeable(); i {
		res[7] = 'w'
	}
	if i, _ := l.Other_executable(); i {
		res[8] = 'x'
	}

	return string(res), nil
}
