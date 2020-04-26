package kernel

import (
	"archive/tar"
	"bufio"
	"io"
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"
)

func ParseSysctl(r io.Reader, sep string) (map[string]string, error) {
	kernelParameters := map[string]string{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		keyval := strings.Split(line, sep)

		if len(keyval) == 2 {
			kernelParameters[strings.TrimSpace(keyval[0])] = strings.TrimSpace(keyval[0])
		} else {
			log.Debug().Str("line", line).Msg("cannot parse sysctl line")
			continue
		}
	}

	return kernelParameters, nil
}

func ParseLinuxSysctlProc(sysctlRootPath string, reader io.Reader) (map[string]string, error) {
	kernelParameters := map[string]string{}

	// parse kernel parameters from tar stream
	tr := tar.NewReader(reader)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !h.FileInfo().IsDir() {
			content, _ := ioutil.ReadAll(tr)
			// remove leading sysctl path
			k := strings.Replace(h.Name, sysctlRootPath, "", -1)
			k = strings.Replace(k, "/", ".", -1)
			kernelParameters[k] = strings.TrimSpace(string(content))
		}
	}

	return kernelParameters, nil
}
