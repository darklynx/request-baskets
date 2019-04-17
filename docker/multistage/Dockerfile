# Builds Request Baskets service using multi-stage builds
# Version 1.2

# Stage 1. Building service
FROM golang:latest as builder
WORKDIR /go/src/rbaskets
COPY . .
RUN GIT_VERSION="$(git describe --dirty='*' || git symbolic-ref -q --short HEAD)" \
  && GIT_COMMIT="$(git rev-parse HEAD)" \
  && GIT_COMMIT_SHORT="$(git rev-parse --short HEAD)" \
  && set -x \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.GitVersion=${GIT_VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitCommitShort=${GIT_COMMIT_SHORT}" \
    -o /go/bin/rbaskets

# Stage 2. Packaging into alpine
FROM alpine:latest
MAINTAINER Vladimir L, vladimir_l@gmx.net
RUN apk --no-cache add ca-certificates
VOLUME /var/lib/rbaskets
COPY docker/entrypoint.sh /bin/entrypoint
COPY --from=builder /go/bin/rbaskets /bin/rbaskets
EXPOSE 55555
CMD /bin/entrypoint
