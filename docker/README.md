# Docker Scripts

## Overview

```bash
$ docker images

REPOSITORY           TAG          IMAGE ID        CREATED           SIZE
request-baskets      multistage   969d4a6f4799    8 minutes ago     11.9MB
request-baskets      minimal      91ee7e035488    15 seconds ago    15.9MB
request-baskets      ubuntu       cf2c7f5a5c75    3 minutes ago     155MB
request-baskets      onbuild      31bc1924e99e    5 minutes ago     719MB
...
```

## Use "multistage" docker build

Using a multi-stage docker builds is now considered to be the most efficient and [officially recommended](https://docs.docker.com/develop/develop-images/multistage-build/) way of building a minimalistic version of docker images for simple services or any other kinds of software.

In depth it is similar to the approach taken when building "minimal" version of docker container (see below), but instead of using an additional shell script to combine several steps (stages) it relies on the relatively new feature of Docker to describe all steps in a single [Dockerfile](./multistage/Dockerfile) and pass the result from one step to another using new syntax: `COPY --from=...`

This [Dockerfile](./multistage/Dockerfile) is designed to run from the root folder of service application. During the first stage (named `builder`) `golang:latest` docker container is used to compile the service with some tunings to minimize the size of resulting executable file. The second stage prepares "alpine" container and copies the service file from first stage into the new container, it also configures the default command to run when container is started.

To build the service image and launch an instance of container run following commands from the project root folder:

```bash
$ docker build -f docker/multistage/Dockerfile -t request-baskets .
$ docker run --rm --name rbaskets -d -p 55555:55555 request-baskets
2f5c3c236795c66324...

$ docker logs rbaskets
2018/07/27 00:01:45 [info] generated master token: bShKSx57jbbEIUa...
2018/07/27 00:01:45 [info] using Bolt database to store baskets
2018/07/27 00:01:45 [info] Bolt database location: /var/lib/rbaskets/baskets.db
2018/07/27 00:01:45 [info] HTTP server is listening on 0.0.0.0:55555
```

Now you are ready to use the Request Baskets service locally, just open [http://localhost:55555](http://localhost:55555) in your browser.


## Use "ubuntu" docker build

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


## Use "minimal" docker build

Building minimalistic docker image with `request-baskets` service is spit in 2 steps:

 * Compilation of Go project with a help of `golang:latest` container
 * Creation of `request-baskets` service image based on `alpine:latest` image

In order to simplify the process [build script](./minimal/build.sh) is provided. Simply run it from any location and it should build a minimalistic image with `request-baskets` service for you.

**Behind the scene**

During initial step [golang](https://github.com/docker-library/docs/tree/master/golang) Docker container is used to compile `request-baskets` from source code hosted on GitHub. The result of compilation is saved outside of the container within the [script's directory](./minimal). This is achieved by mapping the current directory as a volume for compilation target `/go/bin` (default for `golang` image) inside the container. Additionally compilation is configured to enable static linking (note `CGO_ENABLED=0` environment variable).

Second step uses [Dockerfile](./minimal/Dockerfile) to build image based on simply wonderful [alpine](https://github.com/docker-library/docs/tree/master/alpine) image from official Docker library. This image has less then 5 MB size and with `request-baskets` service included the final image takes just ~15 MB.

**Important:** since `alpine` image includes minimal to none dependencies to minimize the original image size the [static linking](http://www.blang.io/posts/2015-04_golang-alpine-build-golang-binaries-for-alpine-linux/) during Go compilation is a solution to build an executable that may run inside `alpine` container.

Similar to "ubuntu" Docker script this [Dockerfile](./minimal/Dockerfile) also declares a volume to expose the Bolt database location.


## Use "onbuild" docker build

:warning: `ONBUILD` image variants are [deprecated](https://github.com/docker-library/official-images/issues/2076), and their use is discouraged.

This is an example of how to use a former great [golang:onbuild](https://github.com/docker-library/docs/tree/master/golang) image from official Docker library for Go language.

Building an image that is based on `golang:onbuild` image should be triggered from the root folder of Go application. The source code from the current folder is copied into container and compiled inside. The result of compilation is packed into a new image with a command that launches the application. The content of [Dockerfile](./onbuild/Dockerfile) in this case is minimal.

Usage example (should be started from source code root directory):
```bash
$ docker build -f docker/onbuild/Dockerfile -t request-baskets .
$ docker run -it --rm --name rbaskets -p 55555:55555 request-baskets
```

Now you can visit [http://localhost:55555](http://localhost:55555) in your browser.

To stop the service simply hit `Ctrl+C` - this will stop and delete the running container (note `--rm` flag).

Use of `golang:onbuild` image allows you to build and run Go applications without Go SDK being installed on your computer. However the Go SDK with all dependencies remains in the final image of built application, which results in really big image size (see above).
