package os

import (
	"crypto/x509"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/core/certificates"
)

func (s *mqlOsRootCertificates) id() (string, error) {
	return "osrootcertificates", nil
}

func (s *mqlOsRootCertificates) init(args *resources.Args) (*resources.Args, OsRootCertificates, error) {
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
	mqlFiles := []interface{}{}
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
			mqlFiles = append(mqlFiles, f.(core.File))
			break
		}
	}

	(*args)["files"] = mqlFiles
	return args, nil, nil
}

func (s *mqlOsRootCertificates) GetFiles() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (s *mqlOsRootCertificates) GetContent(files []interface{}) ([]interface{}, error) {
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

func (s *mqlOsRootCertificates) GetList(content []interface{}) ([]interface{}, error) {
	certificateList := []*x509.Certificate{}
	for i := range content {
		certs, err := certificates.ParseCertFromPEM(strings.NewReader(content[i].(string)))
		if err != nil {
			return nil, err
		}
		certificateList = append(certificateList, certs...)
	}
	return core.CertificatesToMqlCertificates(s.MotorRuntime, certificateList)
}
