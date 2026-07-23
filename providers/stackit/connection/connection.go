// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/services/alb"
	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"
	"github.com/stackitcloud/stackit-sdk-go/services/authorization"
	"github.com/stackitcloud/stackit-sdk-go/services/certificates"
	"github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/kms"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
	"github.com/stackitcloud/stackit-sdk-go/services/mariadb"
	"github.com/stackitcloud/stackit-sdk-go/services/modelserving"
	"github.com/stackitcloud/stackit-sdk-go/services/mongodbflex"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
	"github.com/stackitcloud/stackit-sdk-go/services/observability"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
	postgresflex "github.com/stackitcloud/stackit-sdk-go/services/postgresflex/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
	"github.com/stackitcloud/stackit-sdk-go/services/resourcemanager"
	"github.com/stackitcloud/stackit-sdk-go/services/secretsmanager"
	"github.com/stackitcloud/stackit-sdk-go/services/serverbackup"
	"github.com/stackitcloud/stackit-sdk-go/services/serverupdate"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	sqlserverflex "github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex/v2api"
	telemetrylink "github.com/stackitcloud/stackit-sdk-go/services/telemetrylink/v1betaapi"
	telemetryrouter "github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter/v1betaapi"
	vpn "github.com/stackitcloud/stackit-sdk-go/services/vpn/v1api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

var PlatformIdStackitProject = "//platformid.api.mondoo.app/runtime/stackit/project/"

const DefaultRegion = "eu01"

type StackitConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	projectID string
	region    string

	// project metadata captured during Verify so the provider can enrich the
	// detected asset (real name + labels + parent) the way the gcp and aws
	// providers do, without issuing a second resource-manager call.
	projectName   string
	projectParent string
	projectLabels map[string]string

	// configOpts includes WithRegion(region) — use for region-scoped services
	// (iaas, ske, objectstorage, loadbalancer, postgres-flex, mongodb-flex).
	configOpts []config.ConfigurationOption
	// configOptsGlobal omits WithRegion — use for global / project-only
	// services (resource-manager, dns, the rest of DBaaS, secrets-manager,
	// observability, service-account). Those APIs reject WithRegion.
	configOptsGlobal []config.ConfigurationOption

	iaasOnce             sync.Once
	iaasClient           *iaas.APIClient
	iaasErr              error
	skeOnce              sync.Once
	skeClient            *ske.APIClient
	skeErr               error
	dnsOnce              sync.Once
	dnsClient            *dns.APIClient
	dnsErr               error
	objectStorageOnce    sync.Once
	objectStorageClient  *objectstorage.APIClient
	objectStorageErr     error
	loadBalancerOnce     sync.Once
	loadBalancerClient   *loadbalancer.APIClient
	loadBalancerErr      error
	resourceManagerOnce  sync.Once
	resourceManagerClnt  *resourcemanager.APIClient
	resourceManagerErr   error
	postgresFlexOnce     sync.Once
	postgresFlexClient   *postgresflex.APIClient
	postgresFlexErr      error
	mongoDbFlexOnce      sync.Once
	mongoDbFlexClient    *mongodbflex.APIClient
	mongoDbFlexErr       error
	openSearchOnce       sync.Once
	openSearchClient     *opensearch.APIClient
	openSearchErr        error
	mariaDbOnce          sync.Once
	mariaDbClient        *mariadb.APIClient
	mariaDbErr           error
	redisOnce            sync.Once
	redisClient          *redis.APIClient
	redisErr             error
	rabbitMqOnce         sync.Once
	rabbitMqClient       *rabbitmq.APIClient
	rabbitMqErr          error
	logMeOnce            sync.Once
	logMeClient          *logme.APIClient
	logMeErr             error
	sqlServerFlexOnce    sync.Once
	sqlServerFlexClient  *sqlserverflex.APIClient
	sqlServerFlexErr     error
	secretsManagerOnce   sync.Once
	secretsManagerClient *secretsmanager.APIClient
	secretsManagerErr    error
	observabilityOnce    sync.Once
	observabilityClient  *observability.APIClient
	observabilityErr     error
	serviceAccountOnce   sync.Once
	serviceAccountClient *serviceaccount.APIClient
	serviceAccountErr    error
	sfsOnce              sync.Once
	sfsClient            *sfs.APIClient
	sfsErr               error
	telemetryRouterOnce  sync.Once
	telemetryRouterClnt  *telemetryrouter.APIClient
	telemetryRouterErr   error
	telemetryLinkOnce    sync.Once
	telemetryLinkClient  *telemetrylink.APIClient
	telemetryLinkErr     error
	modelServingOnce     sync.Once
	modelServingClient   *modelserving.APIClient
	modelServingErr      error
	albOnce              sync.Once
	albClient            *alb.APIClient
	albErr               error
	albWafOnce           sync.Once
	albWafClient         *albwaf.APIClient
	albWafErr            error
	certificatesOnce     sync.Once
	certificatesClient   *certificates.APIClient
	certificatesErr      error
	kmsOnce              sync.Once
	kmsClient            *kms.APIClient
	kmsErr               error
	authorizationOnce    sync.Once
	authorizationClient  *authorization.APIClient
	authorizationErr     error
	serverBackupOnce     sync.Once
	serverBackupClient   *serverbackup.APIClient
	serverBackupErr      error
	serverUpdateOnce     sync.Once
	serverUpdateClient   *serverupdate.APIClient
	serverUpdateErr      error
	vpnOnce              sync.Once
	vpnClient            *vpn.APIClient
	vpnErr               error
}

func NewStackitConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*StackitConnection, error) {
	projectID, _ := getOptionValueFrom(conf.Options, ProjectIDEnvVar, OptionProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("a STACKIT project ID is required; use --%s or set %s",
			OptionProjectID, ProjectIDEnvVar)
	}

	region, _ := getOptionValueFrom(conf.Options, RegionEnvVar, OptionRegion)
	if region == "" {
		region = DefaultRegion
	}

	globalOpts := buildAuthOptions(conf)
	regionalOpts := append([]config.ConfigurationOption{config.WithRegion(region)}, globalOpts...)

	conn := &StackitConnection{
		Connection:       plugin.NewConnection(id, asset),
		Conf:             conf,
		asset:            asset,
		projectID:        projectID,
		region:           region,
		configOpts:       regionalOpts,
		configOptsGlobal: globalOpts,
	}
	return conn, nil
}

// buildAuthOptions assembles the auth+endpoint options shared by every
// service client. The caller is responsible for prepending WithRegion when the
// target service is regional.
func buildAuthOptions(conf *inventory.Config) []config.ConfigurationOption {
	var opts []config.ConfigurationOption

	if endpoint, ok := getOptionValueFrom(conf.Options, EndpointEnvVar, OptionEndpoint); ok {
		opts = append(opts, config.WithEndpoint(endpoint))
	}

	if key, ok := credentialFor(conf, OptionServiceAccountKey, ServiceAccountKeyEnvVar); ok {
		opts = append(opts, config.WithServiceAccountKey(key))
	} else if path, ok := getOptionValueFrom(conf.Options, ServiceAccountKeyPathEnvVar, OptionServiceAccountKeyPath); ok {
		opts = append(opts, config.WithServiceAccountKeyPath(path))
	}

	if pk, ok := credentialFor(conf, OptionPrivateKey, PrivateKeyEnvVar); ok {
		opts = append(opts, config.WithPrivateKey(pk))
	} else if path, ok := getOptionValueFrom(conf.Options, PrivateKeyPathEnvVar, OptionPrivateKeyPath); ok {
		opts = append(opts, config.WithPrivateKeyPath(path))
	}

	if token, ok := credentialFor(conf, OptionToken, TokenEnvVar); ok {
		opts = append(opts, config.WithToken(token))
	}

	return opts
}

func credentialFor(conf *inventory.Config, option, envVar string) (string, bool) {
	for _, cred := range conf.Credentials {
		if cred.Type != vault.CredentialType_password || len(cred.Secret) == 0 {
			continue
		}
		if cred.User == option {
			return string(cred.Secret), true
		}
	}
	if v, ok := conf.Options[option]; ok && v != "" {
		return v, true
	}
	if v := os.Getenv(envVar); v != "" {
		return v, true
	}
	return "", false
}

