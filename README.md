[![Build Status](https://travis-ci.com/jbrekelmans/kube-compose.svg?branch=master)](https://travis-ci.com/jbrekelmans/kube-compose)
[![License](https://img.shields.io/badge/license-Apache_v2.0-blue.svg)](https://github.com/jbrekelmans/kube-compose/blob/master/LICENSE.md)

# Introduction

kube-compose is a CI tool that can create and destroy environments in Kubernetes based on docker compose files.

## Contents

* [Installation](#Installation)
* [Getting Started](#Getting-Started)
  * [Prerequisites](#Prerequisites)
  * [Running Tests](#Running-Tests)
  * [Build And Package](#Build-And-Package)
* [Commands](#Commands)
* [Examples](#Examples)
* [Advanced Usage](#Advanced-Usage)

## Installation

Use the following to be able to install on MacOS via Homebrew:

Running the below command will add the Homebrew tap to our repository

```bash
brew tap kube-compose/homebrew-kube-compose
```

Now you've added our custom tap, you can download with the following command:

```bash
brew install kube-compose
```

To upgrade kube-compose to the latest stable release use the following command:

```bash
brew upgrade kube-compose
```

Otherwise download the binary from https://github.com/jbrekelmans/kube-compose/releases, and place it on your `PATH`.

## Getting Started

### Prerequisites

NA

### Testing

Use `kubectl` to set the target Kubernetes namespace and the service account of kube-compose.

Run `kube-compose` with the test [docker-compose.yml](test/docker-compose.yml):

```bash
(cd test && ../kube-compose --env-id test123 up)
```

This writes to the directory `test/output` the created Kubernetes resources.

To clean up after the test:

```bash
kubectl delete $(kubectl get all -lenv=test123 -oname)
```

### Build And Package

You can compile the kube-compose binary using either Go or Docker-compose.

Using Go:

```go
go build -o kube-compose .
```

Using Docker-compose

```bash
docker-compose build
```

## Commands

The following is a list of all available commands:
 
```bash
Available Commands:
  up          A brief description of your command
  down        A brief description of your command
  help        Help about any command
```

Intuitively, the `kube-compose up` mirrors functionality of `docker-compose up`, but runs containers on a Kubernetes cluster instead of on the host docker. Likewise `kube-compose down` behaves in a similar fashion.

## Environment Variables

kube-compose currently supports 2 environment variables. If these environment variables are set, you don't need to pass the `--namespace` and `--env-id` flags.

```bash
export KUBECOMPOSE_NAMESPACE=""
export KUBECOMPOSE_ENVID=""
```

## Examples

kube-compose loads pod and services definitions implicitly defined in a docker compose file, and creates them in a target namespace via the following command:

```bash
kube-compose -e mybuildid up
```

The target namespace and service account token are loaded from the context set in `~/.kube/config`. This means that k8s Client Tools kubectl commands can be used to configure kube-compose's target namespace and service account.

If no `~/.kube/config` exists and kube-compose is run inside a pod in Kubernetes, the pod's namespace becomes the target namespace, and the service account used to create pods and services is the pod's service account.

The namespace can be overridden via the `--namespace` option, for example: `kube-compose --namespace ci up`.

### Foreground mode to view the logs of running pods

```bash
kube-compose --namespace default --env-id test123 up

kube-compose --namespace default --env-id test123 down
```

```bash
kube-compose up -n default -e test123

kube-compose down -n default -e test123

```

If environment variables are already set.

```bash
kube-compose up

kube-compose down
```

Start individual services defined in docker-compose.yml

```bash
kube-compose up service-1

kube-compose up service-1 service-2
```

### Detach mode

```bash
kube-compose --namespace default --env-id test123 up --detach
```

```bash
kube-compose up -n default -e test123 -d
```

If environment variables are already set.

```bash
kube-compose up -d
```

## Why another tool

Although [kompose](https://github.com/kubernetes/kompose) can already convert docker compose files into Kubernetes resources. The main differences between kube-compose and Kompose are:

1. kube-compose generates Kubernetes resource names and selectors that are unique for each build to support shared namespaces and scaling to many concurrent CI environments.

1. kube-compose creates pods with `restartPolicy: Never` instead of deployments, so that failed pods can be inspected, no logs are lost due to pod restarts, and Kubernetes cluster resources are used more efficiently.

1. kube-compose allows startup dependencies to be specified by respecting [docker compose](https://docs.docker.com/compose/compose-file/compose-file-v2#depends_on)'s `depends_on` field.

1. kube-compose currently depends on the docker daemon to pull Docker images and extract their healthcheck.

## Advanced Usage

If you require that an application is not started until one of its dependencies is healthy, you can add `condition: service_healthy` to the `depends_on`, and give the dependency a [Docker healthchecks](https://docs.docker.com/engine/reference/builder#healthcheck).

Docker healthchecks are converted into [Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/).
