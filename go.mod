module go.mondoo.com/cnquery/v11

go 1.23.0

// Later versions break the build of the os provider for Windows
replace modernc.org/libc => modernc.org/libc v1.53.3

require (
	cloud.google.com/go/logging v1.13.0
	cloud.google.com/go/secretmanager v1.14.5
	github.com/99designs/keyring v1.2.2
	github.com/BurntSushi/toml v1.4.0
	github.com/CycloneDX/cyclonedx-go v0.9.2
	github.com/Masterminds/semver v1.5.0
	github.com/ProtonMail/go-crypto v1.1.6
	github.com/StackExchange/wmi v1.2.1
	// pin v0.3.0
	github.com/alecthomas/participle v0.3.0
	github.com/alecthomas/participle/v2 v2.1.1
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.9
	github.com/aws/aws-sdk-go-v2/credentials v1.17.62
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.208.0
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.28.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.2
	github.com/aws/aws-sdk-go-v2/service/ssm v1.57.2
	github.com/aws/smithy-go v1.22.3
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.9.1
	// pin v0.2.6
	github.com/c-bata/go-prompt v0.2.6
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/lipgloss v1.0.0
	github.com/cockroachdb/errors v1.11.3
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/docker/cli v27.5.1+incompatible
	github.com/docker/docker v27.5.1+incompatible
	github.com/facebookincubator/nvdtools v0.1.5
	github.com/glebarez/go-sqlite v1.22.0
	github.com/go-git/go-git/v5 v5.14.0
	github.com/go-jose/go-jose/v3 v3.0.4
	github.com/go-viper/mapstructure/v2 v2.2.1
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/go-cmdtest v0.4.0
	github.com/google/go-cmp v0.7.0
	github.com/google/go-containerregistry v0.20.3
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.6.3
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/vault/api v1.16.0
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kevinburke/ssh_config v1.2.0
	github.com/knqyf263/go-apk-version v0.0.0-20200609155635-041fdbb8563f
	github.com/knqyf263/go-rpmdb v0.1.1
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/masterzen/winrm v0.0.0-20240702205601-3fad6e106085
	github.com/mattn/go-isatty v0.0.20
	github.com/miekg/dns v1.1.63
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6
	github.com/muesli/reflow v0.3.0
	github.com/muesli/termenv v0.16.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/opencontainers/image-spec v1.1.1
	github.com/package-url/packageurl-go v0.1.3
	github.com/pierrec/lz4/v4 v4.1.22
	github.com/pkg/sftp v1.13.7
	// pin v1.2.0-beta.2
	github.com/pkg/term v1.2.0-beta.2
	github.com/rs/zerolog v1.33.0
	github.com/segmentio/fasthash v1.0.3
	github.com/segmentio/ksuid v1.0.4
	github.com/sethvargo/go-password v0.3.1
	github.com/spdx/tools-golang v0.5.5
	github.com/spf13/afero v1.12.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.10.0
	github.com/toravir/csd v0.0.0-20200911003203-13ae77ad849c
	github.com/ulikunitz/xz v0.5.12
	go.mondoo.com/ranger-rpc v0.6.5
	go.opentelemetry.io/otel v1.34.0 // indirect
	golang.org/x/crypto v0.36.0
	golang.org/x/net v0.37.0
	golang.org/x/sync v0.12.0
	golang.org/x/sys v0.31.0
	golang.org/x/term v0.30.0
	golang.org/x/text v0.23.0
	golang.org/x/tools v0.31.0
	google.golang.org/genproto v0.0.0-20250212204824-5a70512c5d8b
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250212204824-5a70512c5d8b
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.1
	// pin v0.28.9
	k8s.io/apimachinery v0.32.2
	// pin v0.28.9
	k8s.io/client-go v0.32.2
	// pin v0.28.9
	k8s.io/component-base v0.32.2
	// pin v0.28.9
	k8s.io/kubelet v0.32.2
	k8s.io/utils v0.0.0-20241210054802-24370beab758
	sigs.k8s.io/yaml v1.4.0
)

require (
	dario.cat/mergo v1.0.1 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6 // indirect
	github.com/anchore/go-struct-converter v0.0.0-20221118182256-c68fdcfa2092 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/bodgit/ntlmssp v0.0.0-20240506230425-31973bb52d9b // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/google/renameio v1.0.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/masterzen/simplexml v0.0.0-20190410153822-31eea3082786 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/tidwall/transform v0.0.0-20201103190739-32f242e2dbde // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.59.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.uber.org/mock v0.5.0
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

require (
	cloud.google.com/go v0.118.2 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/iam v1.4.0 // indirect
	cloud.google.com/go/kms v1.20.5 // indirect
	cloud.google.com/go/longrunning v0.6.4 // indirect
	cloud.google.com/go/storage v1.50.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.2
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.41.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.31.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.29.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/cloudflare/circl v1.6.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dvsekhvalnov/jose2go v1.8.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/getsentry/sentry-go v0.31.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.9 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/hnakamur/go-scp v1.0.2
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/magiconair/properties v1.8.9 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-tty v0.0.7 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	go.mondoo.com/mondoo-go v0.0.0-20250212000342-6747bee9aaa7
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250215185904-eff6e970281f
	golang.org/x/mod v0.24.0
	golang.org/x/oauth2 v0.26.0 // indirect
	golang.org/x/time v0.10.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/api v0.221.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250212204824-5a70512c5d8b // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.32.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
	modernc.org/sqlite v1.35.0 // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
)

require (
	github.com/GoogleCloudPlatform/berglas/v2 v2.0.7
	github.com/moby/buildkit v0.20.1
	github.com/moby/sys/mount v0.3.4
)

require (
	cel.dev/expr v0.20.0 // indirect
	cloud.google.com/go/auth v0.14.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.7 // indirect
	cloud.google.com/go/monitoring v1.24.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.26.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.50.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.50.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cncf/xds/go v0.0.0-20250121191232-2f005788dc42 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-errors/errors v1.5.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.4 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/smarty/assertions v1.15.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tonistiigi/go-csvvalue v0.0.0-20240710180619-ddb21b71c0b4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.34.0 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)
