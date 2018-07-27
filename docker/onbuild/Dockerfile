# Docker file to build Request Baskets service
# Version 1.0

# From latest onbuild (currently "golang:1.6-onbuild")
FROM golang:onbuild

MAINTAINER Vladimir L, vladimir_l@gmx.net

# Expose ports
EXPOSE 55555

# We need to change default binding ip
CMD ["go-wrapper", "run", "-l", "0.0.0.0"]
