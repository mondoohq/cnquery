package image

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/runtime"
	"go.mondoo.io/mondoo/motor/transports/docker/cache"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"go.mondoo.io/mondoo/nexus/assets"
)

type DockerImageTransport struct {
	tar.Transport
}

func (t *DockerImageTransport) Kind() assets.Kind {
	return assets.Kind_KIND_CONTAINER_IMAGE
}

func (t *DockerImageTransport) Runtime() string {
	return runtime.RUNTIME_DOCKER_IMAGE
}

func newWithClose(endpoint *motorapi.Endpoint, close func()) (*DockerImageTransport, error) {
	t := &DockerImageTransport{
		Transport: tar.Transport{
			Fs:      tar.NewFs(endpoint.Path),
			CloseFN: close,
		},
	}

	var err error
	if endpoint != nil && len(endpoint.Path) > 0 {
		err := t.LoadFile(endpoint.Path)
		if err != nil {
			log.Error().Err(err).Str("tar", endpoint.Path).Msg("tar> could not load tar file")
			return nil, err
		}
	}
	return t, err
}

//  provide a container image stream
func New(rc io.ReadCloser) (*DockerImageTransport, error) {
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
	filename := f.Name()

	return newWithClose(&motorapi.Endpoint{Path: filename}, func() {
		// remove temporary file on stream close
		os.Remove(filename)
	})
}

// Option is a functional option
// see https://www.sohamkamani.com/golang/options-pattern/
type Option func(*options) error

type options struct {
	insecure bool
	auth     authn.Authenticator
}

func WithInsecure(insecure bool) Option {
	return func(o *options) error {
		o.insecure = insecure
		return nil
	}
}

func WithAuthenticator(auth authn.Authenticator) Option {
	return func(o *options) error {
		o.auth = auth
		return nil
	}
}

func LoadFromRegistry(ref name.Reference, opts ...Option) (v1.Image, io.ReadCloser, error) {
	o := &options{
		insecure: false,
	}

	for _, option := range opts {
		if err := option(o); err != nil {
			return nil, nil, err
		}
	}

	if o.auth == nil {
		auth, err := authn.DefaultKeychain.Resolve(ref.Context())
		if err != nil {
			fmt.Printf("getting creds for %q: %v", ref, err)
			return nil, nil, err
		}
		o.auth = auth
	}

	// mimic http.DefaultTransport
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if o.insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	img, err := remote.Image(ref, remote.WithAuth(o.auth), remote.WithTransport(tr))
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
