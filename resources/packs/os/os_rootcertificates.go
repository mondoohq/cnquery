package os

import (
	"crypto/x509"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/core/certificates"
)

func (s *lumiOsRootCertificates) id() (string, error) {
	return "osrootcertificates", nil
}

func (s *lumiOsRootCertificates) init(args *lumi.Args) (*lumi.Args, OsRootCertificates, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, nil, err
	}

	pi, err := s.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, nil, err
	}

	var files []string
	if pi.IsFamily(platform.FAMILY_LINUX) {
		files = certificates.LinuxCertFiles
	} else if pi.IsFamily(platform.FAMILY_BSD) {
		files = certificates.BsdCertFiles
	} else {
		return nil, nil, errors.New("root certificates are not unsupported on this platform: " + pi.Name + " " + pi.Version)
	}

	// search the first file that exists, it mimics the behavior go is doing
	lumiFiles := []interface{}{}
	for i := range files {
		log.Trace().Str("path", files[i]).Msg("os.rootcertificates> check root certificate path")
		fileInfo, err := osProvider.FS().Stat(files[i])
		if err != nil {
			log.Trace().Err(err).Str("path", files[i]).Msg("os.rootcertificates> file does not exist")
			continue
		}
		log.Debug().Str("path", files[i]).Msg("os.rootcertificates> found root certificate bundle path")
		if !fileInfo.IsDir() {
			f, err := s.MotorRuntime.CreateResource("file", "path", files[i])
			if err != nil {
				return nil, nil, err
			}
			lumiFiles = append(lumiFiles, f.(core.File))
			break
		}
	}

	(*args)["files"] = lumiFiles
	return args, nil, nil
}

func (s *lumiOsRootCertificates) GetFiles() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s *lumiOsRootCertificates) GetContent(files []interface{}) ([]interface{}, error) {
	contents := []interface{}{}

	for i := range files {
		file := files[i].(core.File)

		// TODO: this can be heavily improved once we do it right, since this is constantly
		// re-registered as the file changes
		err := s.MotorRuntime.WatchAndCompute(file, "content", s, "content")
		if err != nil {
			return nil, err
		}

		content, err := file.Content()
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}

	return contents, nil
}

func (s *lumiOsRootCertificates) GetList(content []interface{}) ([]interface{}, error) {
	certificateList := []*x509.Certificate{}
	for i := range content {
		certs, err := certificates.ParseCertFromPEM(strings.NewReader(content[i].(string)))
		if err != nil {
			return nil, err
		}
		certificateList = append(certificateList, certs...)
	}
	return core.CertificatesToLumiCertificates(s.MotorRuntime, certificateList)
}
