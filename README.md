# Request Baskets

Request baskets is an HTTP request collector service. It is strongly inspired by ideas from [RequestHub](https://github.com/kyledayton/requesthub) project.

Distinguishing features of Request Baskets service:

 * RESTful API to manage and configure baskets (see `doc/api-swagger.yaml`)
 * All baskets are protected by **unique** tokens from unauthorized access (end-points to collect requests do not require authorization though)
 * Individually configurable capacity for every basket
 * Pagination support to retrieve collections: basket names, collected requests

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

Request baskets service supports several command line configuration parameters. Use `-h` or `--help` to print command line help:
```
$ request-baskets --help
Usage of request-baskets:
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

## Usage

Open [http://localhost:55555](http://localhost:55555) in your browser. The main page will display a list of baskets that may be accessed if the basket *token* is known. It is possible to create a new basket if the name is not in use.

If basket was successfully created the authorization *token* will be displayed. It is **important** to remember the *token* - it authorizes the access to manage created basket and view collected HTTP requests. The token is temporary stored in browser session to simplify UI integration and improve user experience. However, once browser tab is closed, the token will be lost.

To collect HTTP requests send them (GET, POST, PUT, DELETE, etc.) to `http://localhost:55555/<basket_name>`

To view collected requests and manage basket:
 * Open basket web UI `http://localhost:55555/web/<basket_name>`
 * Use [RESTful API](https://github.com/darklynx/request-baskets/blob/master/doc/api-swagger.yaml) exposed at `http://localhost:55555/baskets/<basket_name>`

It is possible to forward all incoming HTTP requests to arbitrary URL by configuring basket via web UI or RESTful API.
