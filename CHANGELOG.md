# Changelog

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.2]

### Added
- Confirmation-gated write actions: `docker start|stop|restart <name>`, `tasks run <id>`, `packages start|stop <id>` (all require `--yes`). CLI-only, not exposed as MCP tools.

## [0.1.1]

### Fixed
- Homebrew cask now strips the macOS quarantine attribute on install, so the binary runs without a Gatekeeper block.
- Correct cask description (was a leftover from another tool).

## [0.1.0]

### Added
- DSM 7 client with token login, cached session, 2-factor support (`--otp`, positional, or interactive prompt), and auto re-login on expiry.
- Read commands: `status`, `health` (human-readable sizes and percentages), `storage` (disk SMART), `update`, `security`, `network`, `ups`, `processes`/`top`, `packages`, `docker`/`ps`, `tasks`, `backup`, `shares`, `users`, `groups`, `connections`/`who`, `apis`.
- Control: `reboot` and `shutdown` (confirmation-gated), `raw` for any `SYNO.*` API.
- MCP server (`dsmctl mcp`) exposing 18 read tools plus `dsm_raw`.
- `init`, `doctor`, shell completions, man pages, and per-command markdown docs.
