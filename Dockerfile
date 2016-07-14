# Docker file to build Request Baskets service
# Version 1.0

MAINTAINER Vladimir L, vladimir_l@gmx.net

# From latest onbuild (currently "golang:1.6-onbuild")
FROM golang:onbuild

# Expose ports
EXPOSE 55555
