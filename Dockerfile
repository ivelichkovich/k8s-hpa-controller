FROM alpine:3.5


RUN apk --update --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

COPY dist/hpa-controller /

WORKDIR /

ENTRYPOINT ["/hpa-controller"]
