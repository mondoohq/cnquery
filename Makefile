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

cnquery/generate: clean/proto motor/generate resources/generate llx/generate lr shared/generate explorer/generate

motor/generate:
	go generate .
	go generate ./motor/providers
	go generate ./motor/providers/k8s
	go generate ./motor/platform
	go generate ./motor/asset
	go generate ./motor/vault
	go generate ./motor/inventory/v1

motor/test:
	gotestsum -f short-verbose $(shell go list ./motor/...)

.PHONY: lr
lr: | lr/build lr/test

lr/build:
	export GOPRIVATE="github.com/mondoohq"
	go generate .
	go generate ./resources/packs/core/vadvisor/cvss
	go build -o lr resources/lr/cli/main.go
	./lr go resources/packs/core/core.lr
	./lr docs json resources/packs/core/core.lr.manifest.yaml
	./lr go resources/packs/os/os.lr
	./lr docs json resources/packs/os/os.lr.manifest.yaml
	./lr go resources/packs/aws/aws.lr
	./lr docs json resources/packs/aws/aws.lr.manifest.yaml
	./lr go resources/packs/azure/azure.lr
	./lr docs json resources/packs/azure/azure.lr.manifest.yaml
	./lr go resources/packs/gcp/gcp.lr
	./lr docs json resources/packs/gcp/gcp.lr.manifest.yaml
	./lr go resources/packs/ms365/ms365.lr
	./lr docs json resources/packs/ms365/ms365.lr.manifest.yaml
	./lr go resources/packs/github/github.lr
	./lr docs json resources/packs/github/github.lr.manifest.yaml
	./lr go resources/packs/gitlab/gitlab.lr
	./lr docs json resources/packs/gitlab/gitlab.lr.manifest.yaml
	./lr go resources/packs/terraform/terraform.lr
	./lr docs json resources/packs/terraform/terraform.lr.manifest.yaml
	./lr go resources/packs/k8s/k8s.lr
	./lr docs json resources/packs/k8s/k8s.lr.manifest.yaml
	./lr go resources/packs/vsphere/vsphere.lr
	./lr docs json resources/packs/vsphere/vsphere.lr.manifest.yaml
	./lr go resources/mock/mochi.lr

lr/release:
	export GOPRIVATE="github.com/mondoohq"
	go generate .
	go build -o lr resources/lr/cli/main.go
	./lr docs yaml resources/packs/core/core.lr --version ${MANIFEST_VERSION} --docs-file resources/packs/core/core.lr.manifest.yaml
	./lr docs go resources/packs/core/core.lr.manifest.yaml
	go fmt resources/packs/core/core.lr.manifest.go

lr/test:
	go test ./resources/lr/...

lr/docs: lr/build
	./lr parse resources/packs/core/core.lr > resources/docs/static/$(shell make version).json
	cd resources/docs/static && python -c 'import os, json; print json.dumps(os.listdir("."))' > snapshots.json
	cd resources/docs && node ./refresh_snapshots.js

.PHONY: lr/docs/serve
lr/docs/serve:
	cd resources/docs && yarn
	cd resources/docs && $(shell cd resources/docs && npm bin)/parcel -p 1235 index.html

.PHONY: lr/docs/markdown
lr/docs/markdown: lr/build
	./lr markdown resources/packs/aws/aws.lr \
        --pack-name "Amazon Web Services (AWS)" \
		--docs-file resources/packs/aws/aws.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/aws-pack
	./lr markdown resources/packs/azure/azure.lr \
		--pack-name "Azure" \
		--docs-file resources/packs/azure/azure.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/azure-pack
	./lr markdown resources/packs/core/core.lr \
		--pack-name "Core" \
		--docs-file resources/packs/core/core.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/core-pack
	./lr markdown resources/packs/gcp/gcp.lr \
		--pack-name "Google Cloud Platform (GCP)" \
		--docs-file resources/packs/gcp/gcp.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gcp-pack
	./lr markdown resources/packs/github/github.lr \
		--pack-name "GitHub" \
		--docs-file resources/packs/github/github.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/github-pack
	./lr markdown resources/packs/gitlab/gitlab.lr \
		--pack-name "GitLab" \
		--docs-file resources/packs/gitlab/gitlab.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/gitlab-pack
	./lr markdown resources/packs/k8s/k8s.lr \
		--pack-name "Kubernetes (K8s)" \
		--docs-file resources/packs/k8s/k8s.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/k8s-pack
	./lr markdown resources/packs/ms365/ms365.lr \
		--pack-name "Microsoft 365 (MS365)" \
		--docs-file resources/packs/ms365/ms365.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/ms365-pack
	./lr markdown resources/packs/os/os.lr \
		--pack-name "Operating Systems (OS)" \
		--docs-file resources/packs/os/os.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/os-pack
	./lr markdown resources/packs/terraform/terraform.lr \
		--pack-name "Terraform IaC" \
		--docs-file resources/packs/terraform/terraform.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/terraform-pack
	./lr markdown resources/packs/vsphere/vsphere.lr \
		--pack-name "VMware vSphere" \
		--docs-file resources/packs/vsphere/vsphere.lr.manifest.yaml \
		--output ../docs/docs/mql/resources/vsphere-pack

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

.PHONY: cnquery/build/linux
cnquery/build/linux:
	GOOS=linux go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/build/windows
cnquery/build/windows:
	GOOS=windows go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/build/darwin:
	GOOS=darwin go build ${LDFLAGSDIST} apps/cnquery/cnquery.go

.PHONY: cnquery/install
cnquery/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/cnquery/cnquery.go

cnquery/dist/goreleaser/stable:
	goreleaser release --rm-dist --skip-publish --skip-validate	-f .goreleaser.yml --timeout 120m

cnquery/dist/goreleaser/edge:
	goreleaser release --rm-dist --skip-publish --skip-validate	-f .goreleaser.yml --timeout 120m --snapshot

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
