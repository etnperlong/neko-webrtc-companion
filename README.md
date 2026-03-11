# Neko TURN Refresh

This repository holds a runnable Go-based TURN credential refresh service. The refresh pipeline now wires the Cloudflare fetcher, Neko rewriter, and Docker restarts, so the scheduler and `/trigger` endpoint execute a meaningful refresh cycle that updates credentials and restarts matching containers.

## Configuration

Runtime configuration now loads in this order:

1. built-in defaults
2. optional YAML config file from `CONFIG_FILE`
3. environment variable overrides

That lets local and container deployments mount a config file while still overriding secrets or per-environment values through env vars.

### YAML config file

Set `CONFIG_FILE` to a YAML file path if you want file-based configuration.

```yaml
cron: "0 */6 * * *"
cloudflare_turn_key_id: "key-id"
cloudflare_api_token: "token"
neko_config_path: "/data/neko-config.yaml"
http_addr: ":8080"
cloudflare_turn_ttl: 86400
run_on_start: true
docker_container_name_glob: "neko-rooms-*"
docker_image_glob: "ghcr.io/example/*"
docker_label_true_key: "managed"
docker_restart_timeout: "30s"
```

If `CONFIG_FILE` is unset, the service behaves like the previous env-only mode.

### Environment variables

All variables are loaded at startup via `config.LoadFromEnv`. The runtime **requires** the values below after the full merge; missing any of them prevents the binary from starting.

- `CONFIG_FILE` — optional path to a YAML config file. Relative paths resolve from the current working directory; absolute paths are recommended in production.

- `NEKO_TURN_CRON` — cron expression that governs when the refresh job runs. The value must be a valid cron spec (e.g. `0 */6 * * *`).
- `CLOUDFLARE_TURN_KEY_ID` — identifier for the Cloudflare TURN key pair that should be rotated.
- `CLOUDFLARE_API_TOKEN` — scoped API token that can read/write TURN resources in the target Cloudflare account.
- `NEKO_CONFIG_PATH` — absolute path inside the container to the Neko YAML file that drives the refresh decisions. For example, `NEKO_CONFIG_PATH=/data/neko-config.yaml`. This file must be mounted as a volume (see below).
- `HTTP_ADDR` — address that the HTTP server listens on. Defaults to `:8080` if unset, so this value is optional unless you need a different port.

A few optional overrides help tune the refresh runtime:

- `CLOUDFLARE_TURN_TTL` (default `86400`) — TTL in seconds for the Cloudflare TURN credential request.
- `RUN_ON_START` (default `true`) — controls whether the scheduler runs a refresh cycle immediately at startup instead of waiting for the first cron tick.
- `DOCKER_CONTAINER_NAME_GLOB` (default `neko-rooms-*`) — glob that selects containers whose names should be restarted. The filter is always applied and respects the AND semantics described below.
- `DOCKER_IMAGE_GLOB` — optional glob that additionally filters containers by their image reference. Matches follow standard glob rules and are only evaluated when this value is set.
- `DOCKER_LABEL_TRUE_KEY` — optional label key that must equal `true` on the container to be included. When this value is empty the label check is skipped; when set it behaves like any other AND filter.
- `DOCKER_RESTART_TIMEOUT` — optional duration (e.g. `30s`) that is passed to the Docker restart timeout. When unset or non-positive the default Docker timeout applies.
- `LOG_FORMAT` (default `text`) — selects `text` or `json` structured logs.
- `LOG_LEVEL` (default `info`) — sets the minimum log level: `debug`, `info`, `warn`, or `error`.
- `LOG_COLOR` (default `true`) — enables colored text logs when `LOG_FORMAT=text`; JSON logs are never colorized.

The service uses Go's `log/slog` package for structured logs. Text mode is optimized for local `docker logs` readability, while JSON mode is intended for log aggregation pipelines.

For Kubernetes, a good pattern is to mount non-secret YAML config through a `ConfigMap`, then override secrets like `CLOUDFLARE_API_TOKEN` through `Secret`-backed environment variables.

## Runtime requirements

The refresh cycle now executes Cloudflare fetches, rewrites the configured Neko YAML, and restarts Docker containers that match the configured filters. Deployments therefore still need:

- **Mounted Docker socket**: The refresh service calls Docker SDK APIs (`internal/docker`), so deployments are expected to mount `/var/run/docker.sock` (read-only is sufficient) to reach the host daemon.
- **Mounted Neko config**: `NEKO_CONFIG_PATH` should point to a mounted Neko YAML file (e.g. `~/.config/neko-config.yaml`). The file is read and potentially rewritten during each refresh cycle, so it must reflect the currently active configuration.

