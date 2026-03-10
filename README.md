# Neko TURN Refresh

This repository holds the scaffolding for a Go-based TURN credential refresh service. The current runnable pieces are limited to configuration loading, HTTP health/trigger endpoints, and scheduler plumbing; the actual refresh job is still a TODO (see `internal/app/runScheduledJob`).

## Environment variables

All variables are loaded at startup via `config.LoadFromEnv`. The runtime **requires** the values below; missing any of them prevents the binary from starting.

- `NEKO_TURN_CRON` — cron expression that will govern when the (future) refresh job runs. The value must be a valid cron spec (e.g. `0 */6 * * *`).
- `CLOUDFLARE_TURN_KEY_ID` — identifier for the Cloudflare TURN key pair that should be rotated.
- `CLOUDFLARE_API_TOKEN` — scoped API token that can read/write TURN resources in the target Cloudflare account.
- `NEKO_CONFIG_PATH` — absolute path inside the container to the Neko YAML file that drives the refresh decisions. For example, `NEKO_CONFIG_PATH=/data/neko-config.yaml`. This file must be mounted as a volume (see below).
- `HTTP_ADDR` — address that the HTTP server listens on. Defaults to `:8080` if unset, so this value is optional unless you need a different port.

## Runtime requirements

- **Mounted Docker socket**: The eventual refresh implementation will call Docker SDK APIs (`internal/docker`), so deployments are expected to mount `/var/run/docker.sock` (read-only is sufficient) to reach the host daemon. The current scheduler/trigger scaffold does not yet exercise Docker calls, but the mount is documented now to stay aligned with the intended production setup.
- **Mounted Neko config**: Likewise, `NEKO_CONFIG_PATH` should point to a mounted Neko YAML file (e.g. `~/.config/neko-config.yaml`) in preparation for the refresh job wiring. The path does not affect the present stubbed run, but the volume will be necessary once the refresh logic acts on that configuration.

## Health & control endpoints

The HTTP server exports three endpoints on `HTTP_ADDR`:

| Endpoint | Method | Description |
| --- | --- | --- |
| `/healthz` | `GET` | Liveness probe; always responds with `200 OK`. |
| `/readyz` | `GET` | Readiness probe; returns `503` before the scheduler is marked ready, `200 OK` afterwards. |
| `/trigger` | `POST` | Triggers `runScheduledJob` on demand. Currently the job body is a stub (`internal/app` keeps a TODO) so this endpoint returns `200 OK` but no Cloudflare/Docker work happens yet. |

Use these paths for Kubernetes/Docker health checks while the feature is under development.

## Docker build & run

1. Build the image: `docker build -t neko-turn-refresh:test .`
2. Run the container by providing the required env vars and mounts, for example:

```bash
docker run --rm \
  --env-file .env \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /path/to/neko-config.yaml:/data/neko-config.yaml:ro \
  -p 8080:8080 \
  neko-turn-refresh:test
```

Create `.env` from `.env.example` (copy and fill the placeholder values) before running the container if you prefer not to inline secrets on the command line.

### Docker Compose snippet

```yaml
version: "3.9"
services:
  neko-turn-refresh:
    image: neko-turn-refresh:test
    env_file: .env
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./neko-config.yaml:/data/neko-config.yaml:ro
```

Set `NEKO_CONFIG_PATH=/data/neko-config.yaml` in `.env` to match the above. The compose setup makes it easy to share the socket and config while keeping secrets out of version control.

## Current status

- `internal/app` wires the HTTP handlers and scheduler but the refresh job body (`runScheduledJob`) is currently a no-op placeholder.
- `internal/http` exposes the health, readiness, and trigger endpoints described above. The trigger handler calls the placeholder job so it always succeeds even though no Cloudflare or Docker work happens yet.

Once the refresh service is implemented, the scheduler will run it on the configured cron expression and the trigger endpoint will provide a manual override.
