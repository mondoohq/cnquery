NAMESPACE=mondoo

ifndef LATEST_VERSION_TAG
# echo "read LATEST_VERSION_TAG from git"
LATEST_VERSION_TAG=$(shell git describe --abbrev=0 --tags)
endif

ifndef MANIFEST_VERSION
# echo "read MANIFEST_VERSION from git"
MANIFEST_VERSION=$(shell git describe --abbrev=0 --tags)
endif

ifndef TAG
# echo "read TAG from git"
TAG=$(shell git log --pretty=format:'%h' -n 1)
endif

ifndef VERSION
# echo "read VERSION from git"
VERSION=${LATEST_VERSION_TAG}+$(shell git rev-list --count HEAD)
endif

ifndef TARGETOS
	TARGETOS = $(shell go env GOOS)
endif

BIN_SUFFIX = ""
ifeq ($(TARGETOS),windows)
	BIN_SUFFIX=".exe"
endif

LDFLAGS=-ldflags "-s -w -X go.mondoo.com/cnquery/v11.Version=${VERSION} -X go.mondoo.com/cnquery/v11.Build=${TAG}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.com/cnquery/v11.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnquery/v11.Build=${TAG} -s -w"

.PHONY: info/ldflags
info/ldflags:
	$(info go run ${LDFLAGS} apps/cnquery/cnquery.go)
	@:

#   🧹 CLEAN   #

clean/proto:
	find . -not -path './.*' \( -name '*.ranger.go' -or -name '*.pb.go' -or -name '*.actions.go' -or -name '*-packr.go' -or -name '*.swagger.json' \) -delete

.PHONY: version
version:
	@echo $(VERSION)

#   🔨 TOOLS       #

prep: prep/tools

prep/tools/windows:
	go get -u google.golang.org/protobuf
	go get -u gotest.tools/gotestsum


prep/tools/protolint:
	# protobuf linting
	go install github.com/yoheimuta/protolint/cmd/protolint@latest

prep/tools: prep/tools/protolint prep/tools/mockgen
	# additional helper
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/hashicorp/copywrite@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc-swagger@latest

prep/tools/mockgen:
	go install go.uber.org/mock/mockgen@latest

#   🌙 MQL/MOTOR   #

cnquery/generate: clean/proto llx/generate shared/generate explorer/generate sbom/generate reporter/generate providers

cnquery/generate/core: clean/proto llx/generate shared/generate providers/proto providers/build/mock providers/build/core explorer/generate sbom/generate reporter/generate

define buildProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_DIST_BIN = "./dist/${$@_NAME}")
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	echo "--> [${$@_NAME}] process resources"
	./lr go ${$@_HOME}/resources/${$@_NAME}.lr --dist ${$@_DIST}
	./lr docs yaml ${$@_HOME}/resources/${$@_NAME}.lr --docs-file ${$@_HOME}/resources/${$@_NAME}.lr.manifest.yaml
	./lr docs json ${$@_HOME}/resources/${$@_NAME}.lr.manifest.yaml
	echo "--> [${$@_NAME}] generate CLI json"
	cd ${$@_HOME} && go run ./gen/main.go .
	echo "--> [${$@_NAME}] creating ${$@_BIN}"
	cd ${$@_HOME} && GOOS=${TARGETOS} go build -o ${$@_DIST_BIN}${BIN_SUFFIX} ./main.go
endef

define buildProviderDist
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_DIST_BIN = "./dist/${$@_NAME}")
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	echo "--> [${$@_NAME}] process resources"
	./lr go ${$@_HOME}/resources/${$@_NAME}.lr --dist ${$@_DIST}
	./lr docs yaml ${$@_HOME}/resources/${$@_NAME}.lr --docs-file ${$@_HOME}/resources/${$@_NAME}.lr.manifest.yaml
	./lr docs json ${$@_HOME}/resources/${$@_NAME}.lr.manifest.yaml
	echo "--> [${$@_NAME}] generate CLI json"
	cd ${$@_HOME} && go run ./gen/main.go .
	echo "--> [${$@_NAME}] creating ${$@_BIN}"
	cd ${$@_HOME} && CGO_ENABLED=0 GOOS=${TARGETOS} go build ${LDFLAGSDIST} -o ${$@_DIST_BIN}${BIN_SUFFIX} ./main.go
