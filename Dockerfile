FROM alpine:3.16
RUN apk update &&\
    apk add ca-certificates wget tar &&\
    rm -rf /var/cache/apk/*
COPY cnquery /usr/local/bin
ENTRYPOINT ["cnquery"]
CMD ["help"]