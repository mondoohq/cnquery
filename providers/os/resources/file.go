// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"os"
	"path"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// newFile creates a new file resource
func newFile(runtime *plugin.Runtime, path string) (*mqlFile, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}
	file := f.(*mqlFile)
	return file, nil
}

type mqlFileInternal struct {
	statInfo *shared.FileInfoDetails
}

func (s *mqlFile) id() (string, error) {
	return s.Path.Data, nil
}

func (s *mqlFile) content(path string, exists bool) (string, error) {
	if !exists {
		return "", resources.NotFoundError{Resource: "file", ID: path}
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}
	res, err := afs.ReadFile(path)
	return string(res), err
}

func (s *mqlFile) cacheStatFields(stat shared.FileInfoDetails) error {
	mode := stat.Mode.UnixMode()
	res, err := CreateResource(s.MqlRuntime, "file.permissions", map[string]*llx.RawData{
		"mode":             llx.IntData(int64(uint32(mode) & 0o7777)),
		"user_readable":    llx.BoolData(stat.Mode.UserReadable()),
		"user_writeable":   llx.BoolData(stat.Mode.UserWriteable()),
		"user_executable":  llx.BoolData(stat.Mode.UserExecutable()),
		"group_readable":   llx.BoolData(stat.Mode.GroupReadable()),
		"group_writeable":  llx.BoolData(stat.Mode.GroupWriteable()),
		"group_executable": llx.BoolData(stat.Mode.GroupExecutable()),
		"other_readable":   llx.BoolData(stat.Mode.OtherReadable()),
		"other_writeable":  llx.BoolData(stat.Mode.OtherWriteable()),
		"other_executable": llx.BoolData(stat.Mode.OtherExecutable()),
		"suid":             llx.BoolData(stat.Mode.Suid()),
		"sgid":             llx.BoolData(stat.Mode.Sgid()),
		"sticky":           llx.BoolData(stat.Mode.Sticky()),
		"isDirectory":      llx.BoolData(stat.Mode.IsDir()),
		"isFile":           llx.BoolData(stat.Mode.IsRegular()),
		"isSymlink":        llx.BoolData(stat.Mode.FileMode&os.ModeSymlink != 0),
	})
	if err != nil {
		return err
	}

	s.Exists = plugin.TValue[bool]{
		Data:  true,
		State: plugin.StateIsSet,
	}
	s.Permissions = plugin.TValue[*mqlFilePermissions]{
		Data:  res.(*mqlFilePermissions),
		State: plugin.StateIsSet,
	}
	s.Size = plugin.TValue[int64]{
		Data:  stat.Size,
		State: plugin.StateIsSet,
	}

	statCopy := stat
	s.statInfo = &statCopy

	return nil
}

func (s *mqlFile) loadStatFields(path string) (*shared.FileInfoDetails, bool, error) {
	if s.Exists.IsSet() {
		if !s.Exists.Data {
			return nil, false, s.Exists.Error
		}
		if s.statInfo != nil {
			return s.statInfo, true, nil
		}
		if s.Permissions.IsSet() && s.Size.IsSet() {
			return nil, true, nil
		}
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	stat, err := conn.FileInfo(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.Exists = plugin.TValue[bool]{
				Data:  false,
				State: plugin.StateIsSet,
			}
			return nil, false, nil
		}
		return nil, false, err
	}
	if err := s.cacheStatFields(stat); err != nil {
		return nil, false, err
	}

	return s.statInfo, true, nil
}

func (s *mqlFile) cacheOwnership(stat shared.FileInfoDetails) error {
	raw, err := CreateResource(s.MqlRuntime, "users", nil)
	if err != nil {
		return errors.New("cannot get users info for file: " + err.Error())
	}
	users := raw.(*mqlUsers)

	user, err := users.findID(stat.Uid)
	if err != nil {
		return err
	}
	s.User = plugin.TValue[*mqlUser]{
		Data:  user,
		State: plugin.StateIsSet,
	}

	raw, err = CreateResource(s.MqlRuntime, "groups", nil)
	if err != nil {
		return errors.New("cannot get groups info for file: " + err.Error())
	}
	groups := raw.(*mqlGroups)

	group, err := groups.findID(stat.Gid)
	if err != nil {
		return err
	}
	s.Group = plugin.TValue[*mqlGroup]{
		Data:  group,
		State: plugin.StateIsSet,
	}

	return nil
}

