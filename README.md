[![Build Status](https://travis-ci.com/jbrekelmans/jompose.svg?branch=master)](https://travis-ci.com/jbrekelmans/jompose)

# Introduction
Jompose is a CI tool that can create and destroy environments in Kubernetes based on docker compose files.

# Why another tool?
Although [kompose](https://github.com/kubernetes/kompose) can already convert docker compose files into Kubernetes resources, it does not address some valid CI use cases, and the author wanted to learn Go. The main use cases that Jompose addresses but kompose does not are:
1. Kubernetes resource names and selectors are unique for each build to support shared namespaces and scaling to as many concurrent CI environments as you desire.
1. Creates pods with `restartPolicy: Never` instead of deployments, so that failed pods can be inspected, no logs are lost due to pod restarts, and Kubernetes cluster resources are used more efficiently.
1. Jompose addresses startup dependencies by respecting [docker compose](https://docs.docker.com/compose/compose-file/compose-file-v2#depends_on)'s `depends_on` field.

# Usage
Jompose loads pod and services definitions implicitly defined in a docker compose file, and creates them in a target namespace via the following command:
```
jompose -e mybuildid up
```

The target namespace and service account token are loaded from the context set in `~/.kube/config`. This means that Openshift Origin Client Tools' `oc login` and `oc project` commands can be used to configure Jompose's target namespace and service account.

If no `~/.kube/config` exists and Jompose is run inside a pod in Kubernetes, the pod's namespace becomes the target namespace, and the service account used to create pods and services is the pod's service account.

The namespace can be overriden via the `--namespace` option, for example: `jompose --namespace ci up`.¯

# Advanced usage
If you require that an application is not started until one of its dependencies is healthy, you can add `condition: service_healthy` to the `depends_on`, and give the dependency a [Docker healthchecks](https://docs.docker.com/engine/reference/builder#healthcheck).

Docker healthchecks are converted into [Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/).

# Building
```
go build -o jompose .
```
# Building (docker)
```
docker-compose build
```

# Testing
Use `kubectl` or `oc` to set the target Kubernetes namespace and the service account of Jompose.

Run `jompose` with the test [docker-compose.yml](test/docker-compose.yml):
```
(cd test && ../jompose --env-id test123 up)
```
This writes to the directory `test/output` the created Kubernetes resources.

To clean up after the test:
```
kubectl delete $(kubectl get all -lenv=test123 -oname)
```
