# Injecting BS3 secrets into Docker containers

How to get secrets out of a BS3 vault and into containerized apps **at runtime**,
without ever writing them to disk — using the `bs3-sidecar` container.

## The pattern

One init-once **sidecar** per stack fetches the secrets from the vault and writes
one env file per consumer into a shared **tmpfs (RAM) volume**. Each consumer
subpath-mounts **only its own file** (so containers can't read each other's
secrets) and sources it in its entrypoint. The sidecar exits 0; consumers gate on
it with `depends_on`.

```
bs3-sidecar  ──writes──▶  tmpfs volume   ──subpath-mount (ro)──▶  app   (sources app.env)
 (reads vault)            ├─ app.env                              db    (sources db.env)
                          └─ db.env
```

Why a sidecar instead of `--env-file`: your normal flow runs `bs3 writeenv` as a
**pre-deploy** step on the host, then containers consume the file via
`--env-file`. That works because `--env-file` is read by the Docker client *on
the host at deploy time*. The sidecar moves the fetch to *container start on the
deploy host* — no pre-deploy orchestration step — which is handy for
Komodo/compose-only deploys. The tradeoff: once the fetch happens inside a
container, `--env-file` can't see the file (it's parsed before any container
runs), so consumers source the file in their entrypoint instead.

## Quick start

**1. Build the image** (from the repo root — the build needs the sibling
`logger/` module):

```bash
docker build -f Dockerfile.sidecar -t bs3-sidecar:latest .
```

**2. Write a sidecar config** (`sidecar.yml`) mapping each output filename to the
secrets it should contain (refs are `folder.name`; a bare name = root folder):

```yaml
app.env:
  - myapp.MYAPP_DB_URL
  - myapp.MYAPP_API_KEY
db.env:
  - postgres.POSTGRES_PASSWORD
```

**3. Wire up compose** (see `examples/compose.sidecar.yml` for the full file):

```yaml
services:
  bs3-secrets:
    image: bs3-sidecar:latest
    restart: "no"
    volumes:
      - ${HOME}/.config/bs3/bs3.env:/env/bs3.env:ro   # auth (holds BS3_API_TOKEN)
      - ./sidecar.yml:/config/sidecar.yml:ro          # the filename -> [refs] config
      - bs3secrets:/out                               # writer sees the WHOLE volume

  app:
    image: myorg/myapp:latest
    depends_on:
      bs3-secrets:
        condition: service_completed_successfully     # load-bearing (see gotchas)
    volumes:
      - type: volume
        source: bs3secrets
        target: /run/bs3/app.env
        read_only: true
        volume: { subpath: app.env }                  # app sees ONLY app.env
    entrypoint:
      - /bin/sh
      - -c
      - 'set -a; . /run/bs3/app.env; set +a; exec docker-entrypoint.sh "$@"'
      - --
    command: ["myapp"]

volumes:
  bs3secrets:
    driver_opts: { type: tmpfs, device: tmpfs }       # RAM-backed; never hits disk
```

## How `bs3 sidecar` works

```
bs3 sidecar [config_path]
```

1. Reads connection settings from `bs3.env`. In the image this is `/env/bs3.env`
   (via the baked-in `BS3_ENV_FILE`); you can also pass the settings directly as
   `BS3_SERVER_URL` / `BS3_AUTH_METHOD` / `BS3_API_TOKEN` env vars.
2. Reads the YAML config (`/config/sidecar.yml` by default, or `$BS3_SIDECAR_CONFIG`,
   or the positional arg).
3. For each `filename: [refs]` entry, fetches every secret once and writes a
   `KEY=VALUE` env file to `<out>/<filename>` (out = `/out` by default, or
   `$BS3_OUT_DIR`). The key is the secret's name upper-cased
   (`myapp.myapp_db_url` → `MYAPP_DB_URL=...`).
4. Writes each file **atomically** (temp + rename, mode `0600`) so a consumer
   never reads a half-written file, then exits 0.

