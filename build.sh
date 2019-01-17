#!/bin/sh

# exit if any step fails
set -e

# change current dir to the script dir
cd "$(dirname "$0")"

GIT_VERSION="$(git describe)"
GIT_COMMIT="$(git rev-parse HEAD)"
GIT_COMMIT_SHORT="$(git rev-parse --short HEAD)"

# build service with version information
go get -ldflags "-X main.GitVersion=${GIT_VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitCommitShort=${GIT_COMMIT_SHORT}"
