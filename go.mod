module go.mondoo.com/cnquery

go 1.19

require (
	cloud.google.com/go/bigquery v1.44.0
	cloud.google.com/go/container v1.7.0
	cloud.google.com/go/kms v1.6.0
	cloud.google.com/go/logging v1.6.1
	cloud.google.com/go/pubsub v1.28.0
	cloud.google.com/go/recommender v1.8.0
	cloud.google.com/go/secretmanager v1.9.0
	cloud.google.com/go/serviceusage v1.4.0
	github.com/99designs/keyring v1.2.1
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/data/aztables v1.0.1
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azcertificates v0.7.1
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys v0.9.0
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets v0.11.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2 v2.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor v0.8.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity v0.9.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.1.0
	github.com/BurntSushi/toml v1.2.0
	github.com/Masterminds/semver v1.5.0
	github.com/StackExchange/wmi v1.2.1
	github.com/alecthomas/participle v0.3.0
	github.com/alecthomas/participle/v2 v2.0.0-alpha7
	github.com/aristanetworks/goeapi v1.0.0
	github.com/aws/aws-sdk-go v1.44.137
	github.com/aws/aws-sdk-go-v2 v1.17.3
	github.com/aws/aws-sdk-go-v2/config v1.18.7
	github.com/aws/aws-sdk-go-v2/credentials v1.13.7
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.21
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.18.2
	github.com/aws/aws-sdk-go-v2/service/acm v1.16.5
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.15.27
	github.com/aws/aws-sdk-go-v2/service/applicationautoscaling v1.15.25
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.25.1
	github.com/aws/aws-sdk-go-v2/service/backup v1.18.2
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.21.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.23.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.17.3
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.19.21
	github.com/aws/aws-sdk-go-v2/service/configservice v1.28.2
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.22.3
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.17.9
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.76.1
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.14.13
	github.com/aws/aws-sdk-go-v2/service/ecr v1.17.25
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.13.22
	github.com/aws/aws-sdk-go-v2/service/ecs v1.22.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.18.3
	github.com/aws/aws-sdk-go-v2/service/eks v1.26.0
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.24.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.14.25
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.18.28
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.17.4
	github.com/aws/aws-sdk-go-v2/service/emr v1.20.18
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.16.6
	github.com/aws/aws-sdk-go-v2/service/iam v1.18.25
	github.com/aws/aws-sdk-go-v2/service/kms v1.19.4
	github.com/aws/aws-sdk-go-v2/service/lambda v1.26.2
	github.com/aws/aws-sdk-go-v2/service/organizations v1.17.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.35.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.26.18
	github.com/aws/aws-sdk-go-v2/service/s3 v1.29.6
	github.com/aws/aws-sdk-go-v2/service/s3control v1.28.2
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.58.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.16.10
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.26.0
	github.com/aws/aws-sdk-go-v2/service/sns v1.18.8
	github.com/aws/aws-sdk-go-v2/service/ssm v1.33.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.17.7
	github.com/aws/smithy-go v1.13.5
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20221221165957-55f4180e6214
	github.com/c-bata/go-prompt v0.2.6
	github.com/charmbracelet/bubbles v0.14.0
	github.com/charmbracelet/bubbletea v0.23.0
	github.com/charmbracelet/lipgloss v0.6.0
	github.com/cockroachdb/errors v1.9.0
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/docker/cli v20.10.20+incompatible
	github.com/docker/docker v20.10.20+incompatible
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v4.3.1+incompatible
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/mock v1.6.0
	github.com/golangci/golangci-lint v1.49.0
	github.com/gonvenience/wrap v1.1.2
	github.com/gonvenience/ytbx v1.4.4
	github.com/google/go-cmp v0.5.9
	github.com/google/go-containerregistry v0.12.1
	github.com/google/go-github/v47 v47.1.1-0.20220922183808-8a4bdb5e400f
	github.com/google/uuid v1.3.0
	github.com/gosimple/slug v1.13.1
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.2.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-plugin v1.4.8
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/hcl/v2 v2.14.1
	github.com/hashicorp/vault/api v1.8.2
	github.com/hnakamur/go-scp v1.0.2
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f
	github.com/homeport/dyff v1.5.5
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kevinburke/ssh_config v1.2.0
	github.com/knqyf263/go-apk-version v0.0.0-20200609155635-041fdbb8563f
	github.com/knqyf263/go-rpmdb v0.0.0-20221030135625-4082a22221ce
	github.com/lithammer/fuzzysearch v1.1.5
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/masterzen/winrm v0.0.0-20220513085036-69f69afcd9e9
	github.com/mattn/go-isatty v0.0.16
	github.com/microsoft/kiota-abstractions-go v0.14.0
	github.com/microsoft/kiota-authentication-azure-go v0.5.0
	github.com/microsoft/kiota-serialization-json-go v0.7.2
	github.com/microsoft/kiota-serialization-text-go v0.6.0
	github.com/microsoftgraph/msgraph-sdk-go v0.48.0
	github.com/microsoftgraph/msgraph-sdk-go-core v0.30.1
	github.com/miekg/dns v1.1.50
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/ansi v0.0.0-20221106050444-61f0cd9a192a
	github.com/muesli/termenv v0.13.0
	github.com/okta/okta-sdk-golang/v2 v2.14.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/image-spec v1.1.0-rc2
	github.com/packethost/packngo v0.25.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.5
	github.com/pkg/term v1.2.0-beta.2
	github.com/rs/zerolog v1.28.0
	github.com/segmentio/fasthash v1.0.3
	github.com/segmentio/ksuid v1.0.4
	github.com/sethvargo/go-password v0.2.0
	github.com/slack-go/slack v0.11.4
	github.com/spf13/afero v1.9.2
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.6-0.20201009195203-85dd5c8bc61c
	github.com/spf13/viper v1.14.0
	github.com/stretchr/testify v1.8.1
	github.com/toravir/csd v0.0.0-20200911003203-13ae77ad849c
	github.com/vmware/go-vcloud-director/v2 v2.17.0
	github.com/vmware/goipmi v0.0.0-20181114221114-2333cd82d702
	github.com/vmware/govmomi v0.29.0
	github.com/xanzy/go-gitlab v0.74.0
	github.com/zclconf/go-cty v1.10.0
	go.mondoo.com/ranger-rpc v0.5.1-0.20220923135836-9e7732899d34
	go.opentelemetry.io/otel v1.11.1
	golang.org/x/crypto v0.2.0
	golang.org/x/net v0.3.0
	golang.org/x/oauth2 v0.3.0
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.3.0
	golang.org/x/text v0.5.0
	golang.org/x/tools v0.3.0
	google.golang.org/api v0.103.0
	google.golang.org/genproto v0.0.0-20221202195650-67e5cbc046fd
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools v2.2.0+incompatible
	howett.net/plist v1.0.0
	k8s.io/api v0.25.4
	k8s.io/apiextensions-apiserver v0.25.4
	k8s.io/apimachinery v0.25.4
	k8s.io/client-go v0.25.4
	k8s.io/component-base v0.25.4
	k8s.io/klog/v2 v2.70.1
	k8s.io/kubelet v0.25.0
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed
	sigs.k8s.io/yaml v1.3.0
)

