package image

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

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

func GetImageDescriptor(ref name.Reference, opts ...Option) (*remote.Descriptor, error) {
	o := &options{
		insecure: false,
	}

	for _, option := range opts {
		if err := option(o); err != nil {
			return nil, err
		}
	}

	if o.auth == nil {
		auth, err := authn.DefaultKeychain.Resolve(ref.Context())
		if err != nil {
			fmt.Printf("getting creds for %q: %v", ref, err)
			return nil, err
		}
		o.auth = auth
	}

	return remote.Get(ref, remote.WithAuth(o.auth))
}

func LoadImageFromRegistry(ref name.Reference, opts ...Option) (v1.Image, io.ReadCloser, error) {
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
