# Building the application
```
docker-compose build
```
# Running a test for up
```
docker-compose run k8s-docker-compose up
```
This writes to the directory `test/output` the Kubernetes representation of `test/docker-compose.yml`.

To clean up after the test:
```
kubectl delete pod/db pod/permissions9cxservice pod/authentication9cxservice service/db service/permissions9cxservice service/authentication9cxservice 
```