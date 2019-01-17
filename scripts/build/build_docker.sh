#!/bin/sh

# change current dir to the script dir and then switch to project root
cd "$(dirname "$0")"
cd ../..

# build docker container from project root
docker build -t request-baskets .
