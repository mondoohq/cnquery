package tar

import (
	"archive/tar"
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func NewFs(source string) *FS {
	return &FS{
		Source:  source,
		FileMap: make(map[string]*tar.Header),
	}
}

type FS struct {
	Source  string
	FileMap map[string]*tar.Header
}

func (fs *FS) Name() string {
	return "tarfs"
}

func (fs *FS) Create(name string) (afero.File, error) {
	return nil, errors.New("create not implemented")
}

func (fs *FS) Mkdir(name string, perm os.FileMode) error {
	return errors.New("mkdir not implemented")
}

func (fs *FS) MkdirAll(path string, perm os.FileMode) error {
	return errors.New("mkdirall not implemented")
}

func (fs *FS) Open(path string) (afero.File, error) {
	h, ok := fs.FileMap[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	if h.Typeflag == tar.TypeSymlink {
		resolvedPath := fs.resolveSymlink(h)
		log.Debug().Str("path", path).Str("resolved", Abs(resolvedPath)).Msg("file is a symlink, resolved it")
		h, ok = fs.FileMap[Abs(resolvedPath)]
		if !ok {
			return nil, os.ErrNotExist
		}
	}

	reader, err := fs.open(h)
	if err != nil {
		return nil, err
	}

	return &File{
		path:   path,
		header: h,
		Fs:     fs,
		reader: reader,
	}, nil
}

func (fs *FS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, errors.New("openfile not implemented")
}

func (fs *FS) Remove(name string) error {
	return errors.New("remove not implemented")
}

func (fs *FS) RemoveAll(path string) error {
	return errors.New("removeall not implemented")
}

func (fs *FS) Rename(oldname, newname string) error {
	return errors.New("rename not implemented")
}

func (fs *FS) Stat(name string) (os.FileInfo, error) {
	h, ok := fs.FileMap[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return fs.stat(h)
}

func (fs *FS) Chmod(name string, mode os.FileMode) error {
	return errors.New("chmod not implemented")
}

func (fs *FS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return errors.New("chtimes not implemented")
}

func (fs *FS) Chown(name string, uid, gid int) error {
	return errors.New("chown not implemented")
}

func (fs *FS) stat(header *tar.Header) (os.FileInfo, error) {
	statHeader := header
	if header.Typeflag == tar.TypeSymlink {
		path := fs.resolveSymlink(header)
		h, ok := fs.FileMap[Abs(path)]
		if !ok {
			return nil, errors.New("could not find " + path)
		}
		statHeader = h
	}
	return statHeader.FileInfo(), nil
}

// resolve symlink file
func (fs *FS) resolveSymlink(header *tar.Header) string {
	dest := header.Name
	link := header.Linkname

	var path string
	if filepath.IsAbs(link) {
		var err error
		// we need to remove the root / then
		path, err = filepath.Rel("/", link)
		if err != nil {
			log.Error().Str("link", link).Msg("could not determine the relative root path")
		}

	} else {
		path = Clean(join(dest, "..", link))
	}
	log.Debug().Str("link", link).Str("file", dest).Str("path", path).Msg("tar> is symlink")
	return path
}

func (fs *FS) open(header *tar.Header) (*bufio.Reader, error) {
	log.Debug().Str("file", header.Name).Msg("tar> load file content")

	// open tar file
	f, err := os.Open(fs.Source)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	path := header.Name
	if header.Typeflag == tar.TypeSymlink {
		path = fs.resolveSymlink(header)
	}

	// extract file from tar stream
	reader, err := fsutil.ExtractFileFromTarStream(path, f)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (fs *FS) tar(path string, header *tar.Header) (io.ReadCloser, error) {
	fReader, err := fs.open(header)
	if err != nil {
		return nil, err
	}

	// create a pipe
	tarReader, tarWriter := io.Pipe()

	// get file info, header my just include symlink fileinfo
	fi, err := fs.stat(header)
	if err != nil {
		return nil, err
	}

	// convert raw stream to tar stream
	go fsutil.StreamFileAsTar(header.Name, fi, ioutil.NopCloser(fReader), tarWriter)

	// return the reader
	return tarReader, nil
}

// searches for files and returns the file info
// regex can be nil
func (fs *FS) Find(from string, r *regexp.Regexp, typ string) ([]string, error) {
	list := []string{}
	for k := range fs.FileMap {
		p := strings.HasPrefix(k, from)
		m := true
		if r != nil {
			m = r.MatchString(k)
		}
		log.Trace().Str("path", k).Str("from", from).Str("prefix", from).Bool("prefix", p).Bool("m", m).Msg("check if matches")
		if p && m {
			entry := fs.FileMap[k]
			if (typ == "directory" && entry.Typeflag == tar.TypeDir) || (typ == "file" && entry.Typeflag == tar.TypeReg) {
				list = append(list, k)
				log.Debug().Msg("matches")
				continue
			}
		}
	}
	return list, nil
}