endef

define installProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	$(eval $@_DST = "$(HOME)/.config/mondoo/providers/${$@_NAME}")
	echo "--> install ${$@_NAME}"
	install -d "${$@_DST}"
	install -m 755 ./${$@_DIST}/${$@_NAME} ${$@_DST}/
	install -m 644 ./${$@_DIST}/${$@_NAME}.json ${$@_DST}/
	install -m 644 ./${$@_DIST}/${$@_NAME}.resources.json ${$@_DST}/
endef

define bundleProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_DST = "${$@_DIST}/${$@_NAME}.tar.xz")
	echo "--> bundle ${$@_NAME} to ${$@_DST} (this may take a while)"
	tar -cf ${$@_DST} --no-same-owner \
		--use-compress-program='xz -9v' \
		-C ${$@_DIST} \
		${$@_NAME} ${$@_NAME}.json ${$@_NAME}.resources.json
	ls -lha ${$@_DST}
endef

define testProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_PKGS = $(shell go list ./${$@_HOME}/...))
	echo "--> test ${$@_NAME} in ${$@_HOME}"
	gotestsum --junitfile ./report_${$@_NAME}.xml --format pkgname -- -cover ${$@_PKGS}
endef

define testGoModProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_PKGS = $(shell bash -c "cd ${$@_HOME} && go list ./..."))
	echo "--> test ${$@_NAME} in ${$@_HOME}"
	cd ${$@_HOME} && gotestsum --junitfile ../../report_${$@_NAME}.xml --format pkgname -- -cover ${$@_PKGS}
endef

.PHONY: providers
providers: providers/proto providers/config providers/build

.PHONY: providers/proto
providers/proto:
	go generate .
	go generate ./providers-sdk/v1/vault
	go generate ./providers-sdk/v1/resources
	go generate ./providers-sdk/v1/inventory
	go generate ./providers-sdk/v1/plugin

.PHONY: providers/config
providers/config:
	go run ./providers-sdk/v1/util/configure/configure.go -f providers.yaml -o providers/builtin_dev.go

.PHONY: providers/defaults
providers/defaults:
	go run ./providers-sdk/v1/util/defaults/defaults.go -o providers/defaults.go

.PHONY: providers/lr
providers/lr:
	go build -o lr ./providers-sdk/v1/lr/cli/main.go

providers/lr/install: providers/lr
	cp ./lr ${GOPATH}/bin

.PHONY: providers/build
# Note we need \ to escape the target line into multiple lines
providers/build: \
	providers/build/mock \
	providers/build/core \
	providers/build/network \
	providers/build/os \
	providers/build/ipmi \
	providers/build/oci \
	providers/build/slack \
	providers/build/github \
	providers/build/gitlab \
	providers/build/terraform \
	providers/build/vsphere \
	providers/build/opcua \
	providers/build/okta \
	providers/build/google-workspace \
	providers/build/arista \
	providers/build/equinix \
	providers/build/vcd \
	providers/build/gcp \
	providers/build/k8s \
	providers/build/azure \
	providers/build/ms365 \
	providers/build/aws \
	providers/build/atlassian \
	providers/build/cloudformation \
	providers/build/nmap

.PHONY: providers/install
# Note we need \ to escape the target line into multiple lines
providers/install: \
	providers/install/network \
	providers/install/os \
	providers/install/ipmi \
	providers/install/oci \
	providers/install/slack \
	providers/install/github \
	providers/install/gitlab \
	providers/install/terraform \
	providers/install/vsphere \
	providers/install/opcua \
	providers/install/okta \
	providers/install/google-workspace \
	providers/install/arista \
	providers/install/equinix \
	providers/install/vcd \
	providers/install/gcp \
	providers/install/k8s \
	providers/install/azure \
	providers/install/ms365 \
	providers/install/atlassian \
	providers/install/aws \
	providers/install/cloudformation \
	providers/build/nmap

providers/build/mock: providers/lr
	./lr go providers-sdk/v1/testutils/mockprovider/resources/mockprovider.lr

providers/build/core: providers/lr
	@$(call buildProvider, providers/core)

providers/build/network: providers/lr
	@$(call buildProvider, providers/network)
