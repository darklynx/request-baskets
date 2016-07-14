# Request Baskets service (latest)
# Version 1.0

FROM ubuntu

MAINTAINER Vladimir L, vladimir_l@gmx.net

# Create a volume for rbaskets service (uncomment if backups are required)
#VOLUME /opt/rbaskets

# One liner:
# - Install golang & git
# - Build the service inside temp directory
# - Cleanup
RUN set -x \
	&& apt-get update \
	&& apt-get -y upgrade \
	&& apt-get install -y golang git \
	&& export GOPATH="$(mktemp -d)" \
	&& go get github.com/darklynx/request-baskets \
	&& cp "$GOPATH/bin/request-baskets" /usr/local/bin/ \
	&& mkdir -p /opt/rbaskets \
	&& rm -rf "$GOPATH" \
	&& apt-get purge --auto-remove -y golang git \
	&& apt-get autoclean -y \
	&& apt-get autoremove -y

# Expose ports
EXPOSE 55555

CMD request-baskets -db bolt -file /opt/rbaskets/baskets.db
