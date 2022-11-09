package msgraphclient

import (
	nethttp "net/http"

	"github.com/cockroachdb/errors"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	absser "github.com/microsoft/kiota-abstractions-go/serialization"
	core "github.com/microsoftgraph/msgraph-sdk-go-core"
)

// NOTE:
// This code is almost verbatim from https://github.com/microsoftgraph/msgraph-beta-sdk-go
// We avoid using that package directly because it brings in a bunch of things we don't
// use

var clientOptions = core.GraphClientOptions{
	GraphServiceVersion:        "beta",
	GraphServiceLibraryVersion: "0.27.0",
}

const DefaultMSGraphScope = "https://graph.microsoft.com/.default"

var DefaultMSGraphScopes = []string{DefaultMSGraphScope}

// GetDefaultClientOptions returns the default client options used by the GraphRequestAdapterBase and the middleware.
func GetDefaultClientOptions() core.GraphClientOptions {
	return clientOptions
}

// GraphRequestAdapter is the core service used by GraphServiceClient to make requests to Microsoft Graph.
type GraphRequestAdapter struct {
	core.GraphRequestAdapterBase
}

func NewGraphRequestAdapterWithFn(providerFn func() (absauth.AuthenticationProvider, error)) (*GraphRequestAdapter, error) {
	auth, err := providerFn()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	return NewGraphRequestAdapter(auth)
}

// NewGraphRequestAdapter creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapter(authenticationProvider absauth.AuthenticationProvider) (*GraphRequestAdapter, error) {
	return NewGraphRequestAdapterWithParseNodeFactory(authenticationProvider, nil)
}

// NewGraphRequestAdapterWithParseNodeFactory creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// parseNodeFactory: the factory used to create parse nodes
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapterWithParseNodeFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory) (*GraphRequestAdapter, error) {
	return NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider, parseNodeFactory, nil)
}

// NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// parseNodeFactory: the factory used to create parse nodes
// serializationWriterFactory: the factory used to create serialization writers
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactory(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory) (*GraphRequestAdapter, error) {
	return NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, parseNodeFactory, serializationWriterFactory, nil)
}

// NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient creates a new GraphRequestAdapter with the given parameters
// Parameters:
// authenticationProvider: the provider used to authenticate requests
// parseNodeFactory: the factory used to create parse nodes
// serializationWriterFactory: the factory used to create serialization writers
// httpClient: the client used to send requests
// Returns:
// a new GraphRequestAdapter
func NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider absauth.AuthenticationProvider, parseNodeFactory absser.ParseNodeFactory, serializationWriterFactory absser.SerializationWriterFactory, httpClient *nethttp.Client) (*GraphRequestAdapter, error) {
	baseAdapter, err := core.NewGraphRequestAdapterBaseWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(authenticationProvider, clientOptions, parseNodeFactory, serializationWriterFactory, httpClient)
	if err != nil {
		return nil, err
	}
	result := &GraphRequestAdapter{
		GraphRequestAdapterBase: *baseAdapter,
	}

	return result, nil
}
