# Builds minimalistic Request Baskets service container
# Version 1.0

# From latest alpine (currently "alpine:3.4")
FROM alpine

MAINTAINER Vladimir L, vladimir_l@gmx.net

# Create a volume for request-baskets service data folder
VOLUME /var/lib/rbaskets

# Include CA certs for baskets that forward to https://...
RUN apk --no-cache add ca-certificates

# Copy built service
COPY request-baskets /usr/local/bin/

# Expose ports
EXPOSE 55555

CMD request-baskets -l 0.0.0.0 -db bolt -file /var/lib/rbaskets/baskets.db
