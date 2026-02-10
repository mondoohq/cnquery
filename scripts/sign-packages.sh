#!/bin/bash
set -e

# Sign RPMs using the existing GPG setup
if ls dist/*.rpm 1> /dev/null 2>&1; then
  echo "Signing RPM packages with Docker..."
  docker run --rm \
    -v $(pwd):/workspace \
    -v "$GPG_KEY_PATH":/tmp/signing-key \
    -e RPM_PASSPHRASE="$NFPM_DEFAULT_RPM_PASSPHRASE" \
    registry.fedoraproject.org/fedora:latest \
    sh -c "
      dnf install -y rpm-sign
      gpg --import /tmp/signing-key
      echo 'allow-loopback-pinentry' >> ~/.gnupg/gpg-agent.conf
      find /workspace/dist -name '*.rpm' -exec rpmsign --addsign {} \;
    "
fi

# # Sign DEBs (if you want both)
# if ls dist/*.deb 1> /dev/null 2>&1; then
#   echo "Signing DEB packages with Docker..."
#   docker run --rm \
#     -v $(pwd):/workspace \
#     -v "$GPG_KEY_PATH":/tmp/signing-key \
#     ubuntu:latest \
#     sh -c "
#       apt-get update && apt-get install -y dpkg-sig gnupg
#       gpg --import /tmp/signing-key
#       find /workspace/dist -name '*.deb' -exec dpkg-sig --sign builder {} \;
#     "
# fi

# Regenerate checksums
goreleaser checksum --config .github/.goreleaser-unstable.yml