providers/install/network:
	@$(call installProvider, providers/network)

providers/build/os: providers/lr
	@$(call buildProvider, providers/os)
providers/install/os:
	@$(call installProvider, providers/os)

providers/build/ipmi: providers/lr
	@$(call buildProvider, providers/ipmi)
providers/install/ipmi:
	@$(call installProvider, providers/ipmi)

providers/build/oci: providers/lr
	@$(call buildProvider, providers/oci)
providers/install/oci:
	@$(call installProvider, providers/oci)

providers/build/slack: providers/lr
	@$(call buildProvider, providers/slack)
providers/install/slack:
	@$(call installProvider, providers/slack)

providers/build/github: providers/lr
	@$(call buildProvider, providers/github)
providers/install/github:
	@$(call installProvider, providers/github)

providers/build/gitlab: providers/lr
	@$(call buildProvider, providers/gitlab)
providers/install/gitlab:
	@$(call installProvider, providers/gitlab)

providers/build/terraform: providers/lr
	@$(call buildProvider, providers/terraform)
providers/install/terraform:
	@$(call installProvider, providers/terraform)

providers/build/vsphere: providers/lr
	@$(call buildProvider, providers/vsphere)
providers/install/vsphere:
	@$(call installProvider, providers/vsphere)

providers/build/opcua: providers/lr
	@$(call buildProvider, providers/opcua)
providers/install/opcua:
	@$(call installProvider, providers/opcua)

providers/build/okta: providers/lr
	@$(call buildProvider, providers/okta)
providers/install/okta:
	@$(call installProvider, providers/okta)

providers/build/google-workspace: providers/lr
	@$(call buildProvider, providers/google-workspace)
providers/install/google-workspace:
	@$(call installProvider, providers/google-workspace)

providers/build/arista: providers/lr
	@$(call buildProvider, providers/arista)
providers/install/arista:
	@$(call installProvider, providers/arista)

providers/build/equinix: providers/lr
	@$(call buildProvider, providers/equinix)
providers/install/equinix:
	@$(call installProvider, providers/equinix)

providers/build/vcd: providers/lr
	@$(call buildProvider, providers/vcd)
providers/install/vcd:
	@$(call installProvider, providers/vcd)

providers/build/k8s: providers/lr
	@$(call buildProvider, providers/k8s)
providers/install/k8s:
	@$(call installProvider, providers/k8s)

providers/build/gcp: providers/lr
	@$(call buildProvider, providers/gcp)
providers/install/gcp:
	@$(call installProvider, providers/gcp)

providers/build/azure: providers/lr
	@$(call buildProvider, providers/azure)
providers/install/azure:
	@$(call installProvider, providers/azure)

providers/build/aws: providers/lr
	@$(call buildProvider, providers/aws)
providers/install/aws:
	@$(call installProvider, providers/aws)

providers/build/atlassian: providers/lr
	@$(call buildProvider, providers/atlassian)
providers/install/atlassian:
	@$(call installProvider, providers/atlassian)

providers/build/ms365: providers/lr
	@$(call buildProvider, providers/ms365)
providers/install/ms365:
	@$(call installProvider, providers/ms365)

providers/build/cloudformation: providers/lr
	@$(call buildProvider, providers/cloudformation)
providers/install/cloudformation:
	@$(call installProvider, providers/cloudformation)

providers/build/nmap: providers/lr
	@$(call buildProvider, providers/nmap)
providers/install/nmap:
	@$(call installProvider, providers/nmap)

providers/dist:
	@$(call buildProviderDist, providers/network)
	@$(call buildProviderDist, providers/os)
	@$(call buildProviderDist, providers/ipmi)
	@$(call buildProviderDist, providers/oci)
	@$(call buildProviderDist, providers/slack)
	@$(call buildProviderDist, providers/github)
	@$(call buildProviderDist, providers/gitlab)
	@$(call buildProviderDist, providers/terraform)
	@$(call buildProviderDist, providers/vsphere)
	@$(call buildProviderDist, providers/opcua)
	@$(call buildProviderDist, providers/okta)
	@$(call buildProviderDist, providers/google-workspace)
	@$(call buildProviderDist, providers/arista)
	@$(call buildProviderDist, providers/equinix)
	@$(call buildProviderDist, providers/vcd)
	@$(call buildProviderDist, providers/gcp)
	@$(call buildProviderDist, providers/k8s)
	@$(call buildProviderDist, providers/azure)
	@$(call buildProviderDist, providers/ms365)
	@$(call buildProviderDist, providers/aws)
	@$(call buildProviderDist, providers/atlassian)
	@$(call buildProviderDist, providers/cloudformation)
	@$(call buildProviderDist, providers/nmap)

