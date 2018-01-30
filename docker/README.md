# Docker Scripts

## Overview

```bash
$ docker images

REPOSITORY          TAG        IMAGE ID            CREATED             SIZE
request-baskets     golang     8fe4269dbc19        7 minutes ago       768.4 MB
request-baskets     ubuntu     86d1c3b87154        53 seconds ago      378.8 MB
request-baskets     alpine     7991c7cec214        6 minutes ago       15.2 MB
...
```

## Docker with "golang"

:warning: `ONBUILD` image variants are [deprecated](https://github.com/docker-library/official-images/issues/2076), and their use is discouraged.

This is an example of how to use a great [golang:onbuild](https://github.com/docker-library/docs/tree/master/golang) image from official Docker library for Go language.

Building an image that is based on `golang:onbuild` image should be triggered from the root folder of Go application. The source code from the current folder is copied into container and compiled inside. The result of compilation is packed into a new image with a command that launches the application. The content of [Dockerfile](./golang/Dockerfile) in this case is minimal.

Usage example (should be started from source code root directory):
```bash
$ docker build -t request-baskets .
$ docker run -it --rm --name rbaskets -p 55555:55555 request-baskets
```

Now you can visit [http://localhost:55555](http://localhost:55555) in your browser.

To stop the service simply hit `Ctrl+C` - this will stop and delete the running container (note `--rm` flag).

Use of `golang:onbuild` image allows you to build and run Go applications without Go SDK being installed on your computer. However the Go SDK with all dependencies remains in the final image of built application, which results in really big image size (see above).

## Docker with "ubuntu"

This is an example of how to build and package Go application within container based on well known Linux server images. In this case the image is based on latest [ubuntu](https://github.com/docker-library/docs/tree/master/ubuntu) image from official Docker library.

The [Dockerfile](./ubuntu/Dockerfile) performs several steps within a one-line command:

 * Update current container
 * Install `golang` & `git` packages
 * Build `request-baskets` service and move it in `bin` directory
 * Cleanup: delete built folder and purge `golang` & `git` packages

Since Docker persists every command as a separate layer, one-liner allows to keep the container size smaller and do not include `golang` & `git` packages with dependencies in any layer of final container.

However, the purge command does not remove all dependencies that are installed with `golang` & `git` packages and final image results in more than 350 MB, even though the original Ubuntu image is only 125 MB.

Since application is built using `go get github.com/darklynx/request-baskets` command, which pulls the latest version of application from GitHub, you do not need to launch image built from source directory. This can be done from any location:

```bash
$ cd docker/ubuntu
$ docker build -t request-baskets .
$ docker run -d --name rbaskets -p 55555:55555 request-baskets
```

Last command from above launches an instance of container in detached mode (as a service). This let you use container in the background and provide you following possibilities to control container and service.

Access log of running service:
```bash
$ docker logs rbaskets
```

Stop the container with service:
```bash
$ docker stop rbaskets
```

And start it again:
```bash
$ docker start rbaskets
```

Note that `request-baskets` is configured to save data in Bolt database, hence the collected data is not lost after container restart. Database is placed on a volume that can be accessed from another containers for analyses or backup purposes.

## Docker with "minimal" (alpine)

Building minimalistic docker image with `request-baskets` service is spit in 2 steps:

 * Compilation of Go project with a help of `golang:latest` container
 * Creation of `request-baskets` service image based on `alpine:latest` image

In order to simplify the process [build script](./minimal/build.sh) is provided. Simply run it from any location and it should build a minimalistic image with `request-baskets` service for you.

**Behind the scene**

During initial step [golang](https://github.com/docker-library/docs/tree/master/golang) Docker container is used to compile `request-baskets` from source code hosted on GitHub. The result of compilation is saved outside of the container within the [script's directory](./minimal). This is achieved by mapping the current directory as a volume for compilation target `/go/bin` (default for `golang` image) inside the container. Additionally compilation is configured to enable static linking (note `CGO_ENABLED=0` environment variable).

Second step uses [Dockerfile](./minimal/Dockerfile) to build image based on simply wonderful [alpine](https://github.com/docker-library/docs/tree/master/alpine) image from official Docker library. This image has less then 5 MB size and with `request-baskets` service included the final image takes just ~15 MB.

**Important:** since `alpine` image includes minimal to none dependencies to minimize the original image size the [static linking](http://www.blang.io/posts/2015-04_golang-alpine-build-golang-binaries-for-alpine-linux/) during Go compilation is a solution to build an executable that may run inside `alpine` container.

Similar to "ubuntu" Docker script this [Dockerfile](./minimal/Dockerfile) also declares a volume to expose the Bolt database location.
