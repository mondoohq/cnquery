# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1

FROM alpine:3.20 AS root
RUN apk update &&\
    apk add ca-certificates wget tar &&\
    rm -rf /var/cache/apk/*
COPY cnquery /usr/local/bin
ENTRYPOINT ["cnquery"]
CMD ["help"]

# Rootless version of the container
FROM root AS rootless

RUN addgroup -S mondoo && adduser -S -G mondoo mondoo
USER mondoo