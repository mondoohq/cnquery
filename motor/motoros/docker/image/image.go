package image

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.mondoo.io/mondoo/motor/motoros/docker/cache"
	"go.mondoo.io/mondoo/motor/motoros/tar"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

//  provide a container image stream
func New(rc io.ReadCloser) (types.Transport, error) {
	// we cache the flattened image locally
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}

	err = cache.StreamToTmpFile(rc, f)
	if err != nil {
		return nil, err
	}

	// we return a pure tar image
	return NewFromFile(f.Name())
}

// no cache file required, since the file is cached locally already
func NewFromFile(filename string) (types.Transport, error) {
	return tar.NewWithClose(&types.Endpoint{Path: filename}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
}

func LoadFromRegistry(tag name.Tag) (v1.Image, io.ReadCloser, error) {
	auth, err := authn.DefaultKeychain.Resolve(tag.Registry)
	if err != nil {
		fmt.Printf("getting creds for %q: %v", tag, err)
		return nil, nil, err
	}

	// fmt.Printf("%v\n", tag)
	img, err := remote.Image(tag, remote.WithAuth(auth), remote.WithTransport(http.DefaultTransport))
	if err != nil {
		return nil, nil, err
	}
	return img, mutate.Extract(img), nil
}

type ShaReference struct {
	SHA string
}

func (r ShaReference) Name() string {
	return r.SHA
}

func (r ShaReference) String() string {
	return r.SHA
}

func (r ShaReference) Context() name.Repository {
	return name.Repository{}
}

func (r ShaReference) Identifier() string {
	return r.SHA
}

func (r ShaReference) Scope(scope string) string {
	return ""
}

func LoadFromDockerEngine(sha string) (v1.Image, io.ReadCloser, error) {
	img, err := daemon.Image(&ShaReference{SHA: strings.Replace(sha, "sha256:", "", -1)})
	if err != nil {
		return nil, nil, err
	}
	return img, mutate.Extract(img), nil
}

func ImageToTar(filename string, img v1.Image, baseName, imgName, tagName string) error {
	imgTag := fmt.Sprintf("%s/%s:%s", baseName, imgName, tagName)
	tag, err := name.NewTag(imgTag, name.WeakValidation)
	if err != nil {
		return errors.New(fmt.Sprintf("parsing tag %q: %v", imgTag, err))
	}
	return tarball.WriteToFile(filename, tag, img)
}
