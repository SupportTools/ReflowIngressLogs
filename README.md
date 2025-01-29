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

### Install via Helm

```bash
helm repo add supporttools https://charts.support.tools/
helm repo update
helm install reflow-ingress-logs supporttools/reflow-ingress-logs -f values.yaml
```

To override default values:

```bash
helm install reflow-ingress-logs supporttools/reflow-ingress-logs --values custom-values.yaml
```

To upgrade:

```bash
helm upgrade reflow-ingress-logs supporttools/reflow-ingress-logs -f values.yaml
```

To uninstall:

```bash
helm uninstall reflow-ingress-logs
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
| `DEFAULT_LOG_FORMAT`        | Use the default ingress-nginx log format. (`true` or `false`)  | `true`                                 |

### Custom Log Format

By default, ingress-nginx logs are formatted using the following pattern:

```yaml
log-format-upstream: '$remote_addr - $remote_user [$time_local] $request $status $body_bytes_sent $http_referer $http_user_agent $request_length $request_time [$proxy_upstream_name] [$proxy_alternative_upstream_name] $upstream_addr $upstream_response_length $upstream_response_time $upstream_status $req_id'
```

We use the `[$NAMESPACE-` pattern to filter logs for a specific namespace from the variable `$proxy_upstream_name`, which is formatted as `upstream-<namespace>-<service name>-<service port>` in the default configuration. This works well when namespaces do not contain hyphens. However, if you have similar namespace names like `example` and `example-app`, log entries from `example` might inadvertently match `example-app` due to substring matching.

To avoid this issue, configure a custom log format that explicitly includes the namespace in the log entry:

```yaml
log-format-upstream: '$remote_addr - $remote_user [$time_local] $request $status $body_bytes_sent $http_referer $http_user_agent $request_length $request_time [$proxy_upstream_name] [namespace: $namespace] [$proxy_alternative_upstream_name] $upstream_addr $upstream_response_length $upstream_response_time $upstream_status $req_id'
```

Then set `DEFAULT_LOG_FORMAT` to `false` in the helm values file:

```yaml
settings:
  debug: false
  ingressController:
    label: "app.kubernetes.io/name=ingress-nginx"
    namespace: "ingress-nginx"
    defaultLogFormat: false
```

### RKE2 Helm Configuration

For RKE2 deployments, you can configure `ingress-nginx` logging by applying the following `HelmChartConfig`:

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: rke2-ingress-nginx
  namespace: kube-system
spec:
  valuesContent: |-
    controller:
      config:
        log-format-upstream: '$remote_addr - $remote_user [$time_local] $request $status $body_bytes_sent $http_referer $http_user_agent $request_length $request_time [$proxy_upstream_name] [namespace: $namespace] [$proxy_alternative_upstream_name] $upstream_addr $upstream_response_length $upstream_response_time $upstream_status $req_id'
```

Then use the following values in the `ReflowIngressLogs` helm chart:

```yaml
settings:
  debug: false
  ingressController:
    label: "app.kubernetes.io/name=rke2-ingress-nginx"
    namespace: "kube-system"
    defaultLogFormat: false
```

### Example Configuration

```bash
export DEBUG=true
export KUBECONFIG=/path/to/kubeconfig
export NAMESPACE=app
export INGRESS_NAMESPACE=ingress-nginx
export LABEL_SELECTOR=app.kubernetes.io/name=ingress-nginx
```

---
