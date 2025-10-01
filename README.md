# Better Stack Operator

Define your Better Stack alerts in Kubernetes resources.

The Better Stack Operator keeps Better Stack monitors and heartbeats in sync with Kubernetes by reconciling `BetterStackMonitor` and `BetterStackHeartbeat` custom resources into real Better Stack resources through the public API.

## Highlights

- **Full monitor coverage** – Configure monitor type, contact routes, SSL/domain expiration, maintenance windows, request headers, Playwright scripts, and more directly from Kubernetes.
- **Heartbeat automation** – Manage Better Stack heartbeats, including grace windows, teams, alert toggles, and maintenance directly from manifests.
- **Safe credential handling** – Secrets referenced via `apiTokenSecretRef` supply the Better Stack API token; the operator never persists tokens elsewhere.
- **Lifecycle management** – Finalizers ensure remote monitors are removed when their CRs are deleted, preventing orphaned resources.
- **Status you can trust** – `Ready`, `CredentialsAvailable`, and `Synced` conditions expose reconciliation health.

## Install with Helm

Published charts live at `oci://ghcr.io/loks0n/betterstack-operator/helm/betterstack-operator` (latest release `0.0.11`).

### 1. Provide Better Stack credentials

Choose how the controller should access the API token:

- **Chart-managed secret** – let the release create the secret in its namespace (default `betterstack-operator`).

  ```bash
  helm upgrade --install betterstack-operator \
    oci://ghcr.io/loks0n/betterstack-operator/helm/betterstack-operator \
    --version 0.0.11 \
    --namespace betterstack-operator --create-namespace \
    --set credentials.secret.create=true \
    --set-file credentials.secret.value=./betterstack-token.txt \
    --wait
  ```

  Swap `--set-file` for `--set credentials.secret.value=$TOKEN` if you prefer piping the token directly from an environment variable or secret store. Add tenant copies with `--set-json credentials.secret.additionalNamespaces='["edge","storefront"]'`.

- **Bring-your-own secret** – pre-create it and point the chart at it:

  ```bash
  kubectl create secret generic betterstack-operator-credentials \
    --from-literal=api-key=REPLACE_ME \
    -n betterstack-operator

  helm upgrade --install betterstack-operator \
    oci://ghcr.io/loks0n/betterstack-operator/helm/betterstack-operator \
    --version 0.0.11 \
    --namespace betterstack-operator --create-namespace \
    --set credentials.existingSecret=betterstack-operator-credentials \
    --wait
  ```

The chart-generated secret defaults to `betterstack-operator-credentials` in the release namespace. Use `credentials.secret.namespace` to move the primary secret and `credentials.secret.additionalNamespaces` to duplicate it; whichever path you choose, ensure the secret exists in every namespace where you create `BetterStackMonitor` objects.

### 2. Create resources

#### Monitors

Apply one of the sample CRs to verify the install:

```bash
kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_https.yaml
kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_keyword.yaml
kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_tcp.yaml
```

Check reconciliation status and debug events with:

```bash
kubectl get betterstackmonitors.monitoring.betterstack.io -A
kubectl describe betterstackmonitor demo-monitor
```

Deleting a `BetterStackMonitor` automatically deletes the remote Better Stack monitor thanks to controller finalizers.

#### Heartbeats

Create a heartbeat and sync it to Better Stack:

```bash
kubectl apply -f config/samples/monitoring_v1alpha1_betterstackheartbeat.yaml
```

Inspect the resource with:

```bash
kubectl get betterstackheartbeats.monitoring.betterstack.io -A
kubectl describe betterstackheartbeat demo-heartbeat
```

Deleting a `BetterStackHeartbeat` tears down the remote heartbeat after the finalizer runs.

### Configuration

See `helm/betterstack-operator/values.yaml` for the full list. Frequently tuned values include:

- `credentials.existingSecret` – reference a pre-created secret instead of letting the chart manage one.
- `credentials.secret.*` – control chart-managed secret creation (name override, key, annotations, inline value).
  Use `credentials.secret.namespace` to move the primary secret and `credentials.secret.additionalNamespaces` to fan it out to tenant namespaces.
- `imagePullSecrets` – add registry credentials when pulling the operator image.
- `podAnnotations`, `podLabels`, `podSecurityContext`, `containerSecurityContext` – attach metadata or adjust pod/container security posture.
- `nodeSelector`, `tolerations`, `affinity` – steer the operator onto matching nodes.
- `namespace` – pin all resources to a specific namespace (defaults to the release namespace).
- `manager.*` – adjust controller ports, enable/disable leader election, and pass extra arguments.
- `rbac.create` – disable default RBAC when running with pre-provisioned roles.
- `crds.install` – set to `false` when CRDs are installed out-of-band (e.g., via GitOps).

