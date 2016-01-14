# Request Baskets

Request Baskets is a web service to collect arbitrary HTTP requests and inspect them via RESTful API or simple web UI.

It is strongly inspired by ideas and application design of the [RequestHub](https://github.com/kyledayton/requesthub) project and reproduces functionality offered by [RequestBin](http://requestb.in/) service.

Distinguishing features of Request Baskets service:

 * RESTful API to manage and configure baskets (see `doc/api-swagger.yaml`)
 * All baskets are protected by **unique** tokens from unauthorized access (end-points to collect requests do not require authorization though)
 * Individually configurable capacity for every basket
 * Pagination support to retrieve collections: basket names, collected requests
 * Alternative storage types for configured baskets and collected requests:
   * *In-memory* - ultra fast, but limited to available RAM and collected data is lost after service restart
   * *Bolt DB* - fast persistent storage for collected data based on embedded [Bolt](https://github.com/boltdb/bolt) database, service can be restarted without data loss and storage is not limited by available RAM
   * Can be extended by custom implementations of storage interface

## Screenshot

![Request Baskets](http://i.imgur.com/T2mcNN9.png)

## Install

### Build from source

```bash
$ go get github.com/darklynx/request-baskets
```

### Run

```bash
$ export PATH=$PATH:$GOPATH/bin
$ request-baskets
```

## Configuration

Request Baskets service supports several command line configuration parameters. Use `-h` or `--help` to print command line help:

```
$ request-baskets --help
Usage of bin/request-baskets:
  -db string
    	Baskets storage type: mem - in-memory, bolt - bolt DB (default "mem")
  -file string
    	Database location, only applicable for file databases (default "./baskets.db")
  -maxsize int
    	Maximum allowed basket size (max capacity) (default 2000)
  -p int
    	HTTP service port (default 55555)
  -page int
    	Default page size (default 20)
  -size int
    	Initial basket size (capacity) (default 200)
  -token string
    	Master token, random token is generated if not provided
```

### Parameters

 * `-p` *port* - HTTP service listener port, default value is `55555`
 * `-page` *size* - default page size to retrieve collections
 * `-size` *size* - default basket capacity of new baskets if not specified
 * `-maxsize` *size* - maximum allowed basket capacity, basket capacity greater than this number will be rejected by service
 * `-token` *token* - master token to gain control over all baskets, if not specified a random token will be generated when service is launched and printed to *stdout*
 * `-db` *type* - defines baskets storage type: `mem` - in-memory storage, `bolt` - [Bolt DB](https://github.com/boltdb/bolt/) database
 * `-file` *location* - location of bolt DB database file, only relevant if appropriate storage type is chosen

## Usage

Open [http://localhost:55555](http://localhost:55555) in your browser. The main page will display a list of baskets that may be accessed if the basket *token* is known. It is possible to create a new basket if the name is not in use.

If basket was successfully created the authorization *token* is displayed. It is **important** to remember the *token* because it authorizes the access to management features of created basket and allows to retrieve collected HTTP requests. The token is temporary stored in browser session to simplify UI integration and improve user experience. However, once browser tab is closed, the token will be lost.

To collect HTTP requests send them (GET, POST, PUT, DELETE, etc.) to `http://localhost:55555/<basket_name>`

To view collected requests and manage basket:
 * Open basket web UI `http://localhost:55555/web/<basket_name>`
 * Use [RESTful API](https://github.com/darklynx/request-baskets/blob/master/doc/api-swagger.yaml) exposed at `http://localhost:55555/baskets/<basket_name>`

It is possible to forward all incoming HTTP requests to arbitrary URL by configuring basket via web UI or RESTful API.

### Persistent storage

By default Request Baskets service stores configured baskets and collected HTTP requests in memory. This data is lost after service or server restart. However the service can be configured to keep collected data on file system. This allows service restart without loosing created baskets and collected data.

To start service in persistent mode simply configure the appropriate storage type, such as [bolt DB](https://github.com/boltdb/bolt/):

```
$ request-baskets -db bolt -file /var/lib/request-baskets/baskets.db
2016/01/08 23:15:28 [info] generated master token: abcdefgh1234567...
2016/01/08 23:15:28 [info] using bolt DB to store baskets
2016/01/08 23:15:28 [info] bolt database location: /var/lib/rbaskets/baskets.db
2016/01/08 23:15:28 [info] starting HTTP server on port: 55555
...
```

Any other kind of storages or databases (e.g. MySQL, MongoDb) to keep collected data can be introduced by implementing following interfaces: `BasketsDatabase` and `Basket`
