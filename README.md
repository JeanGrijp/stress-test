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

- Para instalar uma versão específica (recomendada em CI):

```bash
go install github.com/JeanGrijp/stress-test/cmd/stress-test@vX.Y.Z
```

O binário `stress-test` será instalado em `$(go env GOPATH)/bin` (adicione ao PATH).

### Shell completion (opcional)

- zsh (persistente):

```bash
mkdir -p ~/.zsh/completions
stress-test completion zsh > ~/.zsh/completions/_stress-test
echo 'autoload -U compinit; compinit; fpath+=~/.zsh/completions' >> ~/.zshrc
```

- bash (persistente):

```bash
echo 'source <(stress-test completion bash)' >> ~/.bashrc
```

- fish (persistente):

```bash
stress-test completion fish > ~/.config/fish/completions/stress-test.fish
```

- PowerShell (sessão atual):

```powershell
stress-test completion powershell | Out-String | Invoke-Expression
```

## Build from source

```bash
make build
./bin/stress-test version
```

## Usage

Comandos principais:

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

## Dependencies (libs e versões)

Versões conforme `go.mod` atual:

- Go: 1.25
- github.com/spf13/cobra: v1.9.1
- github.com/spf13/pflag: v1.0.6 (indirect)
- github.com/inconshreveable/mousetrap: v1.1.0 (indirect)

## Changelog

### v0.1.0

- Primeira release pública.
- Comandos: run, ramp, curl, version, docs.
- Runner: modos por total de requests, duração e duração+RPS.
- Saída: text ou JSON; gravação em arquivo.
- Ramp: resumo por fase e geral; validações de modos e flags.
- Curl: parser básico de -X/-H/-d/-i/--head/--url com stats opcionais.
- Help: descrições longas e exemplos em todos os comandos.
- Completion: suportado para bash/zsh/fish/powershell.
- Docs: geração via `stress-test docs` e `make docs`.