## Monitor Spec Reference (excerpt)

| Field | Purpose |
| --- | --- |
| `url` | Endpoint or host to monitor. |
| `monitorType` | `status`, `expected_status_code`, `keyword`, `keyword_absence`, `ping`, `tcp`, `udp`, `smtp`, `pop`, `imap`, `dns`, `playwright`. |
| `teamName` | Target Better Stack team (needed for global API tokens). |
| `checkFrequencyMinutes` | Probe frequency in minutes (converted to seconds for the API). |
| `expectedStatusCodes` | Array of acceptable HTTP status codes. |
| `requiredKeyword` | Required keyword (keyword/UDP monitors). |
| `paused` | Pause monitoring without deleting the monitor. |
| `email`, `sms`, `call`, `push`, `criticalAlert` | Notification channel toggles. |
| `policyID`, `expirationPolicyID`, `monitorGroupID`, `teamWaitSeconds` | Escalation settings. |
| `domainExpirationDays`, `sslExpirationDays` | Alert offsets for domain & SSL expiry. |
| `requestTimeoutSeconds`, `recoveryPeriodSeconds`, `confirmationPeriodSeconds` | Timing controls. |
| `followRedirects`, `verifySSL`, `rememberCookies`, `ipVersion` | HTTP/network behaviour. |
| `maintenanceDays`, `maintenanceFrom`, `maintenanceTo`, `maintenanceTimezone` | Maintenance window definition. |
| `requestHeaders`, `requestBody`, `authUsername`, `authPassword` | HTTP customisation. |
| `environmentVariables`, `playwrightScript`, `scenarioName` | Playwright monitor configuration. |
| `additionalAttributes` | Raw overrides merged into the Better Stack API payload. |

## Heartbeat Spec Reference (excerpt)

| Field | Purpose |
| --- | --- |
| `name` | Human friendly heartbeat name shown in Better Stack. |
| `periodSeconds` | Frequency Better Stack expects check-ins. |
| `graceSeconds` | Extra tolerance window after the period before alerting. |
| `teamName` | Target Better Stack team (needed for global tokens). |
| `call`, `sms`, `email`, `push`, `criticalAlert` | Opt individual notification channels in or out. |
| `teamWaitSeconds` | Delay before escalating to the next team. |
| `heartbeatGroupID` | Link the heartbeat to an existing Better Stack group. |
| `sortIndex` | Reorder heartbeats in the Better Stack UI. |
| `paused` | Pause the heartbeat without deleting it. |
| `maintenanceDays`, `maintenanceFrom`, `maintenanceTo`, `maintenanceTimezone` | Maintenance window definition. |
| `policyID` | Override the default Better Stack alert policy. |
| `baseURL` | Better Stack API base URL override (defaults to `https://uptime.betterstack.com/api/v2`). |
| `apiTokenSecretRef` | Secret reference containing the Better Stack API token (`key` defaults to `api-key`). |

See `api/v1alpha1/betterstackmonitor_types.go` for the full schema and commentary.

## Troubleshooting

- `CredentialsAvailable=False` – confirm the referenced secret exists and contains the API key in the expected key.
- `Synced=False` – the Better Stack API rejected the payload; inspect the condition message for validation errors.
- `Ready=True` – the latest spec was successfully applied.

Enable verbose logging with `--zap-log-level=debug` in the manager deployment for extra context.

## Manual installation (development)

The manifests under `config/` are primarily for hacking on the controller:

```bash
kubectl apply -f config/crd/bases/monitoring.betterstack.io_betterstackmonitors.yaml
kubectl apply -f config/rbac/service_account.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -k config/manager
```

You still need to create a matching secret in the target namespace before running the manager locally or via these raw manifests.

## Development

- Module path: `loks0n/betterstack-operator`.
- API types live under `api/v1alpha1`; controller logic is in `controllers/betterstackmonitor_controller.go`.
- The Better Stack API client lives in `pkg/betterstack`.
- E2E helpers are in `test/e2e`, relying on `kind`, `kubectl`, and a Better Stack test token.

### Testing

- **Unit tests**

  ```bash
  go test ./...
  ```

- **End-to-end (Kind + live Better Stack API)**

  ```bash
  BETTERSTACK_TOKEN=your_token \
    go test -tags=e2e ./test/e2e -run TestBetterStackMonitorLifecycle
  ```

  The e2e test boots a Kind cluster, installs the CRD and controller, applies a richly populated `BetterStackMonitor`, and asserts (via the Better Stack API) that create/update/delete operations are reflected remotely. The test cleans up the remote monitor, but run it only against non-production credentials.

Contributions, issues, and ideas are welcome!
