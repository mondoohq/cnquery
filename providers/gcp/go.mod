module go.mondoo.com/cnquery/providers/gcp

replace go.mondoo.com/cnquery => ../..

go 1.20

require (
	cloud.google.com/go/accessapproval v1.7.1
	cloud.google.com/go/bigquery v1.54.0
	cloud.google.com/go/compute v1.23.0
	cloud.google.com/go/container v1.24.0
	cloud.google.com/go/functions v1.15.1
	cloud.google.com/go/iam v1.1.1
	cloud.google.com/go/kms v1.15.0
	cloud.google.com/go/logging v1.7.0
	cloud.google.com/go/longrunning v0.5.1
	cloud.google.com/go/monitoring v1.15.1
	cloud.google.com/go/pubsub v1.33.0
	cloud.google.com/go/recommender v1.10.1
	cloud.google.com/go/run v1.2.0
	cloud.google.com/go/serviceusage v1.7.1
	github.com/aws/smithy-go v1.14.2
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/rs/zerolog v1.30.0
	github.com/stretchr/testify v1.8.4
	go.mondoo.com/cnquery v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.11.0
	google.golang.org/api v0.138.0
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5
	google.golang.org/protobuf v1.31.0
)

require (
	cloud.google.com/go v0.110.6 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/apache/arrow/go/v12 v12.0.0 // indirect
	github.com/apache/thrift v0.16.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cockroachdb/errors v1.9.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/getsentry/sentry-go v0.13.0 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/gofrs/uuid v4.3.1+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.8+incompatible // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/s2a-go v0.1.5 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-plugin v1.4.8 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.mondoo.com/ranger-rpc v0.0.0-20230328135530-12135c17095f // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/tools v0.9.3 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230807174057-1744710a1577 // indirect
	google.golang.org/grpc v1.57.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	moul.io/http2curl v1.0.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
