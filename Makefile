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

LDFLAGS=-ldflags "-s -w -X go.mondoo.io/mondoo.Version=${VERSION} -X go.mondoo.io/mondoo.Build=${TAG}" # -linkmode external -extldflags=-static
LDFLAGSDIST=-tags production -ldflags "-s -w -X go.mondoo.io/mondoo.Version=${LATEST_VERSION_TAG} -X go.mondoo.io/mondoo.Build=${TAG} -s -w"

.PHONY: info/ldflags
info/ldflags:
	$(info go run ${LDFLAGS} apps/mondoo/mondoo.go)
	@:

#   🧹 CLEAN   #

clean/proto:
	find . -not -path './.*' \( -name '*.ranger.go' -or -name '*.pb.go' -or -name '*.actions.go' -or -name '*-packr.go' -or -name '*.swagger.json' \) -dele
te

.PHONY: version
version:
	@echo $(VERSION)

#   🌙 MQL/MOTOR   #

cnquery/generate: clean/proto motor/generate resources/generate llx/generate lr

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
	go build -o lr resources/lr/cli/main.go
	./lr go resources/packs/core/core.lr
	go generate ./resources/packs/core/info
	./lr docs go resources/packs/core/core.lr.manifest.yaml
	go fmt resources/packs/core/core.lr.manifest.go
	./lr go resources/packs/os/os.lr
	go generate ./resources/packs/os/info
	./lr go resources/packs/aws/aws.lr
	go generate ./resources/packs/aws/info
	./lr go resources/packs/azure/azure.lr
	go generate ./resources/packs/azure/info
	./lr go resources/packs/gcp/gcp.lr
	go generate ./resources/packs/gcp/info
	./lr go resources/packs/ms365/ms365.lr
	go generate ./resources/packs/ms365/info
	./lr go resources/packs/github/github.lr
	go generate ./resources/packs/github/info
	./lr go resources/packs/gitlab/gitlab.lr
	go generate ./resources/packs/gitlab/info
	./lr go resources/packs/terraform/terraform.lr
	go generate ./resources/packs/terraform/info
	./lr go resources/packs/k8s/k8s.lr
	go generate ./resources/packs/k8s/info
	./lr go resources/packs/vsphere/vsphere.lr
	go generate ./resources/packs/vsphere/info
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
		--docs-file resources/packs/aws/aws.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/aws
	./lr markdown resources/packs/azure/azure.lr \
		--docs-file resources/packs/azure/azure.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/azure
	./lr markdown resources/packs/core/core.lr \
		--docs-file resources/packs/core/core.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/core
	./lr markdown resources/packs/gcp/gcp.lr \
		--docs-file resources/packs/gcp/gcp.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/gcp
	./lr markdown resources/packs/github/github.lr \
		--docs-file resources/packs/github/github.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/github
	./lr markdown resources/packs/gitlab/gitlab.lr \
		--docs-file resources/packs/gitlab/gitlab.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/gitlab
	./lr markdown resources/packs/k8s/k8s.lr \
		--docs-file resources/packs/k8s/k8s.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/k8s
	./lr markdown resources/packs/ms365/ms365.lr \
		--docs-file resources/packs/ms365/ms365.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/ms365
	./lr markdown resources/packs/os/os.lr \
		--docs-file resources/packs/os/os.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/os
	./lr markdown resources/packs/terraform/terraform.lr \
		--docs-file resources/packs/terraform/terraform.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/terraform
	./lr markdown resources/packs/vsphere/vsphere.lr \
		--docs-file resources/packs/vsphere/vsphere.lr.manifest.yaml \
		--front-matter-file ../web/docs/templates/mql-resources-front-matter.md \
		--output ../web/docs/docs/references/mql/vsphere

.PHONY: resources
resources: | lr resources/generate resources/test

resources/generate:
	go generate ./resources
	go generate ./resources/service

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

