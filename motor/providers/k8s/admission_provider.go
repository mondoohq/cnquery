package k8s

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/fsutil"
	admission "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/version"
)

func newAdmissionProvider(data string, selectedResourceID string) (KubernetesProvider, error) {
	t := &admissionProvider{}
	admission, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode admission review")
		return nil, err
	}

	t.manifestParser, err = newManifestParser(admission, "", selectedResourceID)
	if err != nil {
		return nil, err
	}

	res, err := t.AdmissionReviews()

	for _, r := range res {
		// For each admission we want to also parse the object as an individual asset so we
		// can show the admission review and the resource together in the CI/CD view.
		objs, err := resources.ResourcesFromManifest(bytes.NewReader(r.Request.Object.Raw))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse object from admission review")
		}
		t.objects = append(t.objects, objs...)
	}

	t.selectedResourceID = selectedResourceID
	return t, nil
}

type admissionProvider struct {
	manifestParser
	selectedResourceID string
}

func (t *admissionProvider) RunCommand(command string) (*os_provider.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *admissionProvider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	return os_provider.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *admissionProvider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *admissionProvider) Close() {}

func (t *admissionProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (t *admissionProvider) PlatformInfo() *platform.Platform {
	platformData := getPlatformInfo(t.selectedResourceID, t.Runtime())
	if platformData != nil {
		return platformData
	}

	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes Admission",
		Kind:    providers.Kind_KIND_CODE,
		Runtime: t.Runtime(),
	}
}

func (t *admissionProvider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (t *admissionProvider) Runtime() string {
	return providers.RUNTIME_KUBERNETES_ADMISSION
}

func (t *admissionProvider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *admissionProvider) ServerVersion() *version.Info {
	return nil
}

func (t *admissionProvider) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.manifestParser.SupportedResourceTypes()
}

func (t *admissionProvider) ID() (string, error) {
	reviews, err := t.AdmissionReviews()
	if err != nil {
		return "", err
	}

	return string(reviews[0].Request.UID), nil
}

func (t *admissionProvider) PlatformIdentifier() (string, error) {
	if t.selectedResourceID != "" {
		return t.selectedResourceID, nil
	}

	uid, err := t.ID()
	if err != nil {
		return "", err
	}

	return NewPlatformID(uid), nil
}

func (t *admissionProvider) Identifier() (string, error) {
	return t.PlatformIdentifier()
}

func (t *admissionProvider) Name() (string, error) {
	reviews, err := t.AdmissionReviews()
	if err != nil {
		return "", err
	}
	return "K8S Admission review " + reviews[0].Request.Name, nil
}

func (t *admissionProvider) AdmissionReviews() ([]admission.AdmissionReview, error) {
	res, err := t.Resources("admissionreview.v1.admission", "", "")
	if err != nil {
		return nil, err
	}

	if len(res.Resources) < 1 {
		return nil, fmt.Errorf("no admission review found")
	}

	reviews := make([]admission.AdmissionReview, 0, len(res.Resources))
	for _, r := range res.Resources {
		reviews = append(reviews, *r.(*admission.AdmissionReview))
	}
	return reviews, nil
}

// func loadManifestFile(manifestFile string) ([]byte, error) {
// 	var input io.Reader

// 	// if content is piped
// 	if manifestFile == "-" {
// 		input = os.Stdin
// 	} else {
// 		// return all resources from manifest
// 		filenames := []string{}

// 		fi, err := os.Stat(manifestFile)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if fi.IsDir() {
// 			// NOTE: we are not using filepath.WalkDir since we do not net recursive walking
// 			files, err := ioutil.ReadDir(manifestFile)
// 			if err != nil {
// 				return nil, err
// 			}
// 			for i := range files {
// 				f := files[i]
// 				if f.IsDir() {
// 					continue
// 				}
// 				filename := path.Join(manifestFile, f.Name())

// 				// only load yaml files for now
// 				ext := filepath.Ext(filename)
// 				if ext == ".yaml" || ext == ".yml" {
// 					log.Debug().Str("file", filename).Msg("add file to manifest loading")
// 					filenames = append(filenames, filename)
// 				} else {
// 					log.Debug().Str("file", filename).Msg("ignore file")
// 				}

// 			}

// 		} else {
// 			filenames = append(filenames, manifestFile)
// 		}

// 		input, err = resources.MergeManifestFiles(filenames)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return ioutil.ReadAll(input)
// }
