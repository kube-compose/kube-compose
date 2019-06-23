[![Build Status](https://travis-ci.com/kube-compose/kube-compose.svg?branch=master)](https://travis-ci.com/kube-compose/kube-compose)
[![License](https://img.shields.io/badge/license-Apache_v2.0-blue.svg)](https://github.com/kube-compose/kube-compose/blob/master/LICENSE.md)
[![Coverage Status](https://coveralls.io/repos/github/kube-compose/kube-compose/badge.svg?branch=master&r=5)](https://coveralls.io/github/kube-compose/kube-compose?branch=master?r=5)

# kube-compose

kube-compose can create and destroy environments in Kubernetes based on docker compose files with an emphasis on CI use cases

# Contents

* [Installation](#Installation)
  * [Manual installation](#Manual-installation)
* [Getting Started](#Getting-Started)
* [Examples](#Examples)
  * [Waiting for and ordering startup](#Waiting-for-and-ordering-startup)
  * [Volumes](#Volumes)
    * [x-kube-compose configuration](#x-kube-compose-configuration)
    * [Limitations](#Limitations)
  * [Running containers as specific users](#Running-containers-as-specific-users)
  * [Dynamic test configuration](#Dynamic-test-configuration)
* [Known limitations](#Known-limitations)
* [Developer information](#Developer-information)

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
`kube-compose` targets a Kubernetes namespace, and will need a running Kubernetes cluster and [a kube config file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/). If you do not have a running Kubernetes cluster, consider running one locally using:
1. Docker Desktop
2. [Minikube](https://kubernetes.io/docs/setup/minikube/)

`kube-compose` loads Kubernetes configuration the same way `kubectl` does, and it is recommended you use `kubectl` to manage Kubernetes configuration.

To run `kube-compose` with [the test docker-compose.yml](test/docker-compose.yml):
```bash
kube-compose -f'test/docker-compose.yml' -e'myuniquelabel' up
```
The `-e` flag sets a unique identifier that is used to isolate [labels and selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) and ensure names are unique when deploying to shared namespaces. This is ideal for CI, because there may be many jobs and test environments running at the same time. The above command will also attach to any pods created, so ctrl+c can be used to interrupt the process and return control the the terminal.

Similar to `docker-compose`, an environment can be stopped and destroyed using the `down` command: 
```bash
kube-compose -f'test/docker-compose.yml' -e'myuniquelabel' down
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
When performing system testing in CI, waiting until the application and stubs are ready is common. `kube-compose` supports ordered startup and can wait for the environment to be ready through [depends_on](https://docs.docker.com/compose/compose-file/compose-file-v2/#depends_on) with `condition: service_healthy` and healthchecks. This approach is powerful, because it does not require writing complicated startup scripts. NOTE: version 3 docker compose files do not support `depends_on` conditions anymore (see https://docs.docker.com/compose/startup-order/).

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
...will create the environment and wait for it to have fully started. The service `helper` is used to to make sure that `web` is healthy as soon as `kube-compose` returns, so that we can immediately use the environment (e.g. to run system testing).

NOTE: in the background `kube-compose` converts [Docker healthchecks](https://docs.docker.com/engine/reference/builder/#healthcheck) to [readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) and will only start service `web` when the pod of `db` is ready, and will only start `helper` when the pod of `web` is ready. The pod of `helper` exits immediately, but this pattern is very powerful. 

## Volumes
`kube-compose` currently supports a basic simulation of `docker-compose`'s bind mounted volumes. This supports the use case of mounting configuration files into containers, which is a very common way of parameterising containers (in CI).

`kube-compose` implements this by:
1. Building a helper image with the relevant host files;
1. Running the helper image as an [initContainer](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) that initialises an [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/) volume; -and
1. Mounting the emptyDir volume into the application container as per the configuration of the bind mounted volume.

Simulation of bind mounted volumes can be seen in action by running the `up` command with the following docker compose YAML:
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
  cluster_image_storage:
    # Change to type: 'docker' when using Docker Desktop's cluster
    type: 'docker_registry'
    host: 'docker-registry.apps.openshift-cluster.example.com'
  volume_init_base_image: 'ubuntu:latest'
```
The pod created by the example prints the contents of the docker compose YAML.

### x-kube-compose configuration
Because `kube-compose` builds and runs helper images, both a base image and a image storage location need to be configured. The base image must have `bash` and `cp` installed (see `volume_init_base_image`). The image storage location can be a docker registry or a docker daemon. The latter can be used to test locally with [Docker Dekstop's cluster](https://docs.docker.com/docker-for-mac/kubernetes/).

Currently `kube-compose` can only push to docker registries that are configured like OpenShift's default docker registry. In particular, `kube-compose` makes the following assumptions when the image storage location is a docker registry:
1. Within the cluster the hostname of the docker registry is assumed to be `docker-registry.default.svc:5000`.
1. The kube configuration is assumed to have bearer token credentials, that are supplied as the password to the docker registry (the username will be `unused`), or the docker registry is unauthenticated.
1. References to pushed images have the form `<registry>/<project>/<imagestream>:latest`, [as required by OpenShift](https://blog.openshift.com/remotely-push-pull-container-images-openshift/).

### Limitations
1. Volumes that are not bind mounted volumes are ignored.
1. If a docker compose service makes changes in a mount of a bind mounted volume then those changes will not be reflected in the host file system, and vice versa.
1. If docker compose services `s1` and `s2` have mounts `m1` and `m2`, respectively, and `m1` and `m2` mount overlapping portions of the host file system, then changes in `m1` will not be reflected in `m2` (if `c1=c2` then this can be implemented easily with the current implementation by mounting one volume multiple times).

The third limitation implies that sharing volumes between two docker compose services is not supported, even though this could be implemented through persistent volumes.

## Running containers as specific users
Images and stubs run in CI often cannot be easily modified because they are provided by a third party, and the cluster's pod security policy can deny images from being run with the correct user. For this reason, `kube-compose` allows you to use the `--run-as-user` flag:
```bash
kube-compose up --run-as-user
```
This will set each pod's `runAsUser` (and `runAsGroup`) based on the [`user` property](https://docs.docker.com/compose/compose-file/#domainname-hostname-ipc-mac_address-privileged-read_only-shm_size-stdin_open-tty-user-working_dir) of the `docker-compose` service and the [`USER` configuration](https://docs.docker.com/engine/reference/builder/#user) of the docker image. This will require additional privileges, but is an easy way of making CI just work.

NOTE1: if a Dockerfile does not have a `USER` instruction, then the user is inherited from the base image. This makes it very easy to run images as root.

NOTE2: this may seem like a useless feature, since the deployer can have permissions to create pods running as any user. But the `user` property of a `docker-compose` service would not be respected in this case.

## Dynamic test configuration
When running tests against a dynamic environment, the test configuration will need to be generated. Suppose for example that a `docker-compose` service named `my-service` has been deployed to a Kubernetes namespace named `mynamespace`, and the environment id was set to `myenv`. Then the command...
```bash
kube-compose -e'myenv' get 'my-service' -o'{{.Hostname}}'
```
...will output...
```bash
my-service-myenv.mynamespace.svc.cluster.local
```

The `get` subcommand of `kube-compose` allows dynamic test configuration to be generated through simple Shell scripts.

# Known limitations
1. The `up` subcommand does not an build images of `docker-compose` services ([#188](https://github.com/kube-compose/kube-compose/issues/188)).
1. If no `-f` and `--file` flags are present then `kube-compose` never looks for a `docker-compose.yml` or `docker-compose.yaml` file in parents of the current working directory ([#151](https://github.com/kube-compose/kube-compose/issues/151)).
1. `kube-compose` never loads `docker-compose.override.yml` and `docker-compose.override.yaml` files and behaves as if those do not exists ([#124](https://github.com/kube-compose/kube-compose/issues/124)).
1. When extending a `docker-compose` service using `extends`, only ports and environment are copied from the extended `docker-compose` service ([#48](https://github.com/kube-compose/kube-compose/issues/48)).
1. See [volume limitations](#Limitations).

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
To run unit tests with code coverage:
```bash
go test -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Testing
Use `kubectl` to set the target Kubernetes namespace and the service account of kube-compose.

Run `kube-compose` with the test [docker-compose.yml](test/docker-compose.yml):
```bash
kube-compose -f test/docker-compose.yml --env-id test123 up
```

To clean up after the test:
```bash
kube-compose down
```