providers/bundle:
	@$(call bundleProvider, providers/network)
	@$(call bundleProvider, providers/os)
	@$(call bundleProvider, providers/ipmi)
	@$(call bundleProvider, providers/oci)
	@$(call bundleProvider, providers/slack)
	@$(call bundleProvider, providers/github)
	@$(call bundleProvider, providers/gitlab)
	@$(call bundleProvider, providers/terraform)
	@$(call bundleProvider, providers/vsphere)
	@$(call bundleProvider, providers/opcua)
	@$(call bundleProvider, providers/okta)
	@$(call bundleProvider, providers/google-workspace)
	@$(call bundleProvider, providers/arista)
	@$(call bundleProvider, providers/equinix)
	@$(call bundleProvider, providers/vcd)
	@$(call bundleProvider, providers/gcp)
	@$(call bundleProvider, providers/k8s)
	@$(call bundleProvider, providers/azure)
	@$(call bundleProvider, providers/ms365)
	@$(call bundleProvider, providers/aws)
	@$(call bundleProvider, providers/atlassian)
	@$(call bundleProvider, providers/cloudformation)
	@$(call bundleProvider, providers/nmap)

providers/test:
	@$(call testProvider, providers/core)
	@$(call testProvider, providers/network)
	@$(call testProvider, providers/os)
	@$(call testGoModProvider, providers/ipmi)
	@$(call testGoModProvider, providers/oci)
	@$(call testGoModProvider, providers/slack)
	@$(call testGoModProvider, providers/github)
	@$(call testGoModProvider, providers/gitlab)
	@$(call testGoModProvider, providers/terraform)
	@$(call testGoModProvider, providers/vsphere)
	@$(call testGoModProvider, providers/opcua)
	@$(call testGoModProvider, providers/okta)
	@$(call testGoModProvider, providers/google-workspace)
	@$(call testGoModProvider, providers/arista)
	@$(call testGoModProvider, providers/equinix)
	@$(call testGoModProvider, providers/vcd)
	@$(call testGoModProvider, providers/gcp)
	@$(call testGoModProvider, providers/k8s)
	@$(call testGoModProvider, providers/azure)
	@$(call testGoModProvider, providers/ms365)
	@$(call testGoModProvider, providers/aws)
	@$(call testGoModProvider, providers/atlassian)
	@$(call testGoModProvider, providers/cloudformation)
	@$(call testGoModProvider, providers/nmap)

lr/test:
	go test ./resources/lr/...

# TODO: migrate
.PHONY: lr/docs/serve
lr/docs/serve:
	cd resources/docs && yarn
	cd resources/docs && $(shell cd resources/docs && npm bin)/parcel -p 1235 index.html

