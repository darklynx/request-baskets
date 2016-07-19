#!/bin/sh

# exit if any step fails
set -e

# change current dir to the script dir
cd "$(dirname "$0")"

# build (via get, static link) request-baskets service with latest golang docker
docker run --rm -v "$PWD":/go/bin --env CGO_ENABLED=0 golang go get -v github.com/darklynx/request-baskets

# create a minimalistic docker image (based on alpine) with service only
docker build -t request-baskets .

# cleanup
rm request-baskets

# message
echo
echo "Request Basket service image is ready!"
echo "To start container as a service use following commands:"
echo
echo "    $ docker run --name rbaskets -d -p 55555:55555 request-baskets"
echo "    $ docker logs rbaskets"
