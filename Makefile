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

LDFLAGS=-ldflags "-s -w -X go.mondoo.com/cnquery.Version=${VERSION} -X go.mondoo.com/cnquery.Build=${TAG}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.com/cnquery.Version=${LATEST_VERSION_TAG} -X go.mondoo.com/cnquery.Build=${TAG} -s -w"

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

prep/tools:
	# protobuf tooling
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc@latest
	go install go.mondoo.com/ranger-rpc/protoc-gen-rangerrpc-swagger@latest
	# additional helper
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest


#   🌙 MQL/MOTOR   #

cnquery/generate: clean/proto motor/generate resources/generate llx/generate shared/generate explorer/generate

define genProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	go run "${$@_HOME}"/gen/main.go "${$@_HOME}"
	echo "--> process resources for ${$@_NAME}"
	./lr go "${$@_HOME}"/resources/${$@_NAME}.lr --dist "${$@_DIST}"
	./lr docs json "${$@_HOME}"/resources/${$@_NAME}.lr.manifest.yaml --dist "${$@_DIST}"
	echo "--> creating ${$@_BIN}"
	go build -o "${$@_BIN}" "${$@_HOME}"/main.go
endef

.PHONY: providers
providers:
	go generate .
	go generate ./providers-sdk/v1/vault
	go generate ./providers-sdk/v1/resources
	go generate ./providers-sdk/v1/inventory
	go generate ./providers-sdk/v1/plugin
	go build -o lr ./providers-sdk/v1/lr/cli/main.go
	@$(call genProvider, providers/core)
	@$(call genProvider, providers/os)
# add more providers...

motor/generate:
	go generate .
	go generate ./motor/providers/k8s
	go generate ./motor/platform

motor/test:
	gotestsum -f short-verbose $(shell go list ./motor/...)

lr/test:
	go test ./resources/lr/...

# TODO: migrate
.PHONY: lr/docs/serve
lr/docs/serve:
	cd resources/docs && yarn
	cd resources/docs && $(shell cd resources/docs && npm bin)/parcel -p 1235 index.html