# TODO: migrate
.PHONY: lr/docs/markdown
lr/docs/markdown: providers/lr
	./lr markdown providers/arista/resources/arista.lr \
		--pack-name "Arista EOS" \
		--description "The Arista EOS resource pack lets you use MQL to query and assess the security of your Arista EOS network devices." \
		--docs-file providers/arista/resources/arista.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/arista-pack
	./lr markdown providers/atlassian/resources/atlassian.lr \
		--pack-name "Atlassian" \
		--description "The Atlassian resource pack lets you use MQL to query and assess the security of your Atlassian services." \
		--docs-file providers/atlassian/resources/atlassian.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/atlassian-pack
	./lr markdown providers/aws/resources/aws.lr \
    	--pack-name "Amazon Web Services (AWS)" \
		--description "The Amazon Web Services (AWS) resource pack lets you use MQL to query and assess the security of your AWS cloud services." \
		--docs-file providers/aws/resources/aws.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/aws-pack
	./lr markdown providers/azure/resources/azure.lr \
		--pack-name "Azure" \
		--description "The Azure resource pack lets you use MQL to query and assess the security of your Azure cloud services." \
		--docs-file providers/azure/resources/azure.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/azure-pack
	./lr markdown providers/cloudformation/resources/cloudformation.lr \
		--pack-name "AWS CloudFormation" \
		--description "The AWS CloudFormation resource pack lets you use MQL to query and assess the security of your AWS CloudFormation." \
		--docs-file providers/cloudformation/resources/cloudformation.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/cloudformation-pack
	./lr markdown providers/core/resources/core.lr \
		--pack-name "Core" \
		--description "The Core pack provides basic MQL resources that let you query and assess the security of assets in your infrastructure." \
		--docs-file providers/core/resources/core.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/core-pack
	./lr markdown providers/equinix/resources/equinix.lr \
		--pack-name "Equinix" \
		--description "The Equinix resource pack lets you use MQL to query and assess the security of your Equinix Metal assets." \
		--docs-file providers/equinix/resources/equinix.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/equinix-pack
	./lr markdown providers/gcp/resources/gcp.lr \
		--pack-name "Google Cloud Platform (GCP)" \
		--description "The Google Cloud Platform (GCP) resource pack lets you use MQL to query and assess the security of your Google cloud services." \
		--docs-file providers/gcp/resources/gcp.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gcp-pack
	./lr markdown providers/github/resources/github.lr \
		--pack-name "GitHub" \
		--description "The GitHub resource pack lets you use MQL to query and assess the security of your GitHub organizations and repositories." \
		--docs-file providers/github/resources/github.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/github-pack
	./lr markdown providers/gitlab/resources/gitlab.lr \
		--pack-name "GitLab" \
		--description "The GitLab resource pack lets you use MQL to query and assess the security of your GitLab groups and projects." \
		--docs-file providers/gitlab/resources/gitlab.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gitlab-pack
	./lr markdown providers/google-workspace/resources/google-workspace.lr \
		--pack-name "Google Workspace" \
		--description "The Google Workspace resource pack lets you use MQL to query and assess the security of your Google Workspace identities and configuration." \
		--docs-file providers/google-workspace/resources/google-workspace.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/google-workspace-pack
	./lr markdown providers/ipmi/resources/ipmi.lr \
		--pack-name "IPMI" \
		--description "The IPMI resource pack lets you use MQL to query and assess the security of your IPMI devices." \
		--docs-file providers/ipmi/resources/ipmi.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/ipmi-pack
	./lr markdown providers/k8s/resources/k8s.lr \
		--pack-name "Kubernetes (K8s)" \
		--description "The Kubernetes resource pack lets you use MQL to query and assess the security of your Kubernetes clusters and workloads." \
		--docs-file providers/k8s/resources/k8s.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/k8s-pack
	./lr markdown providers/ms365/resources/ms365.lr \
		--pack-name "Microsoft 365 (MS365)" \
		--description "The Microsoft 365 (MS365) resource pack lets you use MQL to query and assess the security of your Microsoft 365 identities and configuration." \
		--docs-file providers/ms365/resources/ms365.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/ms365-pack
	./lr markdown providers/network/resources/network.lr \
		--pack-name "Network" \
		--description "The Network resource pack lets you use MQL to query and assess the security of domains and network services." \
		--docs-file providers/network/resources/network.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/network-pack
	./lr markdown providers/network/resources/nmap.lr \
		--pack-name "nmap" \
		--description "The Nmap resource pack lets you use MQL to query and assess Nmap data." \
		--docs-file providers/network/resources/nmap.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/nmap-pack
	./lr markdown providers/oci/resources/oci.lr \
		--pack-name "Oracle Cloud Infrastructure (OCI)" \
		--description "The Oracle Cloud Infrastructure (OCI) resource pack lets you use MQL to query and assess the security of your OCI services." \
		--docs-file providers/oci/resources/oci.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/oci-pack
	./lr markdown providers/okta/resources/okta.lr \
		--pack-name "Okta" \
		--description "The Okta resource pack lets you use MQL to query and assess the security of your Okta identities and configuration." \
		--docs-file providers/okta/resources/okta.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/okta-pack
	./lr markdown providers/opcua/resources/opcua.lr \
		--pack-name "OPC UA" \
		--description "The OPC UA resource pack lets you use MQL to query and assess the security of your OPC UA assets." \
		--docs-file providers/opcua/resources/opcua.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/opcua-pack
	./lr markdown providers/os/resources/os.lr \
		--pack-name "Operating Systems (OS)" \
		--description "The Operating Systems (OS) resource pack lets you use MQL to query and assess the security of your operating system packages and configuration." \
		--docs-file providers/os/resources/os.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/os-pack
	./lr markdown providers/slack/resources/slack.lr \
		--pack-name "Slack" \
		--description "The Slack resource pack lets you use MQL to query and assess the security of your Slack identities and configuration." \
		--docs-file providers/slack/resources/slack.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/slack-pack
	./lr markdown providers/terraform/resources/terraform.lr \
		--pack-name "Terraform IaC" \
		--description "The Terraform IaC resource pack lets you use MQL to query and assess the security of your Terraform HCL, plan, and state resources." \
		--docs-file providers/terraform/resources/terraform.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/terraform-pack
	./lr markdown providers/vcd/resources/vcd.lr \
		--pack-name "VMware Cloud Director" \
		--description "The VMware Cloud Director resource pack lets you use MQL to query and assess the security of your VMware Cloud Director configuration." \
		--docs-file providers/vcd/resources/vcd.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/vcd-pack
	./lr markdown providers/vsphere/resources/vsphere.lr \
		--pack-name "VMware vSphere" \
		--description "The VMware vSphere resource pack lets you use MQL to query and assess the security of your VMware vSphere hosts and services." \
		--docs-file providers/vsphere/resources/vsphere.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/vsphere-pack

