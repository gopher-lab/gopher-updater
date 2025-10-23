gopher-updater

`gopher-updater` is a companion tool for GitOps-driven Cosmos chains.

Synchronizing k8s pod updates with Cosmos governance can be a problem, especially when using Flux as an IaC solution. The main issue is that Flux treats GitHub as the source of truth, but updating a Cosmos node needs to happen in sync with Cosmos Governance.

`gopher-updater` takes care of bridging these two worlds. CI pushes its images to a registry, using a well-known tag that Flux doesn't know about (e.g. `release-v1.2.3`). `gopher-updater` then monitors the Governance module and the chain state. When the chain reaches the block height configured in the approved update proposal, it retags the manifest to a tag that Flux does know about. For example, if the governance proposal states that the updated version shouild be `v1.2.3`, `gopher-updater` will retag e.g. `release-v1.2.3` to `testnet-v1.2.3` or `mainnet-v1.2.3` depending on configuration. Flux will then take over and update the pod.

`gopher-updater` connects to the DockerHub registry via the REST API instead of via the Docker daemon. This is to prevent the complexity of configuring Docker-in-Docker and of adding a Docker daemon to the container.

## Configuration

All configuration is done by means of environment variables:

### Connectivity

`API_URL` - URL to connect to the Cosmos chain REST API. Default is `http://localhost:1317`.

### Docker parameters

`DOCKERHUB_USER` - User ID to connect to DockerHub. This is mandatory.

`DOCKERHUB_PASSWORD` - User ID to connect to DockerHub. This is mandatory.

`REPO_PATH` - Path to the repo within the DockerHub registry (e.g. `gopher-lab/gopher`). This is mandatory.

`SOURCE_PREFIX` - Prefix to the source tag (the tag that CI publishes to). The version number in the governance proposal will be appended to this. Default is `release-`.

 `TARGET_PREFIX` - Prefix to the tartet tag (the tag that will be created and that Flux knows about). This is mandatory.

### Other parameters

`POLL_INTERVAL` - How long to wait between Cosmos chain polls, in Golang Duration format. The default is `1m`.

`DRY_RUN` - If set to `true`, the application will not perform any retagging operations on DockerHub. Instead, it will log the actions it would have taken. This is useful for testing and validation. Default is `false`.

`HTTP_PORT` - The port on which to expose health, metrics, and profiling endpoints. Default is `8080`.

## Observability

The service exposes several endpoints for monitoring and debugging:
*   `GET /healthz`: A liveness probe that returns `200 OK` if the service is running.
*   `GET /readyz`: A readiness probe that returns `200 OK` if the service can connect to both the Cosmos chain and DockerHub. Otherwise, it returns `503 Service Unavailable`.
*   `GET /metrics`: Exposes Prometheus metrics for monitoring.
*   `GET /debug/pprof/`: Exposes Go's standard profiling endpoints.

## Usage

### Docker

```bash
docker run \
  -e REPO_PATH="my/repo" \
  -e TARGET_PREFIX="mainnet-" \
  -e DOCKERHUB_USER="myuser" \
  -e DOCKERHUB_PASSWORD="mypassword" \
  -e API_URL="http://my-cosmos-node:1317" \
  -p 8080:8080 \
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
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: REPO_PATH
          value: "my/repo"
        - name: TARGET_PREFIX
          value: "mainnet-"
        - name: DOCKERHUB_USER
          valueFrom:
            secretKeyRef:
              name: dockerhub
              key: username
        - name: DOCKERHUB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: dockerhub
              key: password
        - name: API_URL
          value: "http://my-cosmos-node:1317"
        - name: HTTP_PORT
          value: "8080"
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
        readinessProbe:
          httpGet:
            path: /readyz
            port: http
```
## Development

```bash
make lint
make test
make run
```
