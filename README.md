# Usage
```
oc login https://my-openshift-cluster.example.com/
jompose up
```

# Building
```
go build -o jompose .
```
# Building (docker)
```
docker-compose build
```

# Testing
Use `kubectl` or `oc` to set the current Kubernetes cluster/namespace. `jompose` will target this context.

Run `jompose` with the test docker-compose.yml:
```
(cd test && ../jompose --env-id test123 up)
```
This writes to the directory `test/output` the created Kubernetes resources.

To clean up after the test:
```
kubectl delete $(kubectl get all -lenv=test123 -oname)
```
