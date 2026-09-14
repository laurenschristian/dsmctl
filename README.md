# dsmctl

[![CI](https://github.com/laurenschristian/dsmctl/actions/workflows/ci.yml/badge.svg)](https://github.com/laurenschristian/dsmctl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

One static binary: a CLI and an [MCP](https://modelcontextprotocol.io) server for Synology DiskStation Manager (DSM 7 WebAPI). No agent, no package to install on the NAS; it talks to the DSM web API over HTTP.

```console
dsmctl status                 # model, DSM version, serial, CPU, RAM, uptime, temperature
dsmctl health                 # CPU and memory load, volumes and disk SMART (human-readable)
dsmctl storage                # volumes and physical disks with SMART status
dsmctl update                 # is a DSM update available
dsmctl security               # Security Advisor scan summary
dsmctl network                # interfaces, DNS, gateway
dsmctl ups                    # UPS model, charge, load, runtime
dsmctl processes              # running processes by CPU and memory (alias: top)
dsmctl packages               # installed packages and versions
dsmctl docker                 # containers and their state (alias: ps)
dsmctl tasks                  # scheduled tasks
dsmctl backup                 # Hyper Backup tasks
dsmctl shares                 # shared folders
dsmctl users                  # local users
dsmctl groups                 # local groups
dsmctl connections            # who is connected (alias: who)
dsmctl apis [filter]          # list every DSM WebAPI (for use with raw)
dsmctl reboot --yes           # restart the NAS
dsmctl shutdown --yes         # power off the NAS
dsmctl raw SYNO.Core.System info --version 3   # any DSM API
dsmctl mcp                    # MCP server over stdio
```

Add `--json` to any command for machine-readable output.

## Install

```console
brew install laurenschristian/tap/dsmctl
go install github.com/laurenschristian/dsmctl@latest
```

Or grab a binary from [Releases](https://github.com/laurenschristian/dsmctl/releases).

## Configure

Precedence is flags, then environment (`DSM_URL`, `DSM_USER`, `DSM_PASS`, `DSM_CONFIG`), then the config file
(`~/Library/Application Support/dsmctl/config.yaml` on macOS, `~/.config/dsmctl/config.yaml` on Linux).

```console
dsmctl init --url http://10.0.0.2:5000 --user admin \
  --password-cmd "security find-generic-password -s dsmctl -w"
```

`password_cmd` runs any shell command that prints the password, so the secret can live in a keychain, `pass`, `op read`, `bw get password` or sops. `--password` stores it in plain text with mode 0600 if you prefer.

## Login

DSM sessions are token-based. `dsmctl login` authenticates once and caches the session id in
`~/Library/Application Support/dsmctl/session.json` (mode 0600, kept out of the config file). Every command reuses it and, for accounts without 2-factor authentication, re-logs-in automatically when the session expires.

```console
dsmctl login                  # uses the configured account + password_cmd
dsmctl login --otp 123456     # 2-factor accounts: pass the current code
dsmctl login 123456           # or as a positional; login also prompts if a code is needed
dsmctl logout                 # invalidate and clear the cached session
```

For a 2-factor account, run `dsmctl login --otp <code>` once; the cached session then serves the other commands until it expires, at which point you log in again.

## MCP

`dsmctl mcp` exposes the read operations plus `dsm_raw` as tools: `dsm_status`, `dsm_health`, `dsm_storage`, `dsm_update`, `dsm_security`, `dsm_network`, `dsm_ups`, `dsm_processes`, `dsm_packages`, `dsm_docker`, `dsm_tasks`, `dsm_backup`, `dsm_shares`, `dsm_users`, `dsm_groups`, `dsm_connections`, `dsm_apis`, `dsm_raw`. Reboot and shutdown are intentionally CLI-only.

```console
claude mcp add dsm -- dsmctl mcp
```

Any client that speaks stdio MCP (Cursor, Claude Desktop, Zed) works the same: command `dsmctl`, args `["mcp"]`, and the `DSM_*` env vars or a config file. Ask things like "is any disk unhealthy" and the agent reads storage SMART and answers.

## Coverage

Status, health, storage and SMART, DSM update check, Security Advisor, network, UPS, processes, packages, Docker containers, scheduled and Hyper Backup tasks, shares, users, groups, active connections, and full API discovery (`dsmctl apis`). Reboot and shutdown too. Anything else is `dsmctl raw <api> <method> --version N -p key=value`.

## Development

```console
make hooks    # gofmt, dash check, gitleaks, build, lint on commit; tests + coverage floor on push
make test
make cover    # floor in scripts/coverage.sh
make lint     # golangci-lint, config in .golangci.yml
make sec      # gosec
make docs     # regenerate man/ and docs/cli/
```

See [CONTRIBUTING.md](CONTRIBUTING.md). MIT.
