# Request Baskets service (latest)
# Version 1.1

FROM ubuntu

MAINTAINER Vladimir L, vladimir_l@gmx.net

# Create a volume for request-baskets service data folder
VOLUME /var/lib/rbaskets

# One liner:
# - Install golang & git
# - Build the service inside temp directory
# - Cleanup
RUN set -x \
	&& apt-get update \
	&& apt-get -y upgrade \
	&& apt-get install -y golang-go git \
	&& export GOPATH="$(mktemp -d)" \
	&& go get github.com/darklynx/request-baskets \
	&& cp "$GOPATH/bin/request-baskets" /usr/local/bin/ \
	&& rm -rf "$GOPATH" \
	&& apt-get purge --auto-remove -y git golang-go

# Expose ports
EXPOSE 55555

CMD request-baskets -l 0.0.0.0 -db bolt -file /var/lib/rbaskets/baskets.db
