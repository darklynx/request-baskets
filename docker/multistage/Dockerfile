# Builds Request Baskets service using multi-stage builds
# Version 1.3

# Stage 1. Building service
FROM --platform=$BUILDPLATFORM golang:latest as builder
WORKDIR /go/src/rbaskets
COPY . .
ARG TARGETARCH
ARG GIT_VERSION
ARG GIT_COMMIT
ARG GIT_COMMIT_SHORT

RUN set -x \
  && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -installsuffix cgo \
  -ldflags="-w -s -X main.GitVersion=${GIT_VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitCommitShort=${GIT_COMMIT_SHORT}" \
  -o /go/bin/rbaskets

# Stage 2. Packaging into alpine
FROM --platform=$TARGETPLATFORM alpine:latest
LABEL maintainer="vladimir_l@gmx.net"
RUN apk --no-cache add ca-certificates
VOLUME /var/lib/rbaskets
COPY docker/entrypoint.sh /bin/entrypoint
COPY --from=builder /go/bin/rbaskets /bin/rbaskets
EXPOSE 55555
CMD /bin/entrypoint
