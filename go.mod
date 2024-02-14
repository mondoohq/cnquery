module go.mondoo.com/cnquery/v10

go 1.22

toolchain go1.22.0

require (
	cloud.google.com/go/logging v1.9.0
	cloud.google.com/go/secretmanager v1.11.5
	github.com/99designs/keyring v1.2.2
	github.com/BurntSushi/toml v1.3.2
	github.com/CycloneDX/cyclonedx-go v0.8.0
	github.com/GoogleCloudPlatform/berglas v1.0.3
	github.com/Masterminds/semver v1.5.0
	github.com/ProtonMail/go-crypto v1.0.0
	github.com/StackExchange/wmi v1.2.1
	// pin v0.3.0
	github.com/alecthomas/participle v0.3.0
	github.com/alecthomas/participle/v2 v2.1.1
	github.com/aws/aws-sdk-go v1.50.17
	github.com/aws/aws-sdk-go-v2 v1.25.0
	github.com/aws/aws-sdk-go-v2/config v1.27.0
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.147.0
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.21.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.27.0
	github.com/aws/aws-sdk-go-v2/service/ssm v1.46.0
	github.com/aws/smithy-go v1.20.0
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20240206212017-5795caca6e8e
	// pin v0.2.6
	github.com/c-bata/go-prompt v0.2.6
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/cockroachdb/errors v1.11.1
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/docker/cli v25.0.3+incompatible
	github.com/docker/docker v25.0.3+incompatible
	github.com/facebookincubator/nvdtools v0.1.5
	github.com/glebarez/go-sqlite v1.22.0
	github.com/go-git/go-git/v5 v5.11.0
	github.com/go-jose/go-jose/v3 v3.0.1
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/golang/mock v1.6.0
	github.com/golangci/golangci-lint v1.56.1
	github.com/google/go-cmdtest v0.4.0
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.19.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-hclog v1.6.2
	github.com/hashicorp/go-plugin v1.6.0
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/vault/api v1.12.0
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kevinburke/ssh_config v1.2.0
	github.com/knqyf263/go-apk-version v0.0.0-20200609155635-041fdbb8563f
	github.com/knqyf263/go-rpmdb v0.0.0-20231206071317-a8af76a6220f
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/masterzen/winrm v0.0.0-20231227165926-e811dad5ac77
	github.com/mattn/go-isatty v0.0.20
	github.com/miekg/dns v1.1.58
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6
	github.com/muesli/reflow v0.3.0
	github.com/muesli/termenv v0.15.2
	github.com/olekukonko/tablewriter v0.0.5
	// pin v1.1.0-rc5
	github.com/opencontainers/image-spec v1.1.0-rc5
	github.com/package-url/packageurl-go v0.1.2
	github.com/pierrec/lz4/v4 v4.1.21
	github.com/pkg/sftp v1.13.6
	// pin v1.2.0-beta.2
	github.com/pkg/term v1.2.0-beta.2
	github.com/rs/zerolog v1.32.0
	github.com/segmentio/fasthash v1.0.3
	github.com/segmentio/ksuid v1.0.4
	github.com/sethvargo/go-password v0.2.0
	github.com/spdx/tools-golang v0.5.3
	github.com/spf13/afero v1.11.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.18.2
	github.com/stretchr/testify v1.8.4
	github.com/toravir/csd v0.0.0-20200911003203-13ae77ad849c
	github.com/ulikunitz/xz v0.5.11
	go.mondoo.com/ranger-rpc v0.5.3
	go.opentelemetry.io/otel v1.23.1 // indirect
	golang.org/x/crypto v0.19.0
	golang.org/x/net v0.21.0
	golang.org/x/sync v0.6.0
	golang.org/x/sys v0.17.0
	golang.org/x/term v0.17.0
	golang.org/x/text v0.14.0
	golang.org/x/tools v0.18.0
	google.golang.org/genproto v0.0.0-20240205150955-31a09d347014
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240205150955-31a09d347014
	google.golang.org/grpc v1.61.1
	google.golang.org/protobuf v1.32.0
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.1
	// pin v0.28.4
	k8s.io/apimachinery v0.29.1
	// pin v0.28.4
	k8s.io/client-go v0.29.1
	// pin v0.28.4
	k8s.io/component-base v0.29.1
	// pin v0.28.4
	k8s.io/kubelet v0.29.1
	k8s.io/utils v0.0.0-20240102154912-e7106e64919e
	sigs.k8s.io/yaml v1.4.0
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Antonboom/testifylint v1.1.1 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6 // indirect
	github.com/GaijinEntertainment/go-exhaustruct/v3 v3.2.0 // indirect
	github.com/alecthomas/go-check-sumtype v0.1.4 // indirect
	github.com/anchore/go-struct-converter v0.0.0-20221118182256-c68fdcfa2092 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.0 // indirect
	github.com/bodgit/ntlmssp v0.0.0-20240208161302-b2d35758202d // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/bombsimon/wsl/v4 v4.2.0 // indirect
	github.com/catenacyber/perfsprint v0.6.0 // indirect
	github.com/ccojocar/zxcvbn-go v1.0.1 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/ghostiam/protogetter v0.3.4 // indirect
	github.com/go-errors/errors v1.5.0 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.5.0 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1 // indirect
	github.com/google/renameio v1.0.1 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jjti/go-spancheck v0.5.2 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/macabu/inamedparam v0.1.3 // indirect
	github.com/masterzen/simplexml v0.0.0-20190410153822-31eea3082786 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466 // indirect
	github.com/skeema/knownhosts v1.2.1 // indirect
	github.com/smarty/assertions v1.15.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/tidwall/transform v0.0.0-20201103190739-32f242e2dbde // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go-simpler.org/musttag v0.8.0 // indirect
	go-simpler.org/sloglint v0.4.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.48.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.48.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.22.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.1 // indirect
	go.opentelemetry.io/otel/sdk v1.22.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gotest.tools/v3 v3.5.1 // indirect
)

