#!/bin/sh

# exit if any step fails
set -e

# change current dir to the script dir and then switch to project root
cd "$(dirname "$0")"
cd ../..

# collect git version information
GIT_VERSION="$(git describe --dirty='*')"
GIT_COMMIT="$(git rev-parse HEAD)"
GIT_COMMIT_SHORT="$(git rev-parse --short HEAD)"

# build service with version information from project root for Linux
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.GitVersion=${GIT_VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitCommitShort=${GIT_COMMIT_SHORT}"
