# bt

`bt` is a single static binary that brings GitHub's `gh` ergonomics to
Bitbucket Cloud and Data Center. It is designed to be:

- **Scriptable** — every list/view command supports `--json`, `--yaml`,
  `--template`, and `--jq`.
- **Readable** — colourised unified diffs, diffstat summaries, and
  pager passthrough.
- **Portable** — single static binary, no runtime.
- **Safe** — credentials live in the OS keyring, never in config files.

Start with the [quick start](quickstart.md) or jump straight to the
[diffs guide](diffs.md) — the flagship feature.
