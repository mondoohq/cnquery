package procfs

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"strings"
)

func ParseLinuxSysctl(sysctlRootPath string, reader io.Reader) (map[string]string, error) {
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
