# cert-manager Webhook for Micetro DNS

This webhook lets cert-manager solve DNS-01 ACME challenges through BlueCat Micetro's REST API. It creates and deletes the TXT records that ACME requires for domain validation.

cert-manager calls the webhook when it needs to prove ownership of a domain. The webhook authenticates with Micetro, finds the correct DNS zone, creates a `_acme-challenge` TXT record, and removes it after validation succeeds.

## Prerequisites

- A Kubernetes cluster with cert-manager installed
- A BlueCat Micetro instance with REST API access (v2)
- Credentials for the Micetro API (username and password, or a session token)

## Installation

Install the webhook with Helm:

```bash
helm install cert-manager-webhook-micetro deploy/cert-manager-webhook-micetro \
  --namespace cert-manager
```

There are also examples of an ArgoCD Application and ApplicationSet in the `deploy/argocd` folder.

## Usage

The webhook reads its configuration from the Issuer or ClusterIssuer resource. You provide the configuration in the `solverConfig` field.

### ClusterIssuer Example

```yaml
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-micetro
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: you@example.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
      - dns01:
          webhook:
            groupName: acme.micetro.io
            solverName: micetro-solver
            config:
              version: "26.1"
              host: "https://micetro.example.com"
              authSecretRef:
                namespace: cert-manager
                name: micetro-credentials
                type: basic
              ttl: 60
              timeout: 30
```

### Issuer Example

```yaml
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt-micetro
  namespace: my-app
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: you@example.com
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
      - dns01:
          webhook:
            groupName: acme.micetro.io
            solverName: micetro-solver
            config:
              version: "26.1"
              host: "https://micetro.example.com"
              authSecretRef:
                namespace: cert-manager
                name: micetro-credentials
                type: basic
              ttl: 60
              timeout: 30
```

### Certificate Example

```yaml
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-example-ca
  namespace: my-app
spec:
  secretName: example-com-tls
  dnsNames:
    - example.com
    - www.example.com
  issuerRef:
    name: letsencrypt-micetro
    kind: Issuer
    group: cert-manager.io
```


### Configuration Fields

| Field | Required | Description |
|---|---|---|
| `version` | Yes | Micetro API version. Valid values: `26.1`, `25.2`, `25.1`. |
| `host` | Yes | Base URL of the Micetro API (for example, `https://micetro.example.com`). |
| `authSecretRef` | Yes | Reference to the Kubernetes Secret that holds the API credentials. |
| `ttl` | No | Time-to-live in seconds for the TXT record. Default: 60. Range: 1 to 3600. |
| `timeout` | No | Timeout in seconds for API requests. Default: 30. Range: 1 to 300. |
| `allowedZones` | No | List of zones the webhook can edit. If empty, all zones are permitted. |
| `dnsViewRef` | No | Name of a DNS view in Micetro. If set, the webhook searches for zones within that view. |
| `headers` | No | Additional HTTP headers to send with API requests. |
| `caBundleRef` | No | Reference to a ConfigMap that holds a custom CA certificate bundle for the Micetro API. |

### Authentication Secret

Create a Kubernetes Secret with the Micetro credentials. The `type` field in `authSecretRef` determines which keys the Secret must contain.

For username and password authentication (`type: basic`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: micetro-credentials
  namespace: cert-manager
type: Opaque
stringData:
  username: your-username
  password: your-password
```

For token authentication (`type: token`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: micetro-credentials
  namespace: cert-manager
type: Opaque
stringData:
  token: your-session-token
```

### Custom CA Bundle

If the Micetro API uses a certificate that your cluster does not trust, create a ConfigMap with the CA bundle:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: micetro-ca-bundle
  namespace: cert-manager
data:
  ca-bundle.crt: |
    -----BEGIN CERTIFICATE-----
    ...
    -----END CERTIFICATE-----
```

Then add the reference to the solver configuration:

```yaml
caBundleRef:
  namespace: cert-manager
  name: micetro-ca-bundle
  key: ca-bundle.crt
```

---

## Development

### Build

Build the container image:

```bash
make build
```

### Test

Run the test suite:

```bash
make test
```

### Lint

Run the Go linter:

```bash
make lint
```

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for the full text.
