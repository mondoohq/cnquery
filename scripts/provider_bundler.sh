#!/bin/bash
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1

## Build the provider and bundle it into a single file

REPOROOT=$(git rev-parse --show-toplevel)
PROVIDER_NAME=$1
PROVIDER_PATH=$REPOROOT/providers/$PROVIDER_NAME
PROVIDER_DIST=$PROVIDER_PATH/dist
BUNDLE_DIST=$REPOROOT/dist

cd $REPOROOT

if [ -z $PROVIDER_NAME ]; then
  echo "Please specify a provider name."
  exit 1
fi

# Check if the provider exists
if [ ! -d $PROVIDER_PATH ]; then
  echo "The ${PROVIDER_NAME} provider does not exist.  Please create it first."
  exit 1
fi

# Clean up the dist directory
if [ -d $PROVIDER_DIST ]; then
  echo "Previous build detected.  Cleaning up the ${PROVIDER_NAME} provider (${PROVIDER_PATH})..."
  rm -rf $PROVIDER_DIST
fi

# Create the dist directory
mkdir -p $PROVIDER_DIST
mkdir -p $BUNDLE_DIST

# Record the starting timestamp
START_TIME=$(date +%s)

# Extract the version from the provider
PROVIDER_VERSION=$(grep "Version:" ${PROVIDER_PATH}/config/config.go | cut -f2 -d\")

# Build the provider
echo "Building the ${PROVIDER_NAME} provider (Version: ${PROVIDER_VERSION})..."

# Build the non-binary files first
echo "  - Generate the plugin..."
cd ${PROVIDER_PATH} && go run gen/main.go .
echo "  - Compile the resources..."
${REPOROOT}/lr go ${PROVIDER_PATH}/resources/${PROVIDER_NAME}.lr --dist ${PROVIDER_DIST}
echo "  - Generate the resource docs..."
${REPOROOT}/lr docs json ${PROVIDER_PATH}/resources/${PROVIDER_NAME}.lr.manifest.yaml
#echo "  - Build the provider binary..."
#go build -o ${PROVIDER_DIST}/${PROVIDER_NAME} ${PROVIDER_PATH}/main.go

build_bundle(){
  GOOS=$1
  GOARCH=$2
  GOARM=$3

  echo "Building ${PROVIDER_DIST}/${PROVIDER_NAME} for ${GOOS}/${GOARCH}/${GOARM} ..."
  # we switch into the path to use the local go.mods
  PROVIDER_EXECUTABLE="${PROVIDER_NAME}"
  if [[ "${GOOS}" == "windows" ]]; then
    PROVIDER_EXECUTABLE="${PROVIDER_EXECUTABLE}.exe"
  fi
  cd ${PROVIDER_PATH} && CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} GOARM=${GOARM} go build -tags production -ldflags "-s -w" -o ${PROVIDER_DIST}/${PROVIDER_EXECUTABLE} main.go

  if [[ "${GOOS}" == "windows" ]]; then
    ### SIGN THE BINARY
    echo "  - Signing the binary ${PROVIDER_DIST}/${PROVIDER_EXECUTABLE}..."
    jsign --storetype DIGICERTONE --alias "${SM_CERT_ALIAS}" \
          --storepass "${SM_API_KEY}|${SM_CLIENT_CERT_FILE}|${SM_CLIENT_CERT_PASSWORD}" \
          --tsaurl "http://timestamp.digicert.com"  ${PROVIDER_DIST}/${PROVIDER_EXECUTABLE}
  fi

  # set linux flags that do not work on macos
  TAR_FLAGS=""
  if uname -s | grep -q 'Linux'; then
    TAR_FLAGS="--owner=0 --group=0 --no-same-owner"
  fi

  tar -cf ${BUNDLE_DIST}/${PROVIDER_NAME}_${PROVIDER_VERSION}_${GOOS}_${GOARCH}.tar.xz \
    ${TAR_FLAGS} --use-compress-program='xz -9v' \
    -C ${PROVIDER_DIST} \
    ${PROVIDER_EXECUTABLE} ${PROVIDER_NAME}.json ${PROVIDER_NAME}.resources.json

  if [ $? -ne 0 ]; then
    echo "Failed to build the ${PROVIDER_NAME} provider."
    exit 1
  fi

  rm ${PROVIDER_DIST}/${PROVIDER_EXECUTABLE}
}

# Build Darwin Architectures
build_bundle darwin amd64
build_bundle darwin arm64

# Build Linux Architectures
build_bundle linux amd64
build_bundle linux 386
build_bundle linux arm64
build_bundle linux arm 6
build_bundle linux arm 7
build_bundle linux ppc64le

# Build Windows Architectures
build_bundle windows amd64
build_bundle windows arm64

# Generate SHA256 checksums
echo "  - Generating SHA256 checksums..."
cd $BUNDLE_DIST
shasum -a 256 ${PROVIDER_NAME}_${PROVIDER_VERSION}_*.xz > ${PROVIDER_NAME}_${PROVIDER_VERSION}_SHA256SUMS

printf "\n\n"
echo "  The ${PROVIDER_NAME} provider has been built and bundled successfully."
echo "  The bundles can be found in ${BUNDLE_DIST}."
echo "  Time Elapsed: $((($(date +%s)-$START_TIME)/60)) minutes"
echo "-------------------------------------------------------"
echo ""