# TODO: migrate
.PHONY: lr/docs/markdown
lr/docs/markdown: lr/build
	./lr markdown resources/packs/aws/aws.lr \
    --pack-name "Amazon Web Services (AWS)" \
		--description "The Amazon Web Services (AWS) resource pack lets you use MQL to query and assess the security of your AWS cloud services." \
		--docs-file resources/packs/aws/aws.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/aws-pack
	./lr markdown resources/packs/azure/azure.lr \
		--pack-name "Azure" \
		--description "The Azure resource pack lets you use MQL to query and assess the security of your Azure cloud services." \
		--docs-file resources/packs/azure/azure.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/azure-pack
	./lr markdown resources/packs/core/core.lr \
		--pack-name "Core" \
		--description "The Core pack provides basic MQL resources that let you query and assess the security." \
		--docs-file resources/packs/core/core.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/core-pack
	./lr markdown resources/packs/gcp/gcp.lr \
		--pack-name "Google Cloud Platform (GCP)" \
		--description "The Google Cloud Platform (GCP) resource pack lets you use MQL to query and assess the security of your GCP cloud services." \
		--docs-file resources/packs/gcp/gcp.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gcp-pack
	./lr markdown resources/packs/github/github.lr \
		--pack-name "GitHub" \
		--description "The GitHub resource pack lets you use MQL to query and assess the security of your GitHub organization and repositories." \
		--docs-file resources/packs/github/github.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/github-pack
	./lr markdown resources/packs/gitlab/gitlab.lr \
		--pack-name "GitLab" \
		--description "The GitLab resource pack lets you use MQL to query and assess the security of your GitLab organization and repositories." \
		--docs-file resources/packs/gitlab/gitlab.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gitlab-pack
	./lr markdown resources/packs/k8s/k8s.lr \
		--pack-name "Kubernetes (K8s)" \
		--description "The Kubernetes resource pack lets you use MQL to query and assess the security of your Kubernetes workloads." \
		--docs-file resources/packs/k8s/k8s.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/k8s-pack
	./lr markdown resources/packs/ms365/ms365.lr \
		--pack-name "Microsoft 365 (MS365)" \
		--description "The Microsoft 365 (MS365) resource pack lets you use MQL to query and assess the security of your MS365 identities and configuration." \
		--docs-file resources/packs/ms365/ms365.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/ms365-pack
	./lr markdown resources/packs/os/os.lr \
		--pack-name "Operating Systems (OS)" \
		--description "The Operating Systems (OS) resource pack lets you use MQL to query and assess the security of your operating system packages and configuration." \
		--docs-file resources/packs/os/os.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/os-pack
	./lr markdown resources/packs/terraform/terraform.lr \
		--pack-name "Terraform IaC" \
		--description "The Terraform IaC resource pack lets you use MQL to query and assess the security of your Terraform HCL, plan and state resources." \
		--docs-file resources/packs/terraform/terraform.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/terraform-pack
	./lr markdown resources/packs/vsphere/vsphere.lr \
		--pack-name "VMware vSphere" \
		--description "The VMware vSphere resource pack lets you use MQL to query and assess the security of your VMware vSphere hosts and services." \
		--docs-file resources/packs/vsphere/vsphere.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/vsphere-pack
	./lr markdown resources/packs/okta/okta.lr \
		--pack-name "Okta" \
		--description "The Okta resource pack lets you use MQL to query and assess the security of your Okta identities and configuration." \
		--docs-file resources/packs/okta/okta.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/okta-pack
	./lr markdown resources/packs/googleworkspace/googleworkspace.lr \
		--pack-name "Google Workspace" \
		--description "The Google Workspace resource pack lets you use MQL to query and assess the security of your Google Workspace identities and configuration." \
		--docs-file resources/packs/googleworkspace/googleworkspace.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/googleworkspace-pack
	./lr markdown resources/packs/slack/slack.lr \
		--pack-name "Slack" \
		--description "The Slack resource pack lets you use MQL to query and assess the security of your Slack identities and configuration." \
		--docs-file resources/packs/slack/slack.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/slack-pack
	./lr markdown resources/packs/vcd/vcd.lr \
		--pack-name "VMware Cloud Director" \
		--description "The VMware Cloud Director resource pack lets you use MQL to query and assess the security of your VMware Cloud Director configuration." \
		--docs-file resources/packs/vcd/vcd.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/vcd-pack
	./lr markdown resources/packs/arista/arista.lr \
		--pack-name "Arista EOS" \
		--description "The Arista EOS resource pack lets you use MQL to query and assess the security of your Arista EOS network devices." \
		--docs-file resources/packs/arista/arista.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/arista-pack
	./lr markdown resources/packs/ipmi/ipmi.lr \
		--pack-name "IPMI" \
		--description "The IPMI resource pack lets you use MQL to query and assess the security of your IPMI devices." \
		--docs-file resources/packs/ipmi/ipmi.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/ipmi-pack
	./lr markdown resources/packs/oci/oci.lr \
		--pack-name "Oracle Cloud Infrastructure (OCI)" \
		--description "The Oracle Cloud Infrastructure (OCI) resource pack lets you use MQL to query and assess the security of your OCI cloud services." \
		--docs-file resources/packs/oci/oci.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/oci-pack
	./lr markdown resources/packs/opcua/opcua.lr \
		--pack-name "OPC UA" \
		--description "The OPC-UA resource pack lets you use MQL to query and assess the security of your OPC-UA servers." \
		--docs-file resources/packs/opcua/opcua.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/opcua-pack

.PHONY: resources
resources: | lr resources/generate resources/test

resources/generate:
	go generate ./resources
	go generate ./resources/service
	go generate ./resources/packs/core/vadvisor
	go generate ./resources/packs/core/vadvisor/cvss

resources/test:
	go test -timeout 5s $(shell go list ./resources/... | grep -v '/vendor/')

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

#   🏗 Binary / Build   #

.PHONY: cnquery/build
cnquery/build:
	go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/build/linux
cnquery/build/linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/build/windows
cnquery/build/windows:
	GOOS=windows go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/build/darwin:
	GOOS=darwin go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/install
cnquery/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/dist/goreleaser/stable:
	goreleaser release --clean --skip-publish --skip-validate	-f .goreleaser.yml --timeout 120m

cnquery/dist/goreleaser/edge:
	goreleaser release --clean --skip-publish --skip-validate	-f .goreleaser.yml --timeout 120m --snapshot

shared/generate:
	go generate ./shared/proto/.
	go generate ./upstream/
	go generate ./upstream/health
	go generate ./upstream/mvd/cvss
	go generate ./upstream/mvd

#   ⛹🏽‍ Testing   #

test/lint: test/lint/golangci-lint/run

test: test/go test/lint

test/go: cnquery/generate test/go/plain

test/go/plain:
	# TODO /motor/docker/docker_engine cannot be executed inside of docker
	go test -cover $(shell go list ./... | grep -v '/motor/discovery/docker_engine')

test/go/plain-ci: prep/tools
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/' | grep -v '/motor/discovery/docker_engine')

.PHONY: test/lint/staticcheck
test/lint/staticcheck:
	staticcheck $(shell go list ./... | grep -v /ent/ | grep -v /benchmark/)

.PHONY: test/lint/govet
test/lint/govet:
	go vet $(shell go list ./... | grep -v /ent/ | grep -v /benchmark/)

.PHONY: test/lint/golangci-lint/run
test/lint/golangci-lint/run: prep/tools
	golangci-lint --version
	golangci-lint run
