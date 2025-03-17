module go.mondoo.com/cnquery/v11/providers/aws

replace go.mondoo.com/cnquery/v11 => ../..

go 1.23.0

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.9
	github.com/aws/aws-sdk-go-v2/credentials v1.17.62
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.38.2
	github.com/aws/aws-sdk-go-v2/service/account v1.23.1
	github.com/aws/aws-sdk-go-v2/service/acm v1.31.1
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.29.1
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.35.1
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.52.1
	github.com/aws/aws-sdk-go-v2/service/backup v1.41.1
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.45.2
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.48.2
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.44.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.47.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.56.0
	github.com/aws/aws-sdk-go-v2/service/configservice v1.52.1
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.51.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.42.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.210.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.43.0
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.32.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.54.2
	github.com/aws/aws-sdk-go-v2/service/efs v1.35.1
	github.com/aws/aws-sdk-go-v2/service/eks v1.60.1
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.45.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.29.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.45.0
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.33.1
	github.com/aws/aws-sdk-go-v2/service/emr v1.48.1
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.54.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.40.1
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.36.0
	github.com/aws/aws-sdk-go-v2/service/kms v1.38.1
	github.com/aws/aws-sdk-go-v2/service/lambda v1.70.1
	github.com/aws/aws-sdk-go-v2/service/neptune v1.36.1
	github.com/aws/aws-sdk-go-v2/service/organizations v1.38.1
	github.com/aws/aws-sdk-go-v2/service/rds v1.94.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.54.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.78.2
	github.com/aws/aws-sdk-go-v2/service/s3control v1.55.0
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.180.2
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.2
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.57.0
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.2
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.57.2
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17
	github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb v1.10.2
	github.com/aws/aws-sdk-go-v2/service/timestreamwrite v1.30.1
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.58.0
	github.com/aws/smithy-go v1.22.3
	github.com/cockroachdb/errors v1.11.3
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/rs/zerolog v1.33.0
	github.com/spf13/afero v1.14.0
	github.com/stretchr/testify v1.10.0
	go.mondoo.com/cnquery/v11 v11.45.1
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394
	k8s.io/client-go v0.32.3
)

require (
	cel.dev/expr v0.22.0 // indirect
	cloud.google.com/go v0.119.0 // indirect
	cloud.google.com/go/auth v0.15.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.7 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/iam v1.4.2 // indirect
	cloud.google.com/go/kms v1.21.1 // indirect
	cloud.google.com/go/longrunning v0.6.6 // indirect
	cloud.google.com/go/monitoring v1.24.1 // indirect
	cloud.google.com/go/secretmanager v1.14.6 // indirect
	cloud.google.com/go/storage v1.51.0 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.1 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/CycloneDX/cyclonedx-go v0.9.2 // indirect
	github.com/GoogleCloudPlatform/berglas/v2 v2.0.7 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/anchore/go-struct-converter v0.0.0-20221118182256-c68fdcfa2092 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.29.1 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.9.1 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.0 // indirect
	github.com/cncf/xds/go v0.0.0-20250121191232-2f005788dc42 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v28.0.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v28.0.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.8.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/facebookincubator/nvdtools v0.1.5 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/getsentry/sentry-go v0.31.1 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-git/go-git/v5 v5.14.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-containerregistry v0.20.3 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.6.3 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/api v1.16.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/hnakamur/go-scp v1.0.2 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/knqyf263/go-rpmdb v0.1.1 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/buildkit v0.20.1 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/mount v0.3.4 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/package-url/packageurl-go v0.1.3 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.8 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/spdx/tools-golang v0.5.5 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/tonistiigi/go-csvvalue v0.0.0-20240710180619-ddb21b71c0b4 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.mondoo.com/mondoo-go v0.0.0-20250312000344-2108b0433206 // indirect
	go.mondoo.com/ranger-rpc v0.6.5 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.33.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.4.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/api v0.226.0 // indirect
	google.golang.org/genproto v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.1 // indirect
	k8s.io/api v0.32.3 // indirect
	k8s.io/apimachinery v0.32.3 // indirect
	k8s.io/component-base v0.28.9 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kubelet v0.28.9 // indirect
	k8s.io/utils v0.0.0-20241210054802-24370beab758 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
	modernc.org/sqlite v1.36.1 // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
