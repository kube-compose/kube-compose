# Usage
```
oc login https://my-openshift-cluster.example.com/
k8s-docker-compose up
```

# Building
```
go build -o k8s-docker-compose .
```
# Building (docker)
```
docker-compose build
```

# Testing
Use `kubectl` or `oc` to set the current Kubernetes cluster/namespace. `k8s-docker-compose` will target this context.

Run `k8s-docker-compose` with the test docker-compose.yml:
```
(cd test && ../k8s-docker-compose up)
```
This writes to the directory `test/output` the created Kubernetes resources.

To clean up after the test:
```
kubectl delete $(kubectl get all -lenv=test123 -oname)
```
