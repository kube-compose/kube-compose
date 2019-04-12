[![Build Status](https://travis-ci.com/jbrekelmans/kube-compose.svg?branch=master)](https://travis-ci.com/jbrekelmans/kube-compose)

# Introduction
kube-compose is a CI tool that can create and destroy environments in Kubernetes based on docker compose files.

# Why another tool?
Although [kompose](https://github.com/kubernetes/kompose) can already convert docker compose files into Kubernetes resources. The main differences between kube-compose and Kompose are:
1. kube-compose generates Kubernetes resource names and selectors that are unique for each build to support shared namespaces and scaling to many concurrent CI environments.
1. kube-compose creates pods with `restartPolicy: Never` instead of deployments, so that failed pods can be inspected, no logs are lost due to pod restarts, and Kubernetes cluster resources are used more efficiently.
1. kube-compose allows startup dependencies to be specified by respecting [docker compose](https://docs.docker.com/compose/compose-file/compose-file-v2#depends_on)'s `depends_on` field.
1. kube-compose currently depends on the docker daemon to pull Docker images and extract their healthcheck.

# Installation
Download the binary from https://github.com/jbrekelmans/kube-compose/releases, and place it on your `PATH`.

# Usage
kube-compose loads pod and services definitions implicitly defined in a docker compose file, and creates them in a target namespace via the following command:
```
kube-compose -e mybuildid up
```

The target namespace and service account token are loaded from the context set in `~/.kube/config`. This means that Openshift Origin Client Tools' `oc login` and `oc project` commands can be used to configure kube-compose's target namespace and service account.

If no `~/.kube/config` exists and kube-compose is run inside a pod in Kubernetes, the pod's namespace becomes the target namespace, and the service account used to create pods and services is the pod's service account.

The namespace can be overriden via the `--namespace` option, for example: `kube-compose --namespace ci up`.¯

# Advanced usage
If you require that an application is not started until one of its dependencies is healthy, you can add `condition: service_healthy` to the `depends_on`, and give the dependency a [Docker healthchecks](https://docs.docker.com/engine/reference/builder#healthcheck).

Docker healthchecks are converted into [Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/).

# Building
```
go build -o kube-compose .
```
# Building (docker)
```
docker-compose build
```

# Testing
Use `kubectl` or `oc` to set the target Kubernetes namespace and the service account of kube-compose.

Run `kube-compose` with the test [docker-compose.yml](test/docker-compose.yml):
```
(cd test && ../kube-compose --env-id test123 up)
```
This writes to the directory `test/output` the created Kubernetes resources.

To clean up after the test:
```
kubectl delete $(kubectl get all -lenv=test123 -oname)
```
