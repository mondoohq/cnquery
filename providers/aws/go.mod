module go.mondoo.com/cnquery/v9/providers/aws

replace go.mondoo.com/cnquery/v9 => ../..

go 1.21

toolchain go1.21.3

require (
	github.com/aws/aws-sdk-go v1.45.24
	github.com/aws/aws-sdk-go-v2 v1.21.1
	github.com/aws/aws-sdk-go-v2/config v1.18.44
	github.com/aws/aws-sdk-go-v2/credentials v1.13.42
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.12
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.21.0
	github.com/aws/aws-sdk-go-v2/service/acm v1.19.0
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.18.0
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.22.5
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.30.6
	github.com/aws/aws-sdk-go-v2/service/backup v1.25.0
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.28.5
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.29.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.27.7
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.24.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.22.1
	github.com/aws/aws-sdk-go-v2/service/configservice v1.36.4
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.31.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.22.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.124.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.20.1
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.18.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.30.1
	github.com/aws/aws-sdk-go-v2/service/efs v1.21.7
	github.com/aws/aws-sdk-go-v2/service/eks v1.29.5
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.29.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.17.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.21.4
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.20.6
	github.com/aws/aws-sdk-go-v2/service/emr v1.28.6
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.28.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.22.5
	github.com/aws/aws-sdk-go-v2/service/kms v1.24.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.39.5
	github.com/aws/aws-sdk-go-v2/service/organizations v1.20.6
	github.com/aws/aws-sdk-go-v2/service/rds v1.54.0
	github.com/aws/aws-sdk-go-v2/service/redshift v1.29.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.40.0
	github.com/aws/aws-sdk-go-v2/service/s3control v1.33.0
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.108.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.21.4
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.36.2
	github.com/aws/aws-sdk-go-v2/service/sns v1.22.0
	github.com/aws/aws-sdk-go-v2/service/ssm v1.38.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.1
	github.com/aws/smithy-go v1.15.0
	github.com/cockroachdb/errors v1.11.1
	github.com/rs/zerolog v1.31.0
	github.com/spf13/afero v1.10.0
	github.com/stretchr/testify v1.8.4
	go.mondoo.com/cnquery/v9 v9.0.0-00010101000000-000000000000
	k8s.io/client-go v0.28.2
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.42 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.44 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.15.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.2 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20231003182221-725682229e60 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/cli v24.0.6+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.6+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/getsentry/sentry-go v0.25.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-containerregistry v0.16.1 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-plugin v1.5.2 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/hnakamur/go-scp v1.0.2 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.6 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sethvargo/go-password v0.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/smarty/assertions v1.15.1 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	go.mondoo.com/ranger-rpc v0.5.1 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/mod v0.13.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231009173412-8bfb1ae86b6c // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
