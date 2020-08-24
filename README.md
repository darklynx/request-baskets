# Request Baskets [![Build Status](https://travis-ci.org/darklynx/request-baskets.svg?branch=master)](https://travis-ci.org/darklynx/request-baskets) [![Coverage Status](https://coveralls.io/repos/github/darklynx/request-baskets/badge.svg?branch=master)](https://coveralls.io/github/darklynx/request-baskets?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/darklynx/request-baskets)](https://goreportcard.com/report/github.com/darklynx/request-baskets)

[Request Baskets](https://rbaskets.in) is a web service to collect arbitrary HTTP requests and inspect them via RESTful API or simple web UI.

It is strongly inspired by ideas and application design of the [RequestHub](https://github.com/kyledayton/requesthub) project and reproduces functionality offered by [RequestBin](http://requestb.in/) service.

## Table of Contents

- [Introduction](#introduction)
  - [Features](#features)
  - [Screenshots](#screenshots)
- [Install](#install)
  - [Build from source](#build-from-source)
  - [Run docker container](#run-docker-container)
- [Configuration](#configuration)
  - [Parameters](#parameters)
- [Usage](#usage)
  - [Bolt database](#bolt-database)
  - [PostgreSQL database](#postgresql-database)
  - [MySQL database](#mysql-database)
- [Docker](#docker)
  - [Build docker image](#build-docker-image)
  - [Run container as a service](#run-container-as-a-service)
  - [Cleanup](#cleanup)

## Introduction

[Request Baskets](https://rbaskets.in) service is available on our demonstration server: [https://rbaskets.in](https://rbaskets.in)

However, we encourage you to set up your own server and keep control over the information passed through and collected by Request Baskets service.

### Features

Distinguishing features of Request Baskets service:

 * [RESTful API](./doc/api-swagger.yaml) to manage and configure baskets, see [Request Baskets API](https://rbaskets.in/api.html) documentation in interactive mode
 * All baskets are protected by **unique** tokens from unauthorized access; end-points to collect requests do not require authorization though
 * Individually configurable capacity for every basket
 * Pagination support to retrieve collections: basket names, collected requests
 * Configurable responses for every HTTP method
 * Alternative storage types for configured baskets and collected requests:
   * *In-memory* - ultra fast, but limited to available RAM and collected data is lost after service restart
   * *Bolt DB* - fast persistent storage for collected data based on embedded [Bolt](https://github.com/boltdb/bolt) database, service can be restarted without data loss and storage is not limited by available RAM
   * *SQL database* - classical data storage, multiple instances of service can run simultaneously and collect data in shared data storage, which makes the solution more robust and scaleable ([PostgreSQL](https://www.postgresql.org) and [MySQL](https://www.mysql.com) are only supported at the moment)
   * Can be extended by custom implementations of storage interface

### Screenshots

Basket requests overview:
![Request Baskets - requests](http://i.imgur.com/NWNtYtY.png)

Configuration of basket responses:
![Request Baskets - responses](http://i.imgur.com/ooUdBib.png)

## Install

### Build from source

Build latest:

```bash
$ go get github.com/darklynx/request-baskets
```

Run:

```bash
$ export PATH=$PATH:$GOPATH/bin
$ request-baskets
```

### Run docker container

```
$ docker pull darklynx/request-baskets
$ docker run -p 55555:55555 darklynx/request-baskets
```

## Configuration

Request Baskets service supports several command line configuration parameters. Use `-h` or `--help` to print command line help:

```
$ request-baskets --help
Usage of bin/request-baskets:
  -db string
      Baskets storage type: mem - in-memory, bolt - Bolt DB, sql - SQL database (default "mem")
  -file string
      Database location, only applicable for file or SQL databases (default "./baskets.db")
  -conn string
      Database connection string for SQL databases, if undefined "file" argument is considered
  -l string
      HTTP listen address (default "127.0.0.1")
  -p int
      HTTP service port (default 55555)
  -page int
      Default page size (default 20)
  -size int
      Initial basket size (capacity) (default 200)
  -maxsize int
      Maximum allowed basket size (max capacity) (default 2000)
  -token string
      Master token, random token is generated if not provided
  -basket value
    	Name of a basket to auto-create during service startup (can be specified multiple times)
```

### Parameters

 * `-p` *port* - HTTP service listener port, default value is `55555`
 * `-page` *size* - default page size when retrieving collections
 * `-size` *size* - default new basket capacity, applied if basket capacity is not provided during creation
 * `-maxsize` *size* - maximum allowed basket capacity, basket capacity greater than this number will be rejected by service
 * `-token` *token* - master token to gain control over all baskets, if not defined a random token will be generated when service is launched and printed to *stdout*
 * `-db` *type* - defines baskets storage type: `mem` - in-memory storage, `bolt` - [Bolt](https://github.com/boltdb/bolt) database
 * `-file` *location* - location of Bolt database file, only relevant if appropriate storage type is chosen

## Usage

Open [http://localhost:55555](http://localhost:55555) in your browser. The main page will display a list of baskets that may be accessed if the basket *token* is known. It is possible to create a new basket if the name is not in use.

If basket was successfully created the authorization *token* is displayed. It is **important** to remember the *token* because it authorizes the access to management features of created basket and allows to retrieve collected HTTP requests. The token is temporary stored in browser session to simplify UI integration and improve user experience. However, once browser tab is closed, the token will be lost.

To collect HTTP requests send them (GET, POST, PUT, DELETE, etc.) to `http://localhost:55555/<basket_name>`

To view collected requests and manage basket:
 * Open basket web UI `http://localhost:55555/web/<basket_name>`
 * Use [RESTful API](https://github.com/darklynx/request-baskets/blob/master/doc/api-swagger.yaml) exposed at `http://localhost:55555/baskets/<basket_name>`

It is possible to forward all incoming HTTP requests to arbitrary URL by configuring basket via web UI or RESTful API.

### Bolt database

By default Request Baskets service keeps configured baskets and collected HTTP requests in memory. This data is lost after service or server restart. However a service can be configured to store collected data on file system. In this case the service can be restarted without loosing created baskets and collected data.

To start service in persistent mode simply configure the appropriate storage type, such as [Bolt database](https://github.com/boltdb/bolt):

```bash
$ request-baskets -db bolt -file /var/lib/request-baskets/baskets.db
2016/01/08 23:15:28 [info] generated master token: abcdefgh1234567...
2016/01/08 23:15:28 [info] using Bolt database to store baskets
2016/01/08 23:15:28 [info] Bolt database location: /var/lib/rbaskets/baskets.db
2016/01/08 23:15:28 [info] starting HTTP server on port: 55555
...
```

Any other kind of storages or databases (e.g. MySQL, MongoDb) to keep collected data can be introduced by implementing following interfaces: `BasketsDatabase` and `Basket`

### PostgreSQL database

The first attempt to implement SQL database storage for Request Baskets service is now available for evaluation. Even though the logic to organize the data within SQL database is written in the generic SQL dialect, the code make use of parametrized SQL queries that unfortunately do not have standard to express [parameter placeholders](http://go-database-sql.org/prepared.html#parameter-placeholder-syntax) across different databases.

Current implementation is based on PostgreSQL syntax. So running Request Baskets service with [PostgreSQL database](https://www.postgresql.org) as a storage is fully supported.

Use following example to start the Request Baskets service with PostgreSQL database:

```bash
$ request-baskets -db sql -conn "postgres://rbaskets:pwd@localhost/baskets?sslmode=disable"
2018/01/25 01:06:25 [info] generated master token: mSEAcYvpDlg...
2018/01/25 01:06:25 [info] using SQL database to store baskets
2018/01/25 01:06:25 [info] SQL database type: postgres
2018/01/25 01:06:25 [info] creating database schema
2018/01/25 01:06:25 [info] database is created, version: 1
2018/01/25 01:06:25 [info] HTTP server is listening on 127.0.0.1:55555
...
```

The documentation of [Go driver for PostgreSQL](https://godoc.org/github.com/lib/pq) provides detailed description of connection string and its parameters.

If no configured instance of PostgreSQL server is available to test the Request Baskets service with, there is a quick way to launch one using Docker with following command:

```bash
$ docker run --rm --name pg_baskets -e POSTGRES_USER=rbaskets -e POSTGRES_PASSWORD=pwd \
    -e POSTGRES_DB=baskets -d -p 5432:5432 postgres

# following command will stop and destroy the instance of PostgreSQL container
$ docker stop pg_baskets
```

### MySQL database

Added driver and application support within the SQL basket database for [MySQL](https://www.mysql.com) (or [MariaDB](https://mariadb.org)) database.

Use following example to start the Request Baskets service with MySQL database:

```bash
$ request-baskets -db sql -conn "mysql://rbaskets:pwd@/baskets"
2018/01/28 23:39:59 [info] generated master token: aPgyuLxw723q...
2018/01/28 23:39:59 [info] using SQL database to store baskets
2018/01/28 23:39:59 [info] SQL database type: mysql
2018/01/28 23:39:59 [info] creating database schema
2018/01/28 23:39:59 [info] database is created, version: 1
2018/01/28 23:39:59 [info] HTTP server is listening on 127.0.0.1:55555
...
```

The documentation of [Go driver for MySQL](https://github.com/go-sql-driver/mysql#usage) provides detailed description of connection string and its parameters.

If no configured instance of MySQL server is available to test the Request Baskets service with, there is a quick way to launch one using Docker with following command:

```bash
$ docker run --rm --name mysql_baskets -e MYSQL_USER=rbaskets -e MYSQL_PASSWORD=pwd \
    -e MYSQL_DATABASE=baskets -e MYSQL_RANDOM_ROOT_PASSWORD=yes -d -p 3306:3306 mysql

# following command will stop and destroy the instance of MySQL container
$ docker stop mysql_baskets
```

## Docker

### Build docker image

```bash
$ docker build -t request-baskets .
```

This will create a docker image using [multi-stage docker builds](https://docs.docker.com/develop/develop-images/multistage-build/) approach with 2 stages: compiling the service and packaging the result to a tiny alpine container. The resulting size of built image is ~12 Mb.

Note: since the first stage is using `golang:latest` container to build the service executable, there is no need to have Go lang SDK installed on the machine where process of building the container is taking place.

See [docker folder](./docker) for alternative docker builds with detailed explanation of the process for every variant.

### Run container as a service

```bash
$ docker run --name rbaskets -d -p 55555:55555 request-baskets
$ docker logs rbaskets
```

### Cleanup

Stop and delete docker container:
```bash
$ docker stop rbaskets
$ docker rm rbaskets
```

Delete docker image:
```bash
$ docker rmi request-baskets
```
