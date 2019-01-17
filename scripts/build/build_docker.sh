#!/bin/sh

# exit if any step fails
set -e

# change current dir to the script dir
cd "$(dirname "$0")"
cd ../..

docker build -f docker/multistage/Dockerfile -t request-baskets .
