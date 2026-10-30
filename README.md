# Custom Caddy Build

This repository builds a reproducible Caddy binary. It intentionally contains
no deployment configuration, systemd units, runtime certificates, or compiled
artifacts.

## Included modules

- Caddy v2.11.7 standard modules
- `github.com/TnZzZHlp/caddy/dynamic_proxy` (local source)
- `github.com/caddyserver/replace-response`
- `github.com/mholt/caddy-webdav`

## Build

```bash
make test
make build
```

The binary is written to `build/caddy` and is intentionally ignored by Git.

Verify the custom modules with:

```bash
./build/caddy list-modules --skip-standard
```

Runtime configuration and deployment are managed outside this repository.
