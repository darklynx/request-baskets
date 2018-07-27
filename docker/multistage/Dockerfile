# Builds Request Baskets service using multi-stage builds
# Version 1.0

# Stage 1. Building service
FROM golang:latest as builder
WORKDIR /go/src/rbaskets
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/rbaskets

# Stage 2. Packaging into alpine
FROM alpine:latest
MAINTAINER Vladimir L, vladimir_l@gmx.net
RUN apk --no-cache add ca-certificates
VOLUME /var/lib/rbaskets
COPY --from=builder /go/bin/rbaskets /bin/rbaskets
EXPOSE 55555
CMD /bin/rbaskets -l 0.0.0.0 -db bolt -file /var/lib/rbaskets/baskets.db
