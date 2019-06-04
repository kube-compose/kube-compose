[![Build Status](https://travis-ci.com/kube-compose/kube-compose.svg?branch=master)](https://travis-ci.com/kube-compose/kube-compose)
[![License](https://img.shields.io/badge/license-Apache_v2.0-blue.svg)](https://github.com/kube-compose/kube-compose/blob/master/LICENSE.md)
[![Coverage Status](https://coveralls.io/repos/github/kube-compose/kube-compose/badge.svg?branch=master&r=3)](https://coveralls.io/github/kube-compose/kube-compose?branch=master?r=3)

# kube-compose

kube-compose is a CI tool that can create and destroy environments in Kubernetes based on docker compose files.

# Contents

* [Installation](#Installation)
  * [Manual installation](#Manual-installation)
* [Getting Started](#Getting-Started)
* [Examples](#Examples)
  * [Waiting for and ordering startup](#Waiting-for-and-ordering-startup)
  * [Volumes](#Volumes)
    * [Limitations](#Limitations)
  * [Running containers as specific users](#Running-containers-as-specific-users)
* [Developer information](#Developer-information)
* [Why another tool?](#Why-another-tool?)

# Installation
## Using Homebrew
Add the tap:
```bash
brew tap kube-compose/homebrew-kube-compose
```
Install `kube-compose`:
```bash
brew install kube-compose
```
To upgrade `kube-compose` to the latest stable version:
```bash
brew upgrade kube-compose
```

## Manual installation
Download the binary from https://github.com/kube-compose/kube-compose/releases, ensure it has execute permissions and place it on your `PATH`.

# Getting Started
`kube-compose` targets a Kubernetes namespace, and will need a running Kubernetes cluster and [a kube config file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/). If you do not have a running Kubernetes cluster, consider running one locally using [Minikube](https://kubernetes.io/docs/setup/minikube/).

`kube-compose` loads Kubernetes configuration the same way `kubectl` does, and it is recommended you use `kubectl` to manage Kubernetes configuration.

To run `kube-compose` with [the test docker-compose.yml](test/docker-compose.yml):
```bash
kube-compose -f 'test/docker-compose.yml' -e 'myuniquelabel' up
```
The `-e` flag sets a unique identifier that is used to isolate [labels and selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) and ensure names are unique when deploying to shared namespaces. This is ideal for CI, because there may be many jobs and test environments running at the same time. The above command will also attach to any pods created, so ctrl+c can be used to interrupt the process and return control the the terminal.

Similar to `docker-compose`, an environment can be stopped and destroyed using the `down` command: 
```bash
kube-compose -f 'test/docker-compose.yml' -e 'myuniquelabel' down
```

The CLI of `kube-compose` mirrors `docker-compose` as much as possible, but has some differences. To avoid repeating the `-e` flag you can use the envirnoment variable `KUBECOMPOSE_ENVID`. The above example can also be written as:
```bash
cd test
export KUBECOMPOSE_ENVID='myuniquelabel'
kube-compose up
```
and
```bash
kube-compose down
```
For a full list of options and commands run:
```bash
kube-compose --help
```

# Examples
We will see several examples that support common CI use cases, in particular the following common system testing steps:
1. Start environment
2. Wait until the environment has fully started
3. Run tests
4. Stop environmnent

## Waiting for and ordering startup
When performing system testing in CI, often one has to wait until the application and stubs are ready. `kube-compose` supports ordered startup and can wait for the environment to be ready through [depends_on](https://docs.docker.com/compose/compose-file/compose-file-v2/#depends_on) with `condition: service_healthy` and healthchecks. This approach is powerful, because it does not require writing complicated startup scripts. NOTE: version 3 docker compose files do not support `depends_on` conditions anymore (see https://docs.docker.com/compose/startup-order/).

For example, if `docker-compose.yml` is...
```yaml
version: '2.4'
services:
  web:
    image: web:latest
    depends_on:
      db:
        condition: service_healthy
  db:
    image: db:latest
  helper:
    image: ubuntu:latest
    depends_on:
      web:
        condition: service_healthy
```
...then...
```bash
kube-compose up -d 'helper'
```
...will create the environment and wait for it to have fully started. The service `helper` is used to to make sure that `web` is healthy as soon as `kube-compose` returns, so that we can immediately use the environment (e.g. to running system testing).

NOTE: in the background `kube-compose` converts [Docker healthchecks](https://docs.docker.com/engine/reference/builder/#healthcheck) to [readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) and will only start service `web` when the pod of `db` is ready, and will only start `helper` when the pod of `web` is ready. The pod of `helper` exits immediately, but this pattern is very powerful. 


## Volumes
`kube-compose` currently supports a basic simulation of `docker-compose`'s bind mounted volumes. This supports the use case of mounting configuration files into containers, which is a very common way of parameterising containers (in CI).

`kube-compose` implements this by:
1. Building a helper image with the relevant host files;
2. Pushing the helper image to a registry;
3. Running the helper image as an [initContainer](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) that initialises an [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/) volume; -and
4. Mounting the emptyDir volume into the application container as per the configuration of the bind mounted volume.

NOTE: because `kube-compose` builds and pushes helper images, a base image and docker registry need to be configured. The base image must have `bash` and `cp` installed (see `volume_init_base_image`). Currently `kube-compose` can only push to docker registries that are configured like OpenShift's default docker registry. In particular, `kube-compose` makes the following assumptions:
1. Within the cluster the hostname of the cluster is assumed to be `docker-registry.default.svc:5000`.
2. The kube configuration is assumed to have bearer token credentials, that are supplied as the password to the docker registry (the username will be `unused`).
3. The reference of the image to be pushed has the form `<registry>/<project>/<imagestream>:latest`, [as required by OpenShift](https://blog.openshift.com/remotely-push-pull-container-images-openshift/).

Nevertheless, simulation of bind mount volumes can be demonstrated with the following `docker-compose.yml`:
```yaml
version: '2.4'
services:
  volumedemo:
    image: 'ubuntu:latest'
    entrypoint:
    - /bin/bash
    - -c
    - |
      echo 'Inception...'
      cat /mnt/inception
    volume:
    - './docker-compose.yml:/mnt/inception:ro'
x-kube-compose:
  push_images:
    docker_registry: 'docker-registry.apps.openshift-cluster.example.com'
  volume_init_base_image: 'ubuntu:latest'
```
When `kube-compose up` is run with the above `docker-compose.yml` a pod is created that prints the contents of `docker-compose.yml`.

### Limitations
1. Only bind mounted volumes are simulated.
2. If a docker compose service makes changes in a mount of the bind mounted volume then those changes will not be reflected in the host file system, and vice versa.
4. If docker compose services `s1` and `s2` have mounts `m1` and `m2`, respectively, and `m1` and `m2` mount overlapping portions of the host file system, then changes in `m1` will not be reflected in `m2` (if `c1=c2` then this can be implemented easily with the current implementation by mounting one volume multiple times).

The third limitation implies that sharing volumes between two docker compose services is not supported, even though this could be implemented through persistent volumes.


## Running containers as specific users
Often the images and stubs run in CI cannot be easily modified, and a pod security policy makes it that some images will not run as the correct user. `kube-compose` allows you to use the `--run-as-user`:
```bash
kube-compose up --run-as-user
```
This will set each pod's `runAsUser` (and `runAsGroup`) based on the [`user` property](https://docs.docker.com/compose/compose-file/#domainname-hostname-ipc-mac_address-privileged-read_only-shm_size-stdin_open-tty-user-working_dir) of the docker-compose service and the [`USER` configuration](https://docs.docker.com/engine/reference/builder/#user) of the docker image. This will require additional privileges, but is an easy way of making CI just work.

NOTE: if a Dockerfile does not have a `USER` instruction, then the user is inherited from the base image. This makes it very easy to run images as root.

# Developer information

## Building
```bash
go build .
```

## Linting
Install the linter if you do not have it already:
```bash
brew install golangci-lint
```
Run the linter:
```bash
golangci-lint run
```

## Unit testing
To run unit tests:
```bash
go test ./...
```

## Testing

Use `kubectl` to set the target Kubernetes namespace and the service account of kube-compose.

Run `kube-compose` with the test [docker-compose.yml](test/docker-compose.yml):

```bash
kube-compose -f test/docker-compose.yml --env-id test123 up
```

This writes the created Kubernetes resources to the directory test/output.

To clean up after the test:

```bash
kube-compose down
```