require (
	4d63.com/gochecknoglobals v0.1.0 // indirect
	cloud.google.com/go v0.105.0 // indirect
	cloud.google.com/go/compute v1.13.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.1 // indirect
	cloud.google.com/go/iam v0.7.0 // indirect
	cloud.google.com/go/longrunning v0.3.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/Antonboom/errname v0.1.7 // indirect
	github.com/Antonboom/nilnil v0.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.28 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.21 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20211209120228-48547f28849e // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v0.7.0 // indirect
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6 // indirect
	github.com/Djarvur/go-err113 v0.0.0-20210108212216-aea10b59be24 // indirect
	github.com/GaijinEntertainment/go-exhaustruct/v2 v2.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/OpenPeeDeeP/depguard v1.1.0 // indirect
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/alexkohler/prealloc v1.0.0 // indirect
	github.com/alingse/asasalint v0.0.11 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195 // indirect
	github.com/armon/go-metrics v0.4.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/ashanbrown/forbidigo v1.3.0 // indirect
	github.com/ashanbrown/makezero v1.1.1 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.28 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.11 // indirect
	github.com/aymanbagabas/go-osc52 v1.0.3 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bkielbasa/cyclop v1.2.0 // indirect
	github.com/blizzy78/varnamelen v0.8.0 // indirect
	github.com/bombsimon/wsl/v3 v3.3.0 // indirect
	github.com/breml/bidichk v0.2.3 // indirect
	github.com/breml/errchkjson v0.3.0 // indirect
	github.com/butuzov/ireturn v0.1.1 // indirect
	github.com/cenkalti/backoff/v3 v3.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/charithe/durationcheck v0.0.9 // indirect
	github.com/chavacava/garif v0.0.0-20220630083739-93517212f375 // indirect
	github.com/cjlapao/common-go v0.0.29 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/creack/pty v1.1.11 // indirect
	github.com/curioswitch/go-reassign v0.1.2 // indirect
	github.com/daixiang0/gci v0.7.0 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/denis-tingaikin/go-header v0.4.3 // indirect
	github.com/dnaeon/go-vcr v1.2.0 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dougm/pretty v0.0.0-20171025230240-2ee9d7453c02 // indirect
	github.com/dvsekhvalnov/jose2go v1.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/esimonov/ifshort v1.0.4 // indirect
	github.com/ettle/strcase v0.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/firefart/nonamedreturns v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/getsentry/sentry-go v0.13.0 // indirect
	github.com/go-critic/go-critic v0.6.5 // indirect
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/go-toolsmith/astcast v1.0.0 // indirect
	github.com/go-toolsmith/astcopy v1.0.2 // indirect
	github.com/go-toolsmith/astequal v1.0.3 // indirect
	github.com/go-toolsmith/astfmt v1.0.0 // indirect
	github.com/go-toolsmith/astp v1.0.0 // indirect
	github.com/go-toolsmith/strparse v1.0.0 // indirect
	github.com/go-toolsmith/typep v1.0.2 // indirect
	github.com/go-xmlfmt/xmlfmt v1.1.2 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/golangci/check v0.0.0-20180506172741-cfe4005ccda2 // indirect
	github.com/golangci/dupl v0.0.0-20180902072040-3e9179ac440a // indirect
	github.com/golangci/go-misc v0.0.0-20220329215616-d24fe342adfe // indirect
	github.com/golangci/gofmt v0.0.0-20190930125516-244bba706f1a // indirect
	github.com/golangci/lint-1 v0.0.0-20191013205115-297bf364a8e0 // indirect
	github.com/golangci/maligned v0.0.0-20180506175553-b1d89398deca // indirect
	github.com/golangci/misspell v0.3.5 // indirect
	github.com/golangci/revgrep v0.0.0-20220804021717-745bb2f7c2e6 // indirect
	github.com/golangci/unconvert v0.0.0-20180507085042-28b1c447d1f4 // indirect
	github.com/gonvenience/bunt v1.3.4 // indirect
	github.com/gonvenience/neat v1.3.11 // indirect
	github.com/gonvenience/term v1.0.2 // indirect
	github.com/gonvenience/text v1.0.7 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.0 // indirect
	github.com/googleapis/gax-go/v2 v2.7.0 // indirect
	github.com/gordonklaus/ineffassign v0.0.0-20210914165742-4cc7213b9bc8 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/gostaticanalysis/analysisutil v0.7.1 // indirect
	github.com/gostaticanalysis/comment v1.4.2 // indirect
	github.com/gostaticanalysis/forcetypeassert v0.1.0 // indirect
	github.com/gostaticanalysis/nilerr v0.1.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.1 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/sdk v0.6.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20180604194846-3520598351bb // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jgautheron/goconst v1.5.1 // indirect
	github.com/jingyugao/rowserrcheck v1.1.1 // indirect
	github.com/jirfag/go-printf-func-name v0.0.0-20200119135958-7558a9eaa5af // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julz/importas v0.1.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/kisielk/errcheck v1.6.2 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kulti/thelper v0.6.3 // indirect
	github.com/kunwardeep/paralleltest v1.0.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/kyoh86/exportloopref v0.1.8 // indirect
	github.com/ldez/gomoddirectives v0.2.3 // indirect
	github.com/ldez/tagliatelle v0.3.1 // indirect
	github.com/leonklingele/grouper v1.1.0 // indirect
	github.com/lufeee/execinquery v1.2.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/maratori/testpackage v1.1.0 // indirect
	github.com/masterzen/simplexml v0.0.0-20190410153822-31eea3082786 // indirect
	github.com/matoous/godox v0.0.0-20210227103229-6504466cf951 // indirect
	github.com/mattn/go-ciede2000 v0.0.0-20170301095244-782e8c62fec3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.13 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mbilski/exhaustivestruct v1.2.0 // indirect
	github.com/mgechev/revive v1.2.3 // indirect
	github.com/microsoft/kiota-http-go v0.9.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/moricho/tparallel v0.2.1 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nakabonne/nestif v0.3.1 // indirect
	github.com/nbutton23/zxcvbn-go v0.0.0-20210217022336-fa2cb2858354 // indirect
	github.com/nishanths/exhaustive v0.8.1 // indirect
	github.com/nishanths/predeclared v0.2.2 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/phayes/checkstyle v0.0.0-20170904204023-bfd46e6a821d // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/polyfloyd/go-errorlint v1.0.2 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/quasilyte/go-ruleguard v0.3.18 // indirect
	github.com/quasilyte/gogrep v0.0.0-20220828223005-86e4605de09f // indirect
	github.com/quasilyte/regex/syntax v0.0.0-20210819130434-b3f0c404a727 // indirect
	github.com/quasilyte/stdinfo v0.0.0-20220114132959-f7386bf02567 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/rivo/uniseg v0.3.4 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryancurrah/gomodguard v1.2.4 // indirect
	github.com/ryanrolds/sqlclosecheck v0.3.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sahilm/fuzzy v0.1.0 // indirect
	github.com/sanposhiho/wastedassign/v2 v2.0.7 // indirect
	github.com/sashamelentyev/interfacebloat v1.1.0 // indirect
	github.com/sashamelentyev/usestdlibvars v1.15.1 // indirect
	github.com/securego/gosec/v2 v2.13.1 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shazow/go-diff v0.0.0-20160112020656-b6b7b6733b8c // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/sivchari/containedctx v1.0.2 // indirect
	github.com/sivchari/nosnakecase v1.7.0 // indirect
	github.com/sivchari/tenv v1.7.0 // indirect
	github.com/smartystreets/goconvey v1.7.2 // indirect
	github.com/sonatard/noctx v0.0.1 // indirect
	github.com/sourcegraph/go-diff v0.6.1 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/ssgreg/nlreturn/v2 v2.2.1 // indirect
	github.com/stbenjam/no-sprintf-host-port v0.1.1 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/sylvia7788/contextcheck v1.0.8 // indirect
	github.com/tdakkota/asciicheck v0.1.1 // indirect
	github.com/tetafro/godot v1.4.11 // indirect
	github.com/texttheater/golang-levenshtein v1.0.1 // indirect
	github.com/timakin/bodyclose v0.0.0-20210704033933-f49887972144 // indirect
	github.com/timonwong/logrlint v0.4.0 // indirect
	github.com/tomarrell/wrapcheck/v2 v2.6.2 // indirect
	github.com/tommy-muehle/go-mnd/v2 v2.5.0 // indirect
	github.com/ultraware/funlen v0.0.3 // indirect
	github.com/ultraware/whitespace v0.0.5 // indirect
	github.com/uudashr/gocognit v1.0.6 // indirect
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/virtuald/go-ordered-json v0.0.0-20170621173500-b18e6e673d74 // indirect
	github.com/yagipy/maintidx v1.0.0 // indirect
	github.com/yeya24/promlinter v0.2.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	gitlab.com/bosi/decorder v0.2.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.11.1 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/exp v0.0.0-20220827204233-334a2380cb91 // indirect
	golang.org/x/exp/typeparams v0.0.0-20220827204233-334a2380cb91 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/term v0.3.0 // indirect
	golang.org/x/time v0.1.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gotest.tools/v3 v3.3.0 // indirect
	honnef.co/go/tools v0.3.3 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	lukechampine.com/uint128 v1.1.1 // indirect
	modernc.org/cc/v3 v3.36.0 // indirect
	modernc.org/ccgo/v3 v3.16.6 // indirect
	modernc.org/libc v1.16.7 // indirect
	modernc.org/mathutil v1.4.1 // indirect
	modernc.org/memory v1.1.1 // indirect
	modernc.org/opt v0.1.1 // indirect
	modernc.org/sqlite v1.17.3 // indirect
	modernc.org/strutil v1.1.1 // indirect
	modernc.org/token v1.0.0 // indirect
	moul.io/http2curl v1.0.0 // indirect
	mvdan.cc/gofumpt v0.3.1 // indirect
	mvdan.cc/interfacer v0.0.0-20180901003855-c20040233aed // indirect
	mvdan.cc/lint v0.0.0-20170908181259-adc824a0674b // indirect
	mvdan.cc/unparam v0.0.0-20220831102321-2fc90a84c7ec // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
