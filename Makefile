NAMESPACE=mondoo

ifndef LATEST_VERSION_TAG
# echo "read LATEST_VERSION_TAG from git"
LATEST_VERSION_TAG=$(shell git describe --abbrev=0 --tags)
endif

ifndef MANIFEST_VERSION
# echo "read MANIFEST_VERSION from git"
MANIFEST_VERSION=$(shell git describe --abbrev=0 --tags)
endif

ifndef VERSION
# echo "read VERSION from git"
VERSION=${LATEST_VERSION_TAG}+$(shell git rev-list --count HEAD)
endif
MAJOR_VERSION=v13

ifndef TARGETOS
	TARGETOS = $(shell go env GOOS)
endif

ifndef TARGETARCH
	TARGETARCH = $(shell go env GOARCH)
endif

BIN_SUFFIX = ""
ifeq ($(TARGETOS),windows)
	BIN_SUFFIX=".exe"
endif

LDFLAGS=-ldflags "-s -w -X go.mondoo.com/mql/${MAJOR_VERSION}.Version=${VERSION}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.com/mql/${MAJOR_VERSION}.Version=${LATEST_VERSION_TAG} -s -w"

.PHONY: info/ldflags
info/ldflags:
	$(info go run ${LDFLAGS} apps/mql/mql.go)
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
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/hashicorp/copywrite@latest

prep/tools/mockgen:
	go install go.uber.org/mock/mockgen@latest

#   🌙 MQL/MOTOR   #

mql/generate: clean/proto llx/generate shared/generate sbom/generate reporter/generate providers

mql/generate/core: clean/proto llx/generate shared/generate providers/proto providers/build/mock providers/build/core sbom/generate reporter/generate

define buildProvider
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_DIST_BIN = "./dist/${$@_NAME}")
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	echo "--> [${$@_NAME}] process resources"
	./lr go ${$@_HOME}/resources/${$@_NAME}.lr --dist ${$@_DIST}
	./lr versions ${$@_HOME}/resources/${$@_NAME}.lr
	echo "--> [${$@_NAME}] generate CLI json"
	cd ${$@_HOME} && go run ./gen/main.go .
	@if echo "aws gcp azure" | grep -qw "${$@_NAME}"; then \
		echo "--> [${$@_NAME}] extract permissions"; \
		go run providers-sdk/v1/util/permissions/permissions.go ${$@_HOME}; \
	fi
	@if [ "$(SKIP_COMPILE)" = "yes" ]; then \
		echo "--> [${$@_NAME}] skipping compile"; \
	else \
		echo "--> [${$@_NAME}] creating ${$@_BIN}"; \
		cd ${$@_HOME} && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ${$@_DIST_BIN}${BIN_SUFFIX} ./main.go; \
	fi
endef

define buildProviderDist
	$(eval $@_HOME = $(1))
	$(eval $@_NAME = $(shell basename ${$@_HOME}))
	$(eval $@_DIST = "${$@_HOME}"/dist)
	$(eval $@_DIST_BIN = "./dist/${$@_NAME}")
	$(eval $@_BIN = "${$@_DIST}"/"${$@_NAME}")
	echo "--> [${$@_NAME}] process resources"
	./lr go ${$@_HOME}/resources/${$@_NAME}.lr --dist ${$@_DIST}
	./lr versions ${$@_HOME}/resources/${$@_NAME}.lr
	echo "--> [${$@_NAME}] generate CLI json"
	cd ${$@_HOME} && go run ./gen/main.go .
	@if echo "aws gcp azure" | grep -qw "${$@_NAME}"; then \
		echo "--> [${$@_NAME}] extract permissions"; \
		go run providers-sdk/v1/util/permissions/permissions.go ${$@_HOME}; \
	fi
	echo "--> [${$@_NAME}] creating ${$@_BIN}"
	cd ${$@_HOME} && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ${LDFLAGSDIST} -o ${$@_DIST_BIN}${BIN_SUFFIX} ./main.go
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
	go generate ./providers-sdk/v1/recording

.PHONY: providers/config
providers/config:
	go run ./providers-sdk/v1/util/configure -f providers.yaml -o providers/builtin_dev.go

.PHONY: providers/defaults
providers/defaults:
	go run ./providers-sdk/v1/util/defaults/defaults.go -o providers/defaults.go

.PHONY: providers/lr
providers/lr:
	go build -o lr ./providers-sdk/v1/mqlr/main.go

providers/lr/install: providers/lr
	cp ./lr ${GOPATH}/bin

.PHONY: providers/mqlr
providers/mqlr:
	go build -o mqlr ./providers-sdk/v1/mqlr/main.go

providers/mqlr/install: providers/mqlr
	cp ./mqlr ${GOPATH}/bin

# Provider list — add new providers here.
# core is excluded: it has no install target and is always built as a dependency of providers/build.
PROVIDERS := \
	ansible \
	arista \
	atlassian \
	aws \
	azure \
	cloudflare \
	cloudformation \
	depsdev \
	equinix \
	gcp \
	github \
	gitlab \
	google-workspace \
	ipinfo \
	ipmi \
	k8s \
	mondoo \
	ms365 \
	network \
	nmap \
	oci \
	okta \
	opcua \
	os \
	shodan \
	slack \
	snowflake \
	tailscale \
	terraform \
	vcd \
	vsphere

.PHONY: providers/build
providers/build: \
	providers/build/mock \
	providers/build/core \
	$(addprefix providers/build/,$(PROVIDERS))

.PHONY: providers/install
providers/install: $(addprefix providers/install/,$(PROVIDERS))

providers/build/mock: providers/lr
	./lr go providers-sdk/v1/testutils/mockprovider/resources/mockprovider.lr

