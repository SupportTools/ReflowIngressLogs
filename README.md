Here’s a detailed `README.md` for your `ReflowIngressLogs` project:

---

# ReflowIngressLogs

ReflowIngressLogs is a lightweight Kubernetes utility for dynamically streaming and filtering logs from `ingress-nginx-controller` pods based on a specific namespace. It connects to your Kubernetes cluster, streams logs, and filters entries containing specific patterns, making it easy to monitor namespace-specific ingress activity.

---

## Features

- **Namespace-Based Filtering**: Filters ingress logs for a specific namespace by identifying log entries containing `[$NAMESPACE-`.
- **Dynamic Configuration**: Loads configuration from environment variables with support for defaults.
- **Kubernetes Integration**: Works seamlessly in-cluster or out-of-cluster using `KUBECONFIG`.
- **Robust Logging**: Provides structured logs with configurable log levels and formats (JSON or text).
- **Graceful Shutdown**: Cleans up log streaming on termination signals (`SIGINT`/`SIGTERM`).
- **Health Monitoring**: (Optional) Expose health endpoints for Kubernetes readiness checks.

---

## Installation

### Prerequisites

- Go `1.20+`
- A Kubernetes cluster with `ingress-nginx-controller` deployed.
- Access to Kubernetes API (via `KUBECONFIG` or in-cluster service account).

### Clone the Repository

```bash
git clone https://github.com/supporttools/ReflowIngressLogs.git
cd ReflowIngressLogs
```

---

## Configuration

ReflowIngressLogs is configured using environment variables. Below is the list of available options:

| Environment Variable        | Description                                                      | Default                                |
|-----------------------------|------------------------------------------------------------------|----------------------------------------|
| `DEBUG`                     | Enables debug logging. (`true` or `false`)                      | `false`                                |
| `KUBECONFIG`                | Path to the Kubernetes config file (used for out-of-cluster).   | `""` (in-cluster config is used)       |
| `NAMESPACE`                 | The namespace to filter logs for.                              | **Required**                          |
| `INGRESS_NAMESPACE`         | Namespace where `ingress-nginx-controller` is deployed.         | `ingress-nginx`                        |
| `LABEL_SELECTOR`            | Label selector to identify `ingress-nginx-controller` pods.    | `app.kubernetes.io/name=ingress-nginx` |

### Example Configuration

```bash
export DEBUG=true
export KUBECONFIG=/path/to/kubeconfig
export NAMESPACE=app
export INGRESS_NAMESPACE=ingress-nginx
export LABEL_SELECTOR=app.kubernetes.io/name=ingress-nginx
export LOG_LEVEL=debug
export LOG_FORMAT=json
```

---

## Usage

### Run Locally

```bash
go run cmd/main.go
```

### Build and Run via Docker

#### Build the Image

```bash
docker build -t your-dockerhub-user/reflow-ingress-logs:latest .
```

#### Run the Container

```bash
docker run -e NAMESPACE=app -e DEBUG=true -e INGRESS_NAMESPACE=ingress-nginx your-dockerhub-user/reflow-ingress-logs:latest
```

---

## Deployment in Kubernetes

Deploy the utility as a pod or deployment in your cluster.

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: reflow-ingress-logs
  namespace: logs
spec:
  replicas: 1
  selector:
    matchLabels:
      app: reflow-ingress-logs
  template:
    metadata:
      labels:
        app: reflow-ingress-logs
    spec:
      containers:
        - name: reflow-ingress-logs
          image: your-dockerhub-user/reflow-ingress-logs:latest
          env:
            - name: DEBUG
              value: "true"
            - name: NAMESPACE
              value: "app"
            - name: INGRESS_NAMESPACE
              value: "ingress-nginx"
            - name: LABEL_SELECTOR
              value: "app.kubernetes.io/name=ingress-nginx"
            - name: LOG_LEVEL
              value: "debug"
            - name: LOG_FORMAT
              value: "text"
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "200m"
```

---

## Logging

### Log Levels

Logs can be configured using the `LOG_LEVEL` environment variable:

- `debug`: For detailed logs during development.
- `info`: Default level for general information.
- `warn`: Warnings about non-critical issues.
- `error`: Errors that need immediate attention.
- `fatal`: Critical issues causing shutdown.

### Log Formats

Set `LOG_FORMAT` to `text` or `json` based on your logging needs.

---

## Development

### Build Locally

```bash
go build -o reflow-ingress-logs main.go
```

### Run Tests

```bash
go test ./...
```

---

## Contributing

We welcome contributions! To get started:

1. Fork the repository.
2. Create a feature branch.
3. Submit a pull request.

---

## Troubleshooting

### Common Errors

1. **No pods found**:
   Ensure the correct `INGRESS_NAMESPACE` and `LABEL_SELECTOR` are set to identify `ingress-nginx` pods.

2. **Kubernetes API authentication failed**:
   Ensure the `KUBECONFIG` is correctly set or the pod has appropriate RBAC permissions.

---

## License

This project is licensed under the [Apache License](LICENSE).

---