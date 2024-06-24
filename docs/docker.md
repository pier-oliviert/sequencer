# Docker

A Docker account is necessary if you want to use Docker as a container registry for Sequencer. For Docker to work, you'll need to

1. Create a repository over at [Docker](https://www.docker.com/)
2. Create a secret with an API token from your account.

## Create an API Token
![API Token creation dialog](../images/docker-access-token.png)
API Token can be created in your [profile page on Docker](https://hub.docker.com/settings/security). The token needs to have **Read and Write** permissions. Once you have the token, you need to create a secret within the same namespace as where Sequencer runs (by default `sequencer-system`).

```sh
kubectl create secret generic dockerhub-credentials \
  --from-literal=accessKey="$(YOUR_DOCKER_USERNAME)" \ 
  --from-literal=secretToken="$(YOUR_API_TOKEN)" \
  --namespace sequencer-system
```