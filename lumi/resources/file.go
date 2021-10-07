// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/events"
)

func (s *lumiFile) id() (string, error) {
	return s.Path()
}

func (s *lumiFile) GetContent(path string, exists bool) (string, error) {
	if !exists {
		log.Debug().Str("file", path).Msg("[file]> file does not exist")

		// store the result in cache as we don't expect the file to improve
		// unless it starts existing
		resErr := fmt.Errorf("file %w: '%s' does not exist", lumi.NotFound, path)
		s.Cache.Store("content", &lumi.CacheEntry{
			Data:      "",
			Valid:     true,
			Error:     resErr,
			Timestamp: time.Now().Unix(),
		})

		// returning the error will prevent a cache overwrite
		return "", resErr
	}

	_, ok := s.Cache.Load("content")
	if ok {
		return "", lumi.NotReadyError{}
	}

	log.Debug().Msg("[file]> listen to file " + path)

	watcher := s.Runtime.Motor.Watcher()

	err := watcher.Subscribe("file", path, func(o transports.Observable) {
		log.Debug().Str("file", path).Msg("[file]> got observable")
		content := ""
		f := o.(*events.FileObservable)
		if f.FileOp != events.Enoent && f.FileOp != events.Error {
			// file is available, therefore we can stream the content
			c, err := ioutil.ReadAll(f.File)
			if err == nil {
				content = string(c)
			}

			old, ok := s.Cache.Load("content")
			if ok && old.Valid && old.Data.(string) == content {
				// nothing to be done
				return
			}

			log.Debug().Str("file", path).Msg("[file]> update content")
			s.Cache.Store("content", &lumi.CacheEntry{
				Data:      content,
				Valid:     true,
				Timestamp: time.Now().Unix(),
			})

		} else {

			log.Debug().Str("file", path).Msg("[file]> file does not exist")
			resErr := errors.New("file '" + path + "' does not exist: " + f.Error.Error())
			s.Cache.Store("content", &lumi.CacheEntry{
				Data:      "",
				Valid:     true,
				Timestamp: time.Now().Unix(),
				Error:     resErr,
			})
		}

		err := s.Runtime.Observers.Trigger(s.LumiResource().FieldUID("content"))
		if err != nil {
			log.Error().Err(err).Msg("[file]> failed to trigger content")
		}
	})

	// make sure the watcher is established before doing any these remaining steps
	if err != nil {
		return "", err
	}

	s.Runtime.Observers.OnUnwatch(s.FieldUID("content"), func() {
		s.Cache.Delete("content")
		log.Debug().Msg("[file]> unwatch")
		watcher.Unsubscribe("file", path)
	})

	return "", lumi.NotReadyError{}
}

func (s *lumiFile) GetExists() (bool, error) {
	// TODO: we need to tell motor to watch this for us
	path, _ := s.Path()

	fs := s.Runtime.Motor.Transport.FS()
	afs := &afero.Afero{Fs: fs}
	return afs.Exists(path)
}

func (s *lumiFile) GetBasename(fullPath string) (string, error) {
	return path.Base(fullPath), nil
}

func (s *lumiFile) GetDirname(fullPath string) (string, error) {
	return path.Dir(fullPath), nil
}

func (s *lumiFile) GetPermissions() (FilePermissions, error) {
	perm, size, err := s.stat()
	// cache the other computed fields
	s.Cache.Store("size", &lumi.CacheEntry{Data: size, Valid: true, Timestamp: time.Now().Unix()})
	return perm, err
}

func (s *lumiFile) GetSize() (int64, error) {
	perm, size, err := s.stat()
	// cache the other computed fields
	s.Cache.Store("permissions", &lumi.CacheEntry{Data: perm, Valid: true, Timestamp: time.Now().Unix()})
	return size, err
}

func (s *lumiFile) GetUser() (interface{}, error) {
	path, err := s.Path()
	if err != nil {
		return nil, err
	}

	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	// special handling for windows
	// NOTE: on windows an owner can also be a group, therefore we need to be very careful in implementing it here
	// Probably we are better of in implementing a windows.file resource that deals with specific behavior on windows
	// see https://devblogs.microsoft.com/scripting/hey-scripting-guy-how-can-i-use-windows-powershell-to-determine-the-owner-of-a-file/
	if platform.IsFamily("windows") {
		return nil, errors.New("user is not supported on windows")
	}

	// handle unix
	fi, err := s.Runtime.Motor.Transport.FileInfo(path)
	if err != nil {
		return nil, err
	}

	// handle case where we have no gid available
	// TODO: do we have a better approach than checking for -1?
	if fi.Uid < 0 {
		return nil, nil
	}

	lumiUser, err := s.Runtime.CreateResource("user",
		"uid", fi.Uid,
	)
	if err != nil {
		return nil, err
	}
	return lumiUser.(User), nil
}

func (s *lumiFile) GetGroup() (interface{}, error) {
	path, err := s.Path()
	if err != nil {
		return nil, err
	}

	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if platform.IsFamily("windows") {
		return nil, errors.New("group is not supported on windows")
	}

	fi, err := s.Runtime.Motor.Transport.FileInfo(path)
	if err != nil {
		return nil, err
	}

	// handle case where we have no gid available
	// TODO: do we have a better approach than checking for -1?
	if fi.Gid < 0 {
		return nil, nil
	}

	lumiUser, err := s.Runtime.CreateResource("group",
		"id", strconv.FormatInt(fi.Gid, 10),
		"gid", fi.Gid,
	)
	if err != nil {
		return nil, err
	}
	return lumiUser.(Group), nil
}

func (s *lumiFile) stat() (FilePermissions, int64, error) {
	// TODO: this is a one-off right now, turn it into a watcher
	path, err := s.Path()
	if err != nil {
		return nil, 0, err
	}

	fi, err := s.Runtime.Motor.Transport.FileInfo(path)
	if err != nil {
		return nil, 0, err
	}

	mode := fi.Mode.FileMode

	permRaw, err := s.Runtime.CreateResource("file.permissions",
		"mode", int64(uint32(mode)&07777),
		"user_readable", fi.Mode.UserReadable(),
		"user_writeable", fi.Mode.UserWriteable(),
		"user_executable", fi.Mode.UserExecutable(),
		"group_readable", fi.Mode.GroupReadable(),
		"group_writeable", fi.Mode.GroupWriteable(),
		"group_executable", fi.Mode.GroupExecutable(),
		"other_readable", fi.Mode.OtherReadable(),
		"other_writeable", fi.Mode.OtherWriteable(),
		"other_executable", fi.Mode.OtherExecutable(),
		"suid", fi.Mode.Suid(),
		"sgid", fi.Mode.Sgid(),
		"sticky", fi.Mode.Sticky(),
		"isDirectory", fi.Mode.IsDir(),
		"isFile", fi.Mode.IsRegular(),
		"isSymlink", fi.Mode.FileMode&os.ModeSymlink == 0,
	)
	if err != nil {
		return nil, 0, err
	}

	perm := permRaw.(FilePermissions)
	size := fi.Size

	return perm, size, nil
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
