package tar

import (
	"archive/tar"
	"io"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

// load files from a tar stream
func Load(m *mock.Transport, stream io.Reader) error {
	tr := tar.NewReader(stream)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("error reading tar stream")
			return err
		}

		content, err := ioutil.ReadAll(tr)
		if err != nil {
			log.Error().Str("file", h.Name).Err(err).Msg("mock> could not load file data")
		} else {
			log.Debug().Str("file", h.Name).Str("content", string(content)).Msg("mock> content")
		}
		fi := h.FileInfo()
		m.Fs.Files[h.Name] = &mock.MockFileData{
			Path:    h.Name,
			Content: string(content),
			StatData: mock.FileInfo{
				Mode:    fi.Mode(),
				IsDir:   fi.IsDir(),
				ModTime: fi.ModTime(),
			},
		}
		log.Debug().Str("file", h.Name).Msg("mock> add file to mock backend")
	}
	return nil
}
