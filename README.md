# Better Stack Operator

The Better Stack Operator keeps Better Stack monitors in sync with Kubernetes by reconciling `BetterStackMonitor` custom resources into real monitors through the public Better Stack API.

## Highlights

- **Full monitor coverage** – Configure monitor type, contact routes, SSL/domain expiration, maintenance windows, request headers, Playwright scripts, and more directly from Kubernetes.
- **Safe credential handling** – Secrets referenced via `apiTokenSecretRef` supply the Better Stack API token; the operator never persists tokens elsewhere.
- **Lifecycle management** – Finalizers ensure remote monitors are removed when their CRs are deleted, preventing orphaned resources.
- **Status you can trust** – `Ready`, `CredentialsAvailable`, and `Synced` conditions expose reconciliation health.

## Helm Install

Once a release is tagged (`vX.Y.Z`), the publish workflow builds and pushes a chart to:

```
oci://ghcr.io/loks0n/betterstack-operator/helm/betterstack-operator
```

Install/update the operator with:

```bash
helm upgrade --install betterstack-operator \
  oci://ghcr.io/loks0n/betterstack-operator/helm/betterstack-operator \
  --namespace betterstack-operator --create-namespace \
  --wait
```

Create the `betterstack-credentials` secret before running the chart (see Quick Start step 2).

## Quick Start

1. **Fetch dependencies and (optionally) build**

   ```bash
   go mod tidy
   go build ./...
   ```

2. **Create an API token secret**

   ```bash
   kubectl create secret generic betterstack-credentials \
     --from-literal=api-key=REPLACE_ME \
     -n default
   ```

3. **Install CRD, RBAC, and controller**

   ```bash
   kubectl apply -f config/crd/bases/monitoring.betterstack.io_betterstackmonitors.yaml
   kubectl apply -f config/rbac/service_account.yaml
   kubectl apply -f config/rbac/role.yaml
   kubectl apply -f config/rbac/role_binding.yaml
   kubectl apply -f config/manager/manager.yaml
   ```

4. **Create a monitor resource**

   Choose one (or more) of the sample manifests:

   ```bash
   kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_https.yaml
   kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_keyword.yaml
   kubectl apply -f config/samples/monitoring_v1alpha1_betterstackmonitor_tcp.yaml
   ```

5. **Inspect status**

   ```bash
   kubectl get betterstackmonitors.monitoring.betterstack.io -A
   kubectl describe betterstackmonitor demo-monitor
   ```

Deleting the `BetterStackMonitor` automatically deletes the remote Better Stack monitor.

## Spec Reference (excerpt)

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

See `api/v1alpha1/betterstackmonitor_types.go` for the full schema and commentary.

## Troubleshooting

- `CredentialsAvailable=False` – Confirm the referenced secret exists and contains the API key.
- `Synced=False` – The Better Stack API rejected the payload; inspect the condition message for validation errors.
- `Ready=True` – The latest spec was successfully applied.

Enable verbose logging with `--zap-log-level=debug` in the manager deployment for extra context.

## Testing

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

## Development Notes

- Module path: `loks0n/betterstack-operator`.
- API types live under `api/v1alpha1`; controller logic is in `controllers/betterstackmonitor_controller.go`.
- The Better Stack API client (create/update/get/list/delete) resides in `pkg/betterstack`.
- E2E helpers are in `test/e2e`, relying on `kind`, `kubectl`, and genuine Better Stack credentials.

Contributions, issues, and ideas are welcome!
