# stress-test

[![Release](https://img.shields.io/github/v/release/JeanGrijp/stress-test?sort=semver)](https://github.com/JeanGrijp/stress-test/releases)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](https://go.dev/dl/)
[![Release CI](https://img.shields.io/github/actions/workflow/status/JeanGrijp/stress-test/release.yml?branch=main&label=release)](https://github.com/JeanGrijp/stress-test/actions/workflows/release.yml)

CLI to run and orchestrate HTTP load/stress tests.

## Install

Using Go (Go 1.25+):

```bash
go install github.com/JeanGrijp/stress-test/cmd/stress-test@latest
```

- To install a specific version (recommended for CI):

```bash
go install github.com/JeanGrijp/stress-test/cmd/stress-test@vX.Y.Z
```

The `stress-test` binary will be installed into `$(go env GOPATH)/bin` (ensure it is on your PATH).

### Shell completion (optional)

- zsh (persistent):

```bash
mkdir -p ~/.zsh/completions
stress-test completion zsh > ~/.zsh/completions/_stress-test
echo 'autoload -U compinit; compinit; fpath+=~/.zsh/completions' >> ~/.zshrc
```

- bash (persistent):

```bash
echo 'source <(stress-test completion bash)' >> ~/.bashrc
```

- fish (persistent):

```bash
stress-test completion fish > ~/.config/fish/completions/stress-test.fish
```

- PowerShell (current session):

```powershell
stress-test completion powershell | Out-String | Invoke-Expression
```

## Build from source

```bash
make build
./bin/stress-test version
```

## Usage

Main commands:

```bash
stress-test --help
stress-test run --url https://example.com --requests 100 --concurrency 10
stress-test ramp --url https://example.com --steps 3 --start-concurrency 5 --step-concurrency 5 --requests-per-step 200
stress-test curl -i https://httpbin.org/get
stress-test docs --format markdown --out-dir ./docs/cli
```

## Development

Helpful targets:

```bash
make build     # build local binary with version metadata
make install   # install into GOPATH/bin with version metadata
make clean     # remove build artifacts
make docs      # generate Markdown docs under docs/cli
```

## Releases and Homebrew (optional)

This repo includes GoReleaser and a GitHub Actions workflow that builds multi-OS binaries on tags starting with `v`.

How to cut a public release:

1) Update changelog/README as needed.
2) Create and push a tag:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

This triggers the `release` workflow, which:

- Builds binaries for Linux/macOS/Windows
- Embeds version/commit/date via ldflags
- Publishes a GitHub Release
- Attaches an additional asset `cli-docs.zip` with the CLI docs (Markdown)

After the release is published, users can install exactly that version via:

```bash
go install github.com/JeanGrijp/stress-test/cmd/stress-test@vX.Y.Z
```

Homebrew (optional): the `.goreleaser.yaml` has a tap config. To enable, set `skip_upload: false` and configure a tap repo and token.

## Dependencies (libs and versions)

Versions according to the current `go.mod`:

- Go: 1.25
- github.com/spf13/cobra: v1.9.1
- github.com/spf13/pflag: v1.0.6 (indirect)
- github.com/inconshreveable/mousetrap: v1.1.0 (indirect)

## Changelog

### v0.1.0

- First public release.
- Commands: run, ramp, curl, version, docs.
- Runner: modes by total requests, by duration, and duration+RPS.
- Output: text or JSON; optional write to file.
- Ramp: per-phase and overall summaries; mode/flag validations.
- Curl: basic parser for -X/-H/-d/-i/--head/--url with optional stats.
- Help: long descriptions and examples for all commands.
- Completion: bash/zsh/fish/powershell supported.
- Docs: generation via `stress-test docs` and `make docs`.
