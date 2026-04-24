# bt — a `gh`-style CLI for Bitbucket

`bt` is a single static binary that brings the ergonomics of GitHub's `gh`
to Bitbucket. It talks to both **Bitbucket Cloud** (`bitbucket.org`) and
**Bitbucket Data Center** (self-hosted), and is designed to be scripted.

```
$ bt pr list --repo myorg/service
ID   TITLE                            BRANCH           AUTHOR   STATE
142  Upgrade kafka to 3.8             feat/kafka-38    ada      OPEN
141  Revert flaky health probe        fix/health       linus    MERGED

$ bt compare main...feat/kafka-38 --stat
internal/kafka/consumer.go  41  +++++++++++++++++++++++++++++++++
internal/kafka/producer.go  17  ++++++++++++++++-

 2 file(s) changed, 56 insertion(s)(+), 2 deletion(s)(-)
```

## Highlights

- **gh-style UX**: `bt auth login`, `bt repo clone`, `bt pr create`,
  `bt pr checkout`, `bt pr merge`, `bt api /…`.
- **First-class diff rendering** — the reason `bt` exists.
  - `bt pr diff 142` — colourised, paged unified diff.
  - `bt compare main..feat/foo` / `main...feat/foo` — arbitrary
    branch/tag/commit comparisons, no PR required.
  - `--stat` for a diffstat summary, `--commits` for the commit list.
  - Auto-detects TTY, auto-uses `delta` / `less -R` when available,
    honours `$BT_PAGER`, `$PAGER`, and `NO_COLOR`.
- **Cloud + Data Center** from the same binary, via pluggable backends.
- **Scriptable output**: `--json`, `--yaml`, `--template 'Go tmpl'`,
  `--jq 'expr'` on every list/view command.
- **Secure auth**: OS keyring by default (`zalando/go-keyring`), with
  environment-variable fallback (`BT_TOKEN`, `BT_USERNAME`,
  `BT_APP_PASSWORD`, `BT_PAT`) for CI.
- **Multi-context**: switch between `cloud-main`, `dc-internal`, etc.
  with `bt context use`.
- **Escape hatch**: `bt api …` for anything we don't cover yet.

## Install

```
# Homebrew
brew install arogan178/bitbucket-cli/bt

# Scoop (Windows)
scoop bucket add bitbucket-cli https://github.com/arogan178/scoop-bitbucket-cli
scoop install bt

# Go
go install github.com/arogan178/bitbucket-cli/cmd/bt@latest

# Docker
docker run --rm -it ghcr.io/arogan178/bitbucket-cli:latest --help
```

## Quick start

```
bt auth login                       # interactive; stores in keyring
bt context list
bt context use cloud-main

bt repo list --workspace myorg
bt repo clone myorg/service && cd service

bt pr list --state open
bt pr view 142
bt pr diff 142                      # ← the one you came for
bt compare main...HEAD --stat

bt pr create --title "…" --body @./desc.md --base main --head feat/foo
bt pr checkout 142
bt pr merge 142 --strategy squash
```

## Configuration

Config lives at `$XDG_CONFIG_HOME/bt/config.yaml` (or
`~/.config/bt/config.yaml`). Credentials never touch that file — they go
in the OS keyring under service `bt:<context-name>`.

## Output modes

```
bt pr list --json | jq '.[] | select(.state=="OPEN")'
bt pr list --jq '.[] | {id, title}'
bt pr list --template '{{range .}}{{.ID}} {{.Title}}{{"\n"}}{{end}}'
```

## License

Apache-2.0 — see [LICENSE](./LICENSE).
