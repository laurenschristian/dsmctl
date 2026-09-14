# dsmctl

[![CI](https://github.com/laurenschristian/dsmctl/actions/workflows/ci.yml/badge.svg)](https://github.com/laurenschristian/dsmctl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI and MCP server for Synology DSM One static binary: a CLI and an [MCP](https://modelcontextprotocol.io) server for Synology DiskStation Manager (DSM 7 WebAPI).

> Status: scaffold. The client, commands and MCP tools are being built. `dsmctl doctor` and `dsmctl mcp` work today.

## Install

```console
brew install laurenschristian/tap/dsmctl
go install github.com/laurenschristian/dsmctl@latest
```

## Configure

Precedence is flags, then environment (`DSM_URL`, `DSM_USER`, `DSM_PASS`, `DSM_CONFIG`), then the config file
(`~/Library/Application Support/dsmctl/config.yaml` on macOS, `~/.config/dsmctl/config.yaml` on Linux).
`password_cmd` runs any command that prints the secret, so it can live in a keychain, `op read`, `pass` or sops.

## MCP

```console
claude mcp add dsm -- dsmctl mcp
```

## Development

```console
make hooks    # gofmt, dash check, gitleaks, build, lint on commit; tests + coverage floor on push
make test
make lint
make docs     # regenerate man/ and docs/cli/
```

See [CONTRIBUTING.md](CONTRIBUTING.md). MIT.