func getOptionValueFrom(options map[string]string, envVar, option string) (string, bool) {
	value := os.Getenv(envVar)
	if v, ok := options[option]; ok && v != "" {
		value = v
	}
	return value, value != ""
}

func (c *StackitConnection) ProjectID() string     { return c.projectID }
func (c *StackitConnection) Region() string        { return c.region }
func (c *StackitConnection) ProjectName() string   { return c.projectName }
func (c *StackitConnection) ProjectParent() string { return c.projectParent }

// ProjectLabels returns the labels attached to the STACKIT project, captured
// during Verify. The returned map must not be mutated by callers.
func (c *StackitConnection) ProjectLabels() map[string]string { return c.projectLabels }

func (c *StackitConnection) Verify(ctx context.Context) error {
	client, err := c.ResourceManager()
	if err != nil {
		return err
	}
	resp, err := client.GetProjectExecute(ctx, c.projectID)
	if err != nil {
		log.Warn().Err(err).Msg("stackit> verify project lookup failed")
		return fmt.Errorf("failed to verify STACKIT connection for project %s: %w", c.projectID, err)
	}
	c.captureProjectMetadata(resp)
	return nil
}

// captureProjectMetadata records the human-readable project name, parent
// container, and labels from the resource-manager response so detect() can
// enrich the asset without a second API call.
func (c *StackitConnection) captureProjectMetadata(resp *resourcemanager.GetProjectResponse) {
	if resp == nil {
		return
	}
	c.projectName = resp.GetName()
	if labels, ok := resp.GetLabelsOk(); ok && labels != nil {
		// Snapshot the map so a later SDK mutation of its internal copy cannot
		// change our captured labels, honoring the ProjectLabels() contract.
		c.projectLabels = maps.Clone(labels)
	}
	if parent, ok := resp.GetParentOk(); ok {
		c.projectParent = parent.GetContainerId()
		if c.projectParent == "" {
			c.projectParent = parent.GetId()
		}
	}
}

func (c *StackitConnection) IaaS() (*iaas.APIClient, error) {
	c.iaasOnce.Do(func() {
		// The IaaS API rejects WithRegion in the client config and requires the
		// region as a per-call function parameter (which every IaaS resource
		// method supplies), so build the client without a bound region.
		c.iaasClient, c.iaasErr = iaas.NewAPIClient(c.configOptsGlobal...)
	})
	return c.iaasClient, c.iaasErr
}

func (c *StackitConnection) SKE() (*ske.APIClient, error) {
	c.skeOnce.Do(func() {
		// SKE likewise rejects WithRegion in the client config; the region is
		// passed as a per-call function parameter.
		c.skeClient, c.skeErr = ske.NewAPIClient(c.configOptsGlobal...)
	})
	return c.skeClient, c.skeErr
}

func (c *StackitConnection) DNS() (*dns.APIClient, error) {
	c.dnsOnce.Do(func() {
		c.dnsClient, c.dnsErr = dns.NewAPIClient(c.configOptsGlobal...)
	})
	return c.dnsClient, c.dnsErr
}

func (c *StackitConnection) ObjectStorage() (*objectstorage.APIClient, error) {
	c.objectStorageOnce.Do(func() {
		c.objectStorageClient, c.objectStorageErr = objectstorage.NewAPIClient(c.configOpts...)
	})
	return c.objectStorageClient, c.objectStorageErr
}

func (c *StackitConnection) LoadBalancer() (*loadbalancer.APIClient, error) {
	c.loadBalancerOnce.Do(func() {
		c.loadBalancerClient, c.loadBalancerErr = loadbalancer.NewAPIClient(c.configOpts...)
	})
	return c.loadBalancerClient, c.loadBalancerErr
}

func (c *StackitConnection) ResourceManager() (*resourcemanager.APIClient, error) {
	c.resourceManagerOnce.Do(func() {
		c.resourceManagerClnt, c.resourceManagerErr = resourcemanager.NewAPIClient(c.configOptsGlobal...)
	})
	return c.resourceManagerClnt, c.resourceManagerErr
}

func (c *StackitConnection) PostgresFlex() (*postgresflex.APIClient, error) {
	c.postgresFlexOnce.Do(func() {
		c.postgresFlexClient, c.postgresFlexErr = postgresflex.NewAPIClient(c.configOpts...)
	})
	return c.postgresFlexClient, c.postgresFlexErr
}

