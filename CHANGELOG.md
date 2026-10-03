# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0.0, minor versions may include breaking changes to the native API; they are called out under **Changed** with a migration note.

Provider-compatible adapters follow the real providers' behavior. Fixes that make an adapter match its provider more closely are listed under **Fixed** with the `fidelity:` prefix.

## [Unreleased]

### Added

- `phone` package: E.164 parsing with `valid`/`possible`/`off` modes (using `nyaruka/phonenumbers`), GSM-7 vs UCS-2 encoding detection, segment counting (160/153 for GSM-7, 70/67 for UCS-2, max 10 segments)
- Repo scaffolding for M1: `go.mod`, `Taskfile.yml`, `.golangci.yml` (with `depguard` rules), `biome.json`, `.gitattributes`, `.editorconfig`, `.air.toml`, Apache-2.0 `LICENSE`, README stub, CI workflow (lint + test matrix on Linux/macOS/Windows)
- Design spec for mocksms (`docs/superpowers/specs/2026-10-03-mocksms-design.md`).
- Project documentation: overview, PRD, architecture, tech stack, engineering standards, tasks, progress, and agent instructions (`AGENTS.md`).
