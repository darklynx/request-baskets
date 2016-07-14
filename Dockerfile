# Docker file to build Request Baskets service
# Version 1.0

# From latest alpine (currently "golang:1.6-onbuild")
FROM golang:onbuild

# Expose ports
EXPOSE 55555