func (c *StackitConnection) MongoDbFlex() (*mongodbflex.APIClient, error) {
	c.mongoDbFlexOnce.Do(func() {
		// MongoDB Flex rejects WithRegion in the client config; the region is
		// passed as a per-call function parameter.
		c.mongoDbFlexClient, c.mongoDbFlexErr = mongodbflex.NewAPIClient(c.configOptsGlobal...)
	})
	return c.mongoDbFlexClient, c.mongoDbFlexErr
}

func (c *StackitConnection) OpenSearch() (*opensearch.APIClient, error) {
	c.openSearchOnce.Do(func() {
		c.openSearchClient, c.openSearchErr = opensearch.NewAPIClient(c.configOptsGlobal...)
	})
	return c.openSearchClient, c.openSearchErr
}

func (c *StackitConnection) MariaDb() (*mariadb.APIClient, error) {
	c.mariaDbOnce.Do(func() {
		c.mariaDbClient, c.mariaDbErr = mariadb.NewAPIClient(c.configOptsGlobal...)
	})
	return c.mariaDbClient, c.mariaDbErr
}

func (c *StackitConnection) Redis() (*redis.APIClient, error) {
	c.redisOnce.Do(func() {
		c.redisClient, c.redisErr = redis.NewAPIClient(c.configOptsGlobal...)
	})
	return c.redisClient, c.redisErr
}

func (c *StackitConnection) RabbitMq() (*rabbitmq.APIClient, error) {
	c.rabbitMqOnce.Do(func() {
		c.rabbitMqClient, c.rabbitMqErr = rabbitmq.NewAPIClient(c.configOptsGlobal...)
	})
	return c.rabbitMqClient, c.rabbitMqErr
}

func (c *StackitConnection) LogMe() (*logme.APIClient, error) {
	c.logMeOnce.Do(func() {
		c.logMeClient, c.logMeErr = logme.NewAPIClient(c.configOptsGlobal...)
	})
	return c.logMeClient, c.logMeErr
}

func (c *StackitConnection) SqlServerFlex() (*sqlserverflex.APIClient, error) {
	c.sqlServerFlexOnce.Do(func() {
		c.sqlServerFlexClient, c.sqlServerFlexErr = sqlserverflex.NewAPIClient(c.configOpts...)
	})
	return c.sqlServerFlexClient, c.sqlServerFlexErr
}

func (c *StackitConnection) SecretsManager() (*secretsmanager.APIClient, error) {
	c.secretsManagerOnce.Do(func() {
		c.secretsManagerClient, c.secretsManagerErr = secretsmanager.NewAPIClient(c.configOptsGlobal...)
	})
	return c.secretsManagerClient, c.secretsManagerErr
}

func (c *StackitConnection) Observability() (*observability.APIClient, error) {
	c.observabilityOnce.Do(func() {
		c.observabilityClient, c.observabilityErr = observability.NewAPIClient(c.configOptsGlobal...)
	})
	return c.observabilityClient, c.observabilityErr
}

func (c *StackitConnection) ServiceAccount() (*serviceaccount.APIClient, error) {
	c.serviceAccountOnce.Do(func() {
		c.serviceAccountClient, c.serviceAccountErr = serviceaccount.NewAPIClient(c.configOptsGlobal...)
	})
	return c.serviceAccountClient, c.serviceAccountErr
}

func (c *StackitConnection) Sfs() (*sfs.APIClient, error) {
	c.sfsOnce.Do(func() {
		// SFS rejects WithRegion in the client config; every SFS resource method
		// passes the region as a per-call function parameter.
		c.sfsClient, c.sfsErr = sfs.NewAPIClient(c.configOptsGlobal...)
	})
	return c.sfsClient, c.sfsErr
}

func (c *StackitConnection) ModelServing() (*modelserving.APIClient, error) {
	c.modelServingOnce.Do(func() {
		c.modelServingClient, c.modelServingErr = modelserving.NewAPIClient(c.configOptsGlobal...)
	})
	return c.modelServingClient, c.modelServingErr
}