It's init-once: it writes everything and exits. To pick up rotated secrets,
re-run the stack.

### Image runtime contract

| Mount / var          | Default               | Purpose                                  |
|----------------------|-----------------------|------------------------------------------|
| `/env/bs3.env`       | `BS3_ENV_FILE`        | bind-mount your `bs3.env` (auth)         |
| `/config/sidecar.yml`| `BS3_SIDECAR_CONFIG`  | bind-mount the `filename → [refs]` config|
| `/out`               | `BS3_OUT_DIR`         | mount a shared **tmpfs** volume          |

The `BS3_ENV_FILE` override is what lets you mount `bs3.env` to a clean,
predefined path instead of replicating the host's `~/.config/bs3/...` layout.

## Why per-consumer files + subpath mounts (the isolation point)

If every consumer mounted the *whole* `bs3secrets` volume, splitting secrets into
separate files would be cosmetic — any container could read every file. The
isolation only becomes real because each consumer **subpath-mounts just its own
file**: only that path is projected into the container's mount namespace; the
other files aren't in its filesystem at all. So a compromised `worker` can't read
`db.env`.

The **sidecar itself** legitimately sees every secret — that's its job, and its
blast radius. Keep its footprint minimal and its token scoped (see below).

## Gotchas

- **Subpath needs Docker Engine 26+.** `volume.subpath` landed April 2024. On
  older engines, give each consumer its **own** tmpfs volume and have the sidecar
  write into each (mount all of them in the sidecar, mount one in each consumer).
- **`depends_on: service_completed_successfully` is load-bearing.** If a consumer
  with `subpath: app.env` starts before the file exists, Docker creates the
  subpath as an **empty directory**, and the app then tries to source a directory
  and breaks. Gating consumers on the sidecar completing prevents this.
- **The vault must be unlocked.** BS3 holds the master key in RAM only; after a
  server restart the vault is *locked* until `bs3 openvault`. While locked, the
  sidecar's fetches fail and it exits non-zero — so consumers (gated on its
  success) won't start. Make sure the vault is open before (re)deploying.
- **Trailing newline.** Files are `KEY=VALUE\n` lines and are sourced with
  `set -a; . file`, which handles them correctly. (This is *not* the `_FILE`
  single-raw-value format — don't point a `*_FILE` env var at these files.)
- **Scope the token.** `bs3.env` carries `BS3_API_TOKEN`. Give each stack its own
  token (`bs3 generatetoken <name> [ttl]`) rather than reusing one everywhere, so
  a leak from one stack's sidecar has a bounded blast radius.
- **Never** commit a real `bs3.env`.

## Signals / PID 1

The consumer's entrypoint uses `exec docker-entrypoint.sh "$@"`, so the real app
replaces the shell and **becomes PID 1** — it receives `SIGTERM` from
`docker stop` directly and shuts down gracefully. (Keep the `exec`; without it the
shell stays PID 1 and swallows the signal.) The sidecar writes-and-exits, so its
PID 1 lifetime is trivial.

## Alternative: bind-mount the `bs3` binary (no sidecar image)

If you'd rather not run a sidecar container, you can mount the host's static
`bs3` binary into a consumer and fetch inline. The binary is built
`CGO_ENABLED=0`, so it runs in any Linux image (including Alpine/distroless):

```yaml
volumes:
  - /usr/local/bin/bs3:/usr/local/bin/bs3:ro
  - ${HOME}/.config/bs3/bs3.env:/root/.config/bs3/bs3.env:ro
entrypoint: ["/usr/local/bin/bs3", "envject", "myapp.MYAPP_API_KEY", "--", "docker-entrypoint.sh"]
command: ["myapp"]
```

This is the same model the Komodo periphery containers already use to expose
`bs3`. It avoids building an image but couples the container to a host path and
the host binary's arch/integrity. The sidecar image is the cleaner default for a
multi-service stack.
