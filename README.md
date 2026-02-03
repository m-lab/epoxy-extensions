# epoxy-extensions

An HTTP server that provides extension endpoints for the [M-Lab ePoxy](https://github.com/m-lab/epoxy) boot system. It allows machines booting via ePoxy to perform cluster management operations such as obtaining Kubernetes bootstrap tokens, storing BMC credentials, and removing nodes from the cluster.

## Prerequisites

- Go 1.19+
- `kubeadm` - for creating Kubernetes bootstrap tokens
- `kubectl` - for node management operations
- Google Cloud credentials (for BMC password storage in Datastore)

## Build

```bash
go build -o epoxy-extensions .
```

## Run

```bash
./epoxy-extensions -listen-address=:8800 -bin-dir=/usr/bin
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-listen-address` | `:8800` | Address on which to listen for requests |
| `-bin-dir` | `/usr/bin` | Absolute path to directory containing `kubeadm` and `kubectl` binaries |

## API Endpoints

All extension endpoints require POST requests with an ePoxy extension request body. Requests are rejected if the machine's last boot time exceeds 120 minutes.

### Token Allocation

**`POST /v1/allocate_k8s_token`**

Creates a Kubernetes bootstrap token for the requesting machine.

- Response: `text/plain` - the bootstrap token

**`POST /v2/allocate_k8s_token`**

Creates a Kubernetes bootstrap token with full join details.

- Response: `application/json`
```json
{
  "api_address": "api.example.com:6443",
  "token": "abcdef.0123456789abcdef",
  "ca_hash": "sha256:..."
}
```

### BMC Password Storage

**`POST /v1/bmc_store_password`**

Stores a BMC (iDRAC) password in Google Cloud Datastore. The password is passed in the `p` query parameter of the extension request's `RawQuery` field.

- Response: `200 OK` on success (no body)

### Node Management

**`POST /v1/node/delete`**

Deletes the requesting machine's node from the Kubernetes cluster. Useful for managed instance group (MIG) instances that need to cleanly leave the cluster before termination.

- Response: `200 OK` on success (no body)

### Utility Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Health check, returns "ePoxy Extensions" |
| `/metrics` | GET | Prometheus metrics |

## Metrics

The server exposes Prometheus histograms for request duration:

- `allocate_k8s_token_request_duration_seconds`
- `bmc_store_password_request_duration_seconds`
- `node_request_duration_seconds`

## Testing

```bash
go test ./...
```