func (c *StackitConnection) TelemetryRouter() (*telemetryrouter.APIClient, error) {
	c.telemetryRouterOnce.Do(func() {
		c.telemetryRouterClnt, c.telemetryRouterErr = telemetryrouter.NewAPIClient(c.configOptsGlobal...)
	})
	return c.telemetryRouterClnt, c.telemetryRouterErr
}

func (c *StackitConnection) TelemetryLink() (*telemetrylink.APIClient, error) {
	c.telemetryLinkOnce.Do(func() {
		c.telemetryLinkClient, c.telemetryLinkErr = telemetrylink.NewAPIClient(c.configOptsGlobal...)
	})
	return c.telemetryLinkClient, c.telemetryLinkErr
}

func (c *StackitConnection) ALB() (*alb.APIClient, error) {
	c.albOnce.Do(func() {
		c.albClient, c.albErr = alb.NewAPIClient(c.configOpts...)
	})
	return c.albClient, c.albErr
}

func (c *StackitConnection) AlbWaf() (*albwaf.APIClient, error) {
	c.albWafOnce.Do(func() {
		// Mirrors ALB (same product family): the WAF API is region-scoped via
		// the client config and also accepts the region as a per-call param.
		c.albWafClient, c.albWafErr = albwaf.NewAPIClient(c.configOpts...)
	})
	return c.albWafClient, c.albWafErr
}

func (c *StackitConnection) Certificates() (*certificates.APIClient, error) {
	c.certificatesOnce.Do(func() {
		c.certificatesClient, c.certificatesErr = certificates.NewAPIClient(c.configOpts...)
	})
	return c.certificatesClient, c.certificatesErr
}

func (c *StackitConnection) KMS() (*kms.APIClient, error) {
	c.kmsOnce.Do(func() {
		c.kmsClient, c.kmsErr = kms.NewAPIClient(c.configOpts...)
	})
	return c.kmsClient, c.kmsErr
}

func (c *StackitConnection) Authorization() (*authorization.APIClient, error) {
	c.authorizationOnce.Do(func() {
		c.authorizationClient, c.authorizationErr = authorization.NewAPIClient(c.configOptsGlobal...)
	})
	return c.authorizationClient, c.authorizationErr
}

func (c *StackitConnection) ServerBackup() (*serverbackup.APIClient, error) {
	c.serverBackupOnce.Do(func() {
		// Server Backup rejects WithRegion in the client config; every method
		// passes the region as a per-call function parameter.
		c.serverBackupClient, c.serverBackupErr = serverbackup.NewAPIClient(c.configOptsGlobal...)
	})
	return c.serverBackupClient, c.serverBackupErr
}

func (c *StackitConnection) ServerUpdate() (*serverupdate.APIClient, error) {
	c.serverUpdateOnce.Do(func() {
		// Server Update rejects WithRegion in the client config; every method
		// passes the region as a per-call function parameter.
		c.serverUpdateClient, c.serverUpdateErr = serverupdate.NewAPIClient(c.configOptsGlobal...)
	})
	return c.serverUpdateClient, c.serverUpdateErr
}

func (c *StackitConnection) Vpn() (*vpn.APIClient, error) {
	c.vpnOnce.Do(func() {
		// VPN rejects WithRegion in the client config; every method passes the
		// region as a per-call function parameter.
		c.vpnClient, c.vpnErr = vpn.NewAPIClient(c.configOptsGlobal...)
	})
	return c.vpnClient, c.vpnErr
}

func (c *StackitConnection) Asset() *inventory.Asset { return c.asset }
func (c *StackitConnection) Name() string            { return "stackit" }

func (c *StackitConnection) PlatformInfo() *inventory.Platform {
	p := &inventory.Platform{
		// Matches the provider's AssetUrlTrees (technology=stackit -> project),
		// mirroring how gcp emits {"gcp", projectID, ...} and aws emits
		// {"aws", accountID, ...}.
		TechnologyUrlSegments: []string{"stackit", c.projectID},
	}
	PlatformByName("stackit-project").Apply(p)
	if c.projectName != "" {
		p.Title = "STACKIT Project " + c.projectName
	}
	return p
}

func (c *StackitConnection) Identifier() string {
	if c.projectID == "" {
		return ""
	}
	return PlatformIdStackitProject + c.projectID
}

var ErrMissingProjectID = errors.New("missing STACKIT project ID")
