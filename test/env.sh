docker rm -f registry; docker run -d --name registry -p 5000:5000 registry
docker rmi -f $(docker images -aq)
export KUBECOMPOSE_ENVID='env1'
export BLA='latest'