Development has been validated in a Docker-compatible environment; real deployment should still be tested on the target host to ensure the restart behavior meets expectations.

### Docker restart filters

The DOCKER_* environment variables define filters that all participate with AND semantics. Only configured filters are evaluated (empty values are ignored), containers must match every configured glob, and the `DOCKER_LABEL_TRUE_KEY`-named label (when set) must equal `true` to be considered for a restart. This safeguards unrelated workloads from being affected when multiple filters are combined.

## Health & control endpoints

The HTTP server exports three endpoints on `HTTP_ADDR`:

| Endpoint | Method | Description |
| --- | --- | --- |
| `/healthz` | `GET` | Liveness probe; always responds with `200 OK`. |
| `/readyz` | `GET` | Readiness probe; returns `503` before the scheduler is marked ready, `200 OK` afterwards. |
| `/trigger` | `POST` | Triggers `runScheduledJob` on demand. Returns `200 OK` when a refresh completed (`status: ok`) or detected no config changes (`status: noop`), `409 Conflict` when a refresh is already in progress (`status: busy`), and `500 Internal Server Error` when the refresh fails (`status: failed`). |

Use these paths for Kubernetes/Docker health checks while the feature is under development.

Each trigger response is JSON that includes `status`, `message`, `changed`, and `restarted` to help automation understand whether a restart happened or if the request was skipped.

## Docker build & run

1. Build the image: `docker build -t neko-turn-refresh:test .`
2. Run the container by providing the required env vars and mounts, for example:

```bash
docker run --rm \
  --env-file .env \
  -e CONFIG_FILE=/config/config.yaml \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /path/to/config.yaml:/config/config.yaml:ro \
  -v /path/to/neko-config.yaml:/data/neko-config.yaml \
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
    environment:
      CONFIG_FILE: /config/config.yaml
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./config.yaml:/config/config.yaml:ro
      - ./neko-config.yaml:/data/neko-config.yaml
```

Set `NEKO_CONFIG_PATH=/data/neko-config.yaml` in the YAML config or `.env` to match the above. The Neko config mount must be writable because the refresh cycle rewrites the YAML before restarting matching containers. The app config mount can stay read-only because the service only reads it. The compose setup makes it easy to share the socket and config while keeping secrets out of version control.

Local testing can still follow the simple `docker build`/`docker run` workflow above, but the multi-arch release flow described below uses `docker buildx` to publish each architecture and a manifest tag.

## Current status

- `internal/app` wires the HTTP handlers, scheduler, and `refresh.Service`, so both the cron job and `/trigger` endpoint now execute a runnable refresh cycle.
- `internal/http` exposes the health, readiness, and trigger endpoints described above. The trigger handler now writes structured JSON outcomes and enforces the busy/no-op/failure response codes described earlier.

The refresh pipeline is running in this branch, but be sure to validate it on the target host before relying on automated restarts in production.

## Release

The release pipeline lives in `.github/workflows/release-image.yml`. It runs when you push a Git tag, or when you start it manually through `workflow_dispatch`. Manual runs may supply a `tag` input; if they do not, the workflow falls back to `run-<run-number>`. Releases publish to GitHub Container Registry (`ghcr.io/<owner>/<repo>`).

A `prepare-tags` job on a native `ubuntu-24.04` runner lowercases the repository path and validates that the release tag is already Docker-safe. Supported release tags must use lowercase letters, digits, `.`, `_`, or `-`. It then emits per-architecture tags (`-amd64` and `-arm64`) plus the final manifest tag.

The `build-amd64` job also runs on `ubuntu-24.04` and builds the `linux/amd64` image with `docker buildx`, pushing it with the `-amd64` suffix. The `build-arm64` job runs on the native `ubuntu-24.04-arm` runner and pushes the `linux/arm64` image with the `-arm64` suffix. After both succeed, the `publish-manifest` job (back on `ubuntu-24.04`) runs `docker buildx imagetools create` to publish the final multi-arch manifest tag that points at the two digests.

Per-arch tags use the suffixes so they can be inspected individually, and the manifest tag is the same validated value without the suffix. The workflow also runs `docker buildx imagetools inspect` after publishing, and you can repeat `docker buildx imagetools inspect ghcr.io/<owner>/<repo>:<tag>` manually if you want to verify the final manifest tag resolves to both architecture digests.
