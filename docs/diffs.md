# Diffs

Reading diffs well is the reason `bt` exists. The CLI has three diff
entry points and all three share the same renderer.

## `bt pr diff <id>`

Show the unified diff for a pull request.

```bash
bt pr diff 142
bt pr diff 142 --stat             # diffstat summary
bt pr diff 142 --no-pager         # stream to stdout
bt pr diff 142 --color=never > 142.patch
patch -p1 < 142.patch
```

## `bt compare <base>..<head>`

Compare two arbitrary refs (branches, tags, commits) without creating
a PR. Follows git semantics:

- `base..head`  — two-dot, literal diff from `base` to `head`.
- `base...head` — three-dot, uses the merge-base (*"what's new on head
  since it diverged from base"*).

```bash
bt compare main..feat/foo         # what feat/foo looks like vs main now
bt compare main...feat/foo        # only what feat/foo added since diverging
bt compare main...HEAD --stat     # diffstat of local branch vs main

bt compare main...feat/foo --commits           # human-readable commit list
bt compare main...feat/foo --commits --json    # machine-readable
```

With no argument, `bt compare` uses the current git branch and its
upstream as `base...head`:

```bash
bt compare                        # ≈ git log --oneline @{u}..HEAD
```

## `--stat`

A `git diff --stat`-style summary with a coloured histogram:

```
internal/kafka/consumer.go  41  +++++++++++++++++++++++++++++++++
internal/kafka/producer.go  17  ++++++++++++++++-
README.md                    3  ++-

 3 file(s) changed, 58 insertion(s)(+), 3 deletion(s)(-)
```

## Colour and pager

The renderer writes plain unified diff bytes when stdout isn't a TTY,
so `bt pr diff | patch`, `| git apply`, `| delta` all work. On a TTY
it:

1. Colours `+`/`-`/`@@`/file headers with ANSI escapes.
2. Pages through the first available of: `$BT_PAGER`, `$PAGER`,
   [`delta`](https://github.com/dandavison/delta), `less -R -F -X`.
3. Disables colour automatically when `NO_COLOR` is set.

Overrides:

| Flag | Effect |
| --- | --- |
| `--color=always` | Force ANSI escapes even when piping. |
| `--color=never` | Plain unified diff (good for `patch`). |
| `--no-pager` | Write straight to stdout. |

## Fancy rendering with `delta`

If you install [`delta`](https://github.com/dandavison/delta), `bt`
picks it up automatically — you get syntax highlighting, word-level
diffs, line numbers, side-by-side view — all for free:

```bash
brew install git-delta
bt pr diff 142           # now rendered by delta
```