llx/generate:
	go generate ./llx

.PHONY: llx
llx: | llx/generate llx/test

llx/test:
	go test -timeout 5s $(shell go list ./llx/... | grep -v '/vendor/')

.PHONY: mqlc
mqlc: | llx mqlc/test

mqlc/test:
	go test -timeout 5s $(shell go list ./mqlc/... | grep -v '/vendor/')

explorer/generate:
	go generate ./explorer
	go generate ./explorer/scan
	go generate ./explorer/resources

sbom/generate:
	go generate ./sbom

reporter/generate:
	go generate ./cli/reporter

#   🏗 Binary / Build   #

.PHONY: cnquery/build
cnquery/build:
	go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/build/linux
cnquery/build/linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/build/windows
cnquery/build/windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/build/darwin:
	GOOS=darwin go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/install
cnquery/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/dist/goreleaser/stable:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m

cnquery/dist/goreleaser/edge:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m --snapshot

shared/generate:
	go generate ./shared/proto/.
	go generate ./providers-sdk/v1/upstream/
	go generate ./providers-sdk/v1/upstream/health
	go generate ./providers-sdk/v1/upstream/mvd/cvss
	go generate ./providers-sdk/v1/upstream/mvd

#   ⛹🏽‍ Testing   #

test/lint: test/lint/golangci-lint/run

test: test/go test/lint

benchmark/go:
	go test -bench=. -benchmem go.mondoo.com/cnquery/v11/explorer/scan/benchmark

test/generate: prep/tools/mockgen
	go generate ./providers

test/go: cnquery/generate test/generate test/go/plain

test/go/plain:
	go test -cover $(shell go list ./... | grep -v '/providers/' | grep -v '/test/')

test/go/plain-ci: prep/tools test/generate providers/build
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/' | grep -v '/providers/' | grep -v '/test/')

test/integration:
	go test -cover $(shell go list ./... | grep '/test/')

test/go-cli/plain-ci: prep/tools test/generate providers/build
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep '/test/')

.PHONY: test/lint/staticcheck
test/lint/staticcheck:
	staticcheck $(shell go list ./... | grep -v /providers/slack)

.PHONY: test/lint/govet
test/lint/govet:
	go vet $(shell go list ./... | grep -v /providers/slack)

.PHONY: test/lint/golangci-lint/run
test/lint/golangci-lint/run: prep/tools
	golangci-lint --version
	golangci-lint run

test/lint/extended: prep/tools test/generate
	golangci-lint run --config=.github/.golangci.yml --timeout=30m

test/lint/proto: prep/tools/protolint
	protolint lint .

license: license/headers/check

license/headers/check:
	copywrite headers --plan

license/headers/apply:
	copywrite headers