func (s *mqlFile) loadOwnership(path string) error {
	stat, exists, err := s.loadStatFields(path)
	if err != nil {
		return err
	}
	if !exists {
		return os.ErrNotExist
	}
	if s.User.IsSet() && s.Group.IsSet() {
		return nil
	}
	if stat == nil {
		conn := s.MqlRuntime.Connection.(shared.Connection)
		statValue, err := conn.FileInfo(path)
		if err != nil {
			return err
		}
		stat = &statValue
	}

	return s.cacheOwnership(*stat)
}

func (s *mqlFile) stat() error {
	return s.loadOwnership(s.Path.Data)
}

func (s *mqlFile) size(path string) (int64, error) {
	_, exists, err := s.loadStatFields(path)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, os.ErrNotExist
	}
	return 0, nil
}

func (s *mqlFile) permissions(path string) (*mqlFilePermissions, error) {
	_, exists, err := s.loadStatFields(path)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, os.ErrNotExist
	}
	return nil, nil
}

func (s *mqlFile) user() (*mqlUser, error) {
	return nil, s.loadOwnership(s.Path.Data)
}

func (s *mqlFile) group() (*mqlGroup, error) {
	return nil, s.loadOwnership(s.Path.Data)
}

func (s *mqlFile) empty(path string) (bool, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}
	return afs.IsEmpty(path)
}

func (s *mqlFile) basename(fullPath string) (string, error) {
	return path.Base(fullPath), nil
}

func (s *mqlFile) dirname(fullPath string) (string, error) {
	return path.Dir(fullPath), nil
}

func (s *mqlFile) exists(path string) (bool, error) {
	_, exists, err := s.loadStatFields(path)
	return exists, err
}

func (l *mqlFilePermissions) id() (string, error) {
	res := []byte("----------")

	if l.IsDirectory.Data {
		res[0] = 'd'
	} else if l.IsSymlink.Data {
		res[0] = 'l'
	}

	if l.User_readable.Data {
		res[1] = 'r'
	}
	if l.User_writeable.Data {
		res[2] = 'w'
	}
	if l.User_executable.Data {
		res[3] = 'x'
		if l.Suid.Data {
			res[3] = 's'
		}
	} else {
		if l.Suid.Data {
			res[3] = 'S'
		}
	}

	if l.Group_readable.Data {
		res[4] = 'r'
	}
	if l.Group_writeable.Data {
		res[5] = 'w'
	}
	if l.Group_executable.Data {
		res[6] = 'x'
		if l.Sgid.Data {
			res[6] = 's'
		}
	} else {
		if l.Sgid.Data {
			res[6] = 'S'
		}
	}

	if l.Other_readable.Data {
		res[7] = 'r'
	}
	if l.Other_writeable.Data {
		res[8] = 'w'
	}
	if l.Other_executable.Data {
		res[9] = 'x'
		if l.Sticky.Data {
			res[9] = 't'
		}
	} else {
		if l.Sticky.Data {
			res[9] = 'T'
		}
	}

	return string(res), nil
}

func (l *mqlFilePermissions) string() (string, error) {
	return l.__id, nil
}

func (r *mqlFileContext) id() (string, error) {
	if r.File.Data == nil {
		return "", errors.New("need file to exist for file.context ID")
	}

	fileID, err := r.File.Data.id()
	if err != nil {
		return "", err
	}

	rng := r.Range.Data.String()
	return fileID + ":" + rng, nil
}

func (r *mqlFileContext) content(file *mqlFile, rnge llx.Range) (string, error) {
	if file == nil {
		return "", errors.New("no file information for file.context")
	}

	fileContent := file.GetContent()
	if fileContent.Error != nil {
		return "", fileContent.Error
	}

	return rnge.ExtractString(fileContent.Data, llx.DefaultExtractConfig), nil
}