providers/build/%: providers/lr
	@$(call buildProvider, providers/$*)

providers/install/%:
	@$(call installProvider, providers/$*)

providers/dist: $(addprefix providers/dist/,$(PROVIDERS))
providers/dist/%:
	@$(call buildProviderDist, providers/$*)

providers/bundle: $(addprefix providers/bundle/,$(PROVIDERS))
providers/bundle/%:
	@$(call bundleProvider, providers/$*)

providers/permissions:
	@go run providers-sdk/v1/util/permissions/permissions.go providers/aws
	@go run providers-sdk/v1/util/permissions/permissions.go providers/gcp
	@go run providers-sdk/v1/util/permissions/permissions.go providers/azure

providers/test:
	@$(call testProvider, providers/core)
	@$(call testProvider, providers/network)
	@$(call testProvider, providers/os)
	@$(call testGoModProvider, providers/ansible)
	@$(call testGoModProvider, providers/arista)
	@$(call testGoModProvider, providers/atlassian)
	@$(call testGoModProvider, providers/aws)
	@$(call testGoModProvider, providers/azure)
	@$(call testGoModProvider, providers/cloudflare)
	@$(call testGoModProvider, providers/cloudformation)
	@$(call testGoModProvider, providers/depsdev)
	@$(call testGoModProvider, providers/equinix)
	@$(call testGoModProvider, providers/gcp)
	@$(call testGoModProvider, providers/github)
	@$(call testGoModProvider, providers/gitlab)
	@$(call testGoModProvider, providers/google-workspace)
	@$(call testGoModProvider, providers/ipinfo)
	@$(call testGoModProvider, providers/ipmi)
	@$(call testGoModProvider, providers/k8s)
	@$(call testGoModProvider, providers/mondoo)
	@$(call testGoModProvider, providers/ms365)
	@$(call testGoModProvider, providers/nmap)
	@$(call testGoModProvider, providers/oci)
	@$(call testGoModProvider, providers/okta)
	@$(call testGoModProvider, providers/opcua)
	@$(call testGoModProvider, providers/shodan)
	@$(call testGoModProvider, providers/slack)
	@$(call testGoModProvider, providers/snowflake)
	@$(call testGoModProvider, providers/tailscale)
	@$(call testGoModProvider, providers/terraform)
	@$(call testGoModProvider, providers/vcd)
	@$(call testGoModProvider, providers/vsphere)

lr/test:
	go test ./providers-sdk/v1/mqlr/...

# TODO: migrate
.PHONY: lr/docs/serve
lr/docs/serve:
	cd resources/docs && yarn
	cd resources/docs && $(shell cd resources/docs && npm bin)/parcel -p 1235 index.html

lr/docs/stats:
	@echo "Please remember to re-run before using this:"
	@echo "  make providers/build"
	@echo ""
	go run providers-sdk/v1/util/docs/summarize.go ${PWD}/providers

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

sbom/generate:
	go generate ./sbom

reporter/generate:
	go generate ./cli/reporter

#   🏗 Binary / Build   #

.PHONY: mql/build
mql/build:
	go build ${LDFLAGSDIST} apps/mql/mql.go

.PHONY: mql/build/linux
mql/build/linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGSDIST} apps/mql/mql.go

.PHONY: mql/build/windows
mql/build/windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGSDIST} apps/mql/mql.go

mql/build/darwin:
	GOOS=darwin go build ${LDFLAGSDIST} apps/mql/mql.go

.PHONY: mql/install
mql/install:
	GOBIN=${GOPATH}/bin go install ${LDFLAGSDIST} apps/mql/mql.go

mql/dist/goreleaser/stable:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m

mql/dist/goreleaser/edge:
	goreleaser release --clean --skip=validate,publish -f .goreleaser.yml --timeout 120m --snapshot

shared/generate:
	go generate ./shared/proto/.
	go generate ./providers-sdk/v1/upstream/
	go generate ./providers-sdk/v1/upstream/health
	go generate ./providers-sdk/v1/upstream/mvd/cvss
	go generate ./providers-sdk/v1/upstream/mvd
	go generate ./providers-sdk/v1/upstream/etl

#   ⛹🏽‍ Testing   #

test/lint: test/lint/golangci-lint/run

test: test/go test/lint

race/go:
	go test -race go.mondoo.com/mql/${MAJOR_VERSION}/internal/workerpool

test/generate: prep/tools/mockgen
	go generate ./providers/...

test/go: mql/generate test/generate test/go/plain

test/go/plain:
	go test -cover $(shell go list ./... | grep -v '/providers/' | grep -v '/test/')

test/go/plain-ci: prep/tools test/generate providers/build
	gotestsum --junitfile report.xml --format pkgname -- -cover $(shell go list ./... | grep -v '/vendor/' | grep -v '/providers/' | grep -v '/test/')

test/integration:
	go test -cover -p 1 $(shell go list ./... | grep '/test/')

test/go-cli/plain-ci: prep/tools test/generate providers/build
	gotestsum --junitfile report.xml --format pkgname -- -cover -p 1 $(shell go list ./... | grep '/test/')

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
	golangci-lint run --config=.github/.golangci.yaml --timeout=30m

test/lint/proto: prep/tools/protolint
	protolint lint .

license: license/headers/check

license/headers/check:
	copywrite headers --plan

license/headers/apply:
	copywrite headers

#   📈 METRICS       #

metrics/start: metrics/grafana/start metrics/prometheus/start

metrics/prometheus/start:
	APP_NAME=mql VERSION=${VERSION} prometheus --config.file=prometheus.yml

metrics/grafana/start:
	docker run -d --name=grafana \
		-p 3000:3000               \
		grafana/grafana

metrics/grafana/stop:
	docker stop grafana
