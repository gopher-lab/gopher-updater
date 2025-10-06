# gopher-updater

`gopher-updater` is a companion tool for GitOps-driven Cosmos chains.

Synchronizing k8s pod updates with Cosmos governance can be a problem, especially when using Flux as an IaC solution. The main issue is that Flux treats GitHub as the source of truth, but updating a Cosmos node needs to happen in sync with Cosmos Governance.

`gopher-updater` takes care of bridging these two worlds. CI pushes its images to a registry, using a well-known tag that Flux doesn't know about (e.g. `release-v1.2.3`). `gopher-updater` then monitors the Governance module and the chain state. When the chain reaches the block height configured in the approved update proposal, it retags the manifest to a tag that Flux does know about. For example, if the governance proposal states that the updated version shouild be `v1.2.3`, `gopher-updater` will retag e.g. `release-v1.2.3` to `testnet-v1.2.3` or `mainnet-v1.2.3` depending on configuration. Flux will then take over and update the pod.

`gopher-updater` connects to the DockerHub registry via the REST API instead of via the Docker daemon. This is to prevent the complexity of configuring Docker-in-Docker and of adding a Docker daemon to the container.

## Configuration

All configuration is done by means of environment variables:

### Connectivity

`RPC_URL` - URL to connect to the Cosmos chain REST API. Default is `http://localhost:1317`.

### Docker parameters

`DOCKERHUB_USER` - User ID to connect to DockerHub

`DOCKERHUB_PASSWORD` - User ID to connect to DockerHub

`REPO_PATH` - Path to the repo within the DockerHub registry (e.g. `gopher-lab/gopher`). There is no default, it is mandatory.

`SOURCE_PREFIX` - Prefix to the source tag (the tag that CI publishes to). The version number in the governance proposal will be appended to this. Default is `release-`.

 `TARGET_PREFIX` - Prefix to the tartet tag (the tag that will be created and that Flux knows about). There is no default, it is mandatory.

### Other parameters

`POLL_INTERVAL` - How long to wait between Cosmos chain polls, in Golang Duration format. The default is `1m`.

## Usage

### Docker

```bash
docker run \
  -e REPO_PATH="my/repo" \
  -e TARGET_PREFIX="mainnet-" \
  -e DOCKERHUB_USERNAME="myuser" \
  -e DOCKERHUB_PASSWORD="mypassword" \
  -e GRPC_ADDRESS="grpc.example.com:9090" \
  gopher-updater:latest
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gopher-updater
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gopher-updater
  template:
    metadata:
      labels:
        app: gopher-updater
    spec:
      containers:
      - name: gopher-updater
        image: gopher-updater:latest
        env:
        - name: REPO_PATH
          value: "my/repo"
        - name: TARGET_PREFIX
          value: "mainnet-"
        - name: DOCKERHUB_USERNAME
          valueFrom:
            secretKeyRef:
              name: dockerhub
              key: username
        - name: DOCKERHUB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: dockerhub
              key: password
        - name: GRPC_ADDRESS
          value: "grpc.example.com:9090"
```

## Development

```bash
make lint
make build
make test
make run
```