require (
	4d63.com/gocheckcompilerdirectives v1.2.1 // indirect
	4d63.com/gochecknoglobals v0.2.1 // indirect
	cloud.google.com/go v0.112.0 // indirect
	cloud.google.com/go/compute v1.23.4 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/kms v1.15.7 // indirect
	cloud.google.com/go/longrunning v0.5.5 // indirect
	cloud.google.com/go/storage v1.37.0 // indirect
	github.com/4meepo/tagalign v1.3.3 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/Abirdcfly/dupword v0.0.13 // indirect
	github.com/Antonboom/errname v0.1.12 // indirect
	github.com/Antonboom/nilnil v0.1.7 // indirect
	github.com/Djarvur/go-err113 v0.0.0-20210108212216-aea10b59be24 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/OpenPeeDeeP/depguard/v2 v2.2.0 // indirect
	github.com/alexkohler/nakedret/v2 v2.0.2 // indirect
	github.com/alexkohler/prealloc v1.0.0 // indirect
	github.com/alingse/asasalint v0.0.11 // indirect
	github.com/ashanbrown/forbidigo v1.6.0 // indirect
	github.com/ashanbrown/makezero v1.1.1 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.24.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.21.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.27.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bkielbasa/cyclop v1.2.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/blizzy78/varnamelen v0.8.0 // indirect
	github.com/breml/bidichk v0.2.7 // indirect
	github.com/breml/errchkjson v0.3.6 // indirect
	github.com/butuzov/ireturn v0.3.0 // indirect
	github.com/butuzov/mirror v1.1.0 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/charithe/durationcheck v0.0.10 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/chavacava/garif v0.1.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/containerd/console v1.0.4 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/curioswitch/go-reassign v0.2.0 // indirect
	github.com/daixiang0/gci v0.12.1 // indirect
	github.com/danieljoos/wincred v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/denis-tingaikin/go-header v0.4.3 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.1 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/esimonov/ifshort v1.0.4 // indirect
	github.com/ettle/strcase v0.2.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/firefart/nonamedreturns v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-critic/go-critic v0.11.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-toolsmith/astcast v1.1.0 // indirect
	github.com/go-toolsmith/astcopy v1.1.0 // indirect
	github.com/go-toolsmith/astequal v1.1.0 // indirect
	github.com/go-toolsmith/astfmt v1.1.0 // indirect
	github.com/go-toolsmith/astp v1.1.0 // indirect
	github.com/go-toolsmith/strparse v1.1.0 // indirect
	github.com/go-toolsmith/typep v1.1.0 // indirect
	github.com/go-xmlfmt/xmlfmt v1.1.2 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golangci/check v0.0.0-20180506172741-cfe4005ccda2 // indirect
	github.com/golangci/dupl v0.0.0-20180902072040-3e9179ac440a // indirect
	github.com/golangci/go-misc v0.0.0-20220329215616-d24fe342adfe // indirect
	github.com/golangci/gofmt v0.0.0-20231019111953-be8c47862aaa // indirect
	github.com/golangci/lint-1 v0.0.0-20191013205115-297bf364a8e0 // indirect
	github.com/golangci/maligned v0.0.0-20180506175553-b1d89398deca // indirect
	github.com/golangci/misspell v0.4.1 // indirect
	github.com/golangci/revgrep v0.5.2 // indirect
	github.com/golangci/unconvert v0.0.0-20180507085042-28b1c447d1f4 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gordonklaus/ineffassign v0.1.0 // indirect
	github.com/gostaticanalysis/analysisutil v0.7.1 // indirect
	github.com/gostaticanalysis/comment v1.4.2 // indirect
	github.com/gostaticanalysis/forcetypeassert v0.1.0 // indirect
	github.com/gostaticanalysis/nilerr v0.1.1 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/hnakamur/go-scp v1.0.2
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jgautheron/goconst v1.7.0 // indirect
	github.com/jingyugao/rowserrcheck v1.1.1 // indirect
	github.com/jirfag/go-printf-func-name v0.0.0-20200119135958-7558a9eaa5af // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julz/importas v0.1.0 // indirect
	github.com/kisielk/errcheck v1.7.0 // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/kkHAIKE/contextcheck v1.1.4 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kulti/thelper v0.6.3 // indirect
	github.com/kunwardeep/paralleltest v1.0.9 // indirect
	github.com/kyoh86/exportloopref v0.1.11 // indirect
	github.com/ldez/gomoddirectives v0.2.3 // indirect
	github.com/ldez/tagliatelle v0.5.0 // indirect
	github.com/leonklingele/grouper v1.1.1 // indirect
	github.com/lufeee/execinquery v1.2.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/maratori/testableexamples v1.0.0 // indirect
	github.com/maratori/testpackage v1.1.1 // indirect
	github.com/matoous/godox v0.0.0-20230222163458-006bad1f9d26 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-tty v0.0.5 // indirect
	github.com/mbilski/exhaustivestruct v1.2.0 // indirect
	github.com/mgechev/revive v1.3.7 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/moricho/tparallel v0.3.1 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/nakabonne/nestif v0.3.1 // indirect
	github.com/nishanths/exhaustive v0.12.0 // indirect
	github.com/nishanths/predeclared v0.2.2 // indirect
	github.com/nunnatsa/ginkgolinter v0.15.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/polyfloyd/go-errorlint v1.4.8 // indirect
	github.com/prometheus/client_golang v1.17.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/quasilyte/go-ruleguard v0.4.0 // indirect
	github.com/quasilyte/gogrep v0.5.0 // indirect
	github.com/quasilyte/regex/syntax v0.0.0-20210819130434-b3f0c404a727 // indirect
	github.com/quasilyte/stdinfo v0.0.0-20220114132959-f7386bf02567 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryancurrah/gomodguard v1.3.0 // indirect
	github.com/ryanrolds/sqlclosecheck v0.5.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sahilm/fuzzy v0.1.1-0.20230530133925-c48e322e2a8f // indirect
	github.com/sanposhiho/wastedassign/v2 v2.0.7 // indirect
	github.com/sashamelentyev/interfacebloat v1.1.0 // indirect
	github.com/sashamelentyev/usestdlibvars v1.24.0 // indirect
	github.com/securego/gosec/v2 v2.18.2 // indirect
	github.com/sethvargo/go-retry v0.2.4 // indirect
	github.com/shazow/go-diff v0.0.0-20160112020656-b6b7b6733b8c // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sivchari/containedctx v1.0.3 // indirect
	github.com/sivchari/nosnakecase v1.7.0 // indirect
	github.com/sivchari/tenv v1.7.1 // indirect
	github.com/sonatard/noctx v0.0.2 // indirect
	github.com/sourcegraph/go-diff v0.7.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/ssgreg/nlreturn/v2 v2.2.1 // indirect
	github.com/stbenjam/no-sprintf-host-port v0.1.1 // indirect
	github.com/stretchr/objx v0.5.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/t-yuki/gocover-cobertura v0.0.0-20180217150009-aaee18c8195c // indirect
	github.com/tdakkota/asciicheck v0.2.0 // indirect
	github.com/tetafro/godot v1.4.16 // indirect
	github.com/timakin/bodyclose v0.0.0-20230421092635-574207250966 // indirect
	github.com/timonwong/loggercheck v0.9.4 // indirect
	github.com/tomarrell/wrapcheck/v2 v2.8.1 // indirect
	github.com/tommy-muehle/go-mnd/v2 v2.5.1 // indirect
	github.com/ultraware/funlen v0.1.0 // indirect
	github.com/ultraware/whitespace v0.1.0 // indirect
	github.com/uudashr/gocognit v1.1.2 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/xen0n/gosmopolitan v1.2.2 // indirect
	github.com/yagipy/maintidx v1.0.0 // indirect
	github.com/yeya24/promlinter v0.2.0 // indirect
	github.com/ykadowak/zerologlint v0.1.5 // indirect
	gitlab.com/bosi/decorder v0.4.1 // indirect
	go.mondoo.com/mondoo-go v0.0.0-20240203120016-30f2eae8932a
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.23.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/exp v0.0.0-20240205201215-2c58cdc269a3
	golang.org/x/exp/typeparams v0.0.0-20231219180239-dc181d75b848 // indirect
	golang.org/x/mod v0.15.0
	golang.org/x/oauth2 v0.17.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/api v0.163.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240205150955-31a09d347014 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	honnef.co/go/tools v0.4.6 // indirect
	k8s.io/api v0.29.1 // indirect
	k8s.io/klog/v2 v2.110.1 // indirect
	modernc.org/libc v1.41.0 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/sqlite v1.28.0 // indirect
	moul.io/http2curl v1.0.0 // indirect
	mvdan.cc/gofumpt v0.6.0 // indirect
	mvdan.cc/interfacer v0.0.0-20180901003855-c20040233aed // indirect
	mvdan.cc/lint v0.0.0-20170908181259-adc824a0674b // indirect
	mvdan.cc/unparam v0.0.0-20240104100049-c549a3470d14 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)
