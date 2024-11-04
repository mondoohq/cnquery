module go.mondoo.com/cnquery/v11/providers/aws

replace go.mondoo.com/cnquery/v11 => ../..

go 1.23.0

require (
	github.com/aws/aws-sdk-go-v2 v1.32.3
	github.com/aws/aws-sdk-go-v2/config v1.28.1
	github.com/aws/aws-sdk-go-v2/credentials v1.17.42
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.18
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.34.3
	github.com/aws/aws-sdk-go-v2/service/acm v1.30.3
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.27.3
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.33.3
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.47.0
	github.com/aws/aws-sdk-go-v2/service/backup v1.39.4
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.40.3
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.44.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.42.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.43.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.47.1
	github.com/aws/aws-sdk-go-v2/service/configservice v1.50.3
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.44.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.36.3
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.187.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.36.3
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.27.3
	github.com/aws/aws-sdk-go-v2/service/ecs v1.49.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.33.3
	github.com/aws/aws-sdk-go-v2/service/eks v1.51.1
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.43.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.28.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.41.0
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.32.3
	github.com/aws/aws-sdk-go-v2/service/emr v1.46.1
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.50.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.37.3
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.32.3
	github.com/aws/aws-sdk-go-v2/service/kms v1.37.3
	github.com/aws/aws-sdk-go-v2/service/lambda v1.64.1
	github.com/aws/aws-sdk-go-v2/service/neptune v1.35.3
	github.com/aws/aws-sdk-go-v2/service/organizations v1.34.3
	github.com/aws/aws-sdk-go-v2/service/rds v1.89.0
	github.com/aws/aws-sdk-go-v2/service/redshift v1.51.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.66.2
	github.com/aws/aws-sdk-go-v2/service/s3control v1.49.3
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.166.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.34.3
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.54.4
	github.com/aws/aws-sdk-go-v2/service/sns v1.33.3
	github.com/aws/aws-sdk-go-v2/service/sqs v1.36.3
	github.com/aws/aws-sdk-go-v2/service/ssm v1.55.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.3
	github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb v1.6.3
	github.com/aws/aws-sdk-go-v2/service/timestreamwrite v1.29.3
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.55.2
	github.com/aws/smithy-go v1.22.0
	github.com/cockroachdb/errors v1.11.3
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/rs/zerolog v1.33.0
	github.com/spf13/afero v1.11.0
	github.com/stretchr/testify v1.9.0
	go.mondoo.com/cnquery/v11 v11.28.1
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	k8s.io/client-go v0.31.2
)

require (
	cel.dev/expr v0.18.0 // indirect
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.10.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.5 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	cloud.google.com/go/iam v1.2.2 // indirect
	cloud.google.com/go/kms v1.20.1 // indirect
	cloud.google.com/go/longrunning v0.6.2 // indirect
	cloud.google.com/go/monitoring v1.21.2 // indirect
	cloud.google.com/go/secretmanager v1.14.2 // indirect
	cloud.google.com/go/storage v1.46.0 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.16.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.3 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/GoogleCloudPlatform/berglas v1.0.3 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.25.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.49.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.49.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.27.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.3 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20241029204838-fb9784500689 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cncf/xds/go v0.0.0-20240905190251-b4127c9b8d78 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/cyphar/filepath-securejoin v0.3.4 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.3.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v27.3.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dvsekhvalnov/jose2go v1.7.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/envoyproxy/go-control-plane v0.13.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/getsentry/sentry-go v0.29.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.0 // indirect
	github.com/go-git/go-git/v5 v5.12.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.6.2 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/api v1.15.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/hnakamur/go-scp v1.0.2 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.7 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20201009195203-85dd5c8bc61c // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.mondoo.com/ranger-rpc v0.6.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.31.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.56.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.56.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.22.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.31.0 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/term v0.25.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/api v0.204.0 // indirect
	google.golang.org/genproto v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/grpc/stats/opentelemetry v0.0.0-20241028142157-ada6787961b3 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.1 // indirect
	k8s.io/utils v0.0.0-20240921022957-49e7df575cb6 // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
