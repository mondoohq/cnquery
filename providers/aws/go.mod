module go.mondoo.com/mql/v13/providers/aws

replace go.mondoo.com/mql/v13 => ../..

go 1.26.4

require (
	github.com/aws/aws-sdk-go-v2 v1.42.1
	github.com/aws/aws-sdk-go-v2/config v1.32.26
	github.com/aws/aws-sdk-go-v2/credentials v1.19.25
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.49.6
	github.com/aws/aws-sdk-go-v2/service/account v1.32.5
	github.com/aws/aws-sdk-go-v2/service/acm v1.41.0
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.47.7
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.40.7
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.35.7
	github.com/aws/aws-sdk-go-v2/service/appflow v1.52.3
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.43.1
	github.com/aws/aws-sdk-go-v2/service/appmesh v1.36.5
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.40.7
	github.com/aws/aws-sdk-go-v2/service/appstream v1.61.1
	github.com/aws/aws-sdk-go-v2/service/appsync v1.54.5
	github.com/aws/aws-sdk-go-v2/service/athena v1.58.5
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.68.0
	github.com/aws/aws-sdk-go-v2/service/backup v1.57.7
	github.com/aws/aws-sdk-go-v2/service/batch v1.66.1
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.64.1
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.56.1
	github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol v1.45.1
	github.com/aws/aws-sdk-go-v2/service/budgets v1.44.7
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.73.0
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.65.3
	github.com/aws/aws-sdk-go-v2/service/cloudhsmv2 v1.35.5
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.56.5
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.61.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.78.1
	github.com/aws/aws-sdk-go-v2/service/codeartifact v1.39.8
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.70.0
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.36.5
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.47.5
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.34.5
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.62.1
	github.com/aws/aws-sdk-go-v2/service/configservice v1.64.2
	github.com/aws/aws-sdk-go-v2/service/controltower v1.29.7
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.65.2
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.64.5
	github.com/aws/aws-sdk-go-v2/service/datasync v1.59.9
	github.com/aws/aws-sdk-go-v2/service/dax v1.30.3
	github.com/aws/aws-sdk-go-v2/service/detective v1.39.6
	github.com/aws/aws-sdk-go-v2/service/directoryservice v1.39.5
	github.com/aws/aws-sdk-go-v2/service/docdb v1.49.6
	github.com/aws/aws-sdk-go-v2/service/docdbelastic v1.21.8
	github.com/aws/aws-sdk-go-v2/service/drs v1.39.5
	github.com/aws/aws-sdk-go-v2/service/dsql v1.14.7
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.59.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.310.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.58.5
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.39.7
	github.com/aws/aws-sdk-go-v2/service/ecs v1.86.1
	github.com/aws/aws-sdk-go-v2/service/efs v1.42.2
	github.com/aws/aws-sdk-go-v2/service/eks v1.88.0
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.54.4
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.35.5
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.34.7
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.55.5
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.42.5
	github.com/aws/aws-sdk-go-v2/service/emr v1.61.2
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.46.7
	github.com/aws/aws-sdk-go-v2/service/firehose v1.44.1
	github.com/aws/aws-sdk-go-v2/service/fms v1.45.7
	github.com/aws/aws-sdk-go-v2/service/fsx v1.66.7
	github.com/aws/aws-sdk-go-v2/service/glue v1.147.0
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.80.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.54.6
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.37.8
	github.com/aws/aws-sdk-go-v2/service/inspector2 v1.49.3
	github.com/aws/aws-sdk-go-v2/service/kafka v1.54.1
	github.com/aws/aws-sdk-go-v2/service/keyspaces v1.26.6
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.44.3
	github.com/aws/aws-sdk-go-v2/service/kinesisvideo v1.34.6
	github.com/aws/aws-sdk-go-v2/service/kms v1.53.5
	github.com/aws/aws-sdk-go-v2/service/lakeformation v1.48.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.94.0
	github.com/aws/aws-sdk-go-v2/service/lightsail v1.56.2
	github.com/aws/aws-sdk-go-v2/service/macie2 v1.52.4
	github.com/aws/aws-sdk-go-v2/service/memorydb v1.34.7
	github.com/aws/aws-sdk-go-v2/service/mq v1.36.1
	github.com/aws/aws-sdk-go-v2/service/neptune v1.46.1
	github.com/aws/aws-sdk-go-v2/service/neptunegraph v1.22.6
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.62.0
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.72.1
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.11
	github.com/aws/aws-sdk-go-v2/service/pipes v1.24.7
	github.com/aws/aws-sdk-go-v2/service/qbusiness v1.35.6
	github.com/aws/aws-sdk-go-v2/service/ram v1.37.4
	github.com/aws/aws-sdk-go-v2/service/rds v1.119.4
	github.com/aws/aws-sdk-go-v2/service/redshift v1.63.4
	github.com/aws/aws-sdk-go-v2/service/route53 v1.63.4
	github.com/aws/aws-sdk-go-v2/service/route53domains v1.36.4
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.46.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.104.1
	github.com/aws/aws-sdk-go-v2/service/s3control v1.71.6
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.256.1
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.18.8
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.42.4
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.71.8
	github.com/aws/aws-sdk-go-v2/service/securitylake v1.26.3
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.62.5
	github.com/aws/aws-sdk-go-v2/service/sfn v1.43.1
	github.com/aws/aws-sdk-go-v2/service/shield v1.35.4
	github.com/aws/aws-sdk-go-v2/service/signer v1.33.7
	github.com/aws/aws-sdk-go-v2/service/sns v1.40.2
	github.com/aws/aws-sdk-go-v2/service/sqs v1.44.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.69.4
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.40.0
	github.com/aws/aws-sdk-go-v2/service/storagegateway v1.44.4
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.4
	github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb v1.20.7
	github.com/aws/aws-sdk-go-v2/service/timestreamwrite v1.36.1
	github.com/aws/aws-sdk-go-v2/service/transfer v1.73.4
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.74.0
	github.com/aws/aws-sdk-go-v2/service/workdocs v1.31.4
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.70.1
	github.com/aws/aws-sdk-go-v2/service/workspacesweb v1.40.8
	github.com/aws/smithy-go v1.27.3
	github.com/cockroachdb/errors v1.14.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/rs/zerolog v1.35.1
	github.com/spf13/afero v1.15.0
	github.com/stretchr/testify v1.11.1
	go.mondoo.com/mql/v13 v13.26.0
	golang.org/x/sync v0.21.0
	k8s.io/client-go v0.36.2
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/CycloneDX/cyclonedx-go v0.11.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/anchore/go-struct-converter v0.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.33.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.7 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.12.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudflare/circl v1.6.4 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.18.2 // indirect
	github.com/containerd/typeurl/v2 v2.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/cyphar/filepath-securejoin v0.7.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.6.1+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.8 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/endobit/oui v0.7.0 // indirect
	github.com/facebookincubator/nvdtools v0.1.5 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.1 // indirect
	github.com/getsentry/sentry-go v0.47.0 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.9.0 // indirect
	github.com/go-git/go-git/v5 v5.19.1 // indirect
	github.com/go-jose/go-jose/v3 v3.0.5 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-containerregistry v0.21.3 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.8.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/hnakamur/go-scp v1.0.2 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.6.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/knqyf263/go-rpmdb v0.1.1 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/buildkit v0.29.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/moby/api v1.55.0 // indirect
	github.com/moby/moby/client v0.5.0 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/sys/mount v0.3.5 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/oklog/run v1.2.0 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.3.0 // indirect
	github.com/olekukonko/ll v0.1.8 // indirect
	github.com/olekukonko/tablewriter v1.1.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/package-url/packageurl-go v0.1.5 // indirect
	github.com/pandatix/go-cvss v0.6.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pjbgf/sha1cd v0.6.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.10 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/protobom/protobom v0.5.6 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/shurcooL/graphql v0.0.0-20240915155400-7ee5256398cf // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/skeema/knownhosts v1.3.2 // indirect
	github.com/spdx/tools-golang v0.5.7 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tailscale/hujson v0.0.0-20260302212456-ecc657c15afd // indirect
	github.com/tonistiigi/go-csvvalue v0.0.0-20240814133006-030d3b2625d0 // indirect
	github.com/vbatts/tar-split v0.12.3 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.mondoo.com/mondoo-go v0.0.0-20260701003351-dac3d802a80f // indirect
	go.mondoo.com/ranger-rpc v0.8.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/mock v0.6.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260630182238-925bb5da69e7 // indirect
	google.golang.org/grpc v1.82.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.1 // indirect
	k8s.io/api v0.36.2 // indirect
	k8s.io/apimachinery v0.36.2 // indirect
	k8s.io/component-base v0.36.2 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260330154417-16be699c7b31 // indirect
	k8s.io/kubelet v0.36.2 // indirect
	k8s.io/utils v0.0.0-20260626114624-be93311217bd // indirect
	modernc.org/libc v1.73.5 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.53.0 // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/release-utils v0.12.4 